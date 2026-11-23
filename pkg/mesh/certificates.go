package mesh

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"

	httpmuxer "github.com/traefik/traefik/v3/pkg/muxer/http"
	tcpmuxer "github.com/traefik/traefik/v3/pkg/muxer/tcp"
	traefiktls "github.com/traefik/traefik/v3/pkg/tls"
	"github.com/traefik/traefik/v3/pkg/types"
)

const (
	pathCertificates = "/v1/certificates"

	// certificatesInterval is how often a satellite asks for the certificates of its routers: a renewal reaches it then.
	defaultCertificatesInterval = 30 * time.Second

	// maxCertificatesBody bounds what a satellite reads of the certificates of an instance.
	maxCertificatesBody = 16 << 20
)

// Certificate is a certificate of a resolver of an instance, as a satellite receives it.
type Certificate struct {
	Resolver string `json:"resolver"`
	// Domains are the main domain, then the alternative names.
	Domains []string `json:"domains"`
	// Certificate is the PEM chain, and Key the PEM private key.
	Certificate string    `json:"certificate"`
	Key         string    `json:"key"`
	NotAfter    time.Time `json:"notAfter"`
}

// CertificateSource returns the certificates of a resolver of this instance.
type CertificateSource func(resolver string) ([]Certificate, error)

// SetCertificateSource gives the mesh the certificates of the resolvers of this instance, for the satellites.
func (m *Mesh) SetCertificateSource(source CertificateSource) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.certificateSource = source
}

type certificatesAnswer struct {
	Certificates []Certificate `json:"certificates"`
}

// serveCertificates answers a satellite with the certificates of the domains of the routers it announces, from the
// resolvers they name. A satellite gets nothing else: the routers it announces are the ones this instance follows.
func (m *Mesh) serveCertificates(rw http.ResponseWriter, req *http.Request) {
	peer := peerFrom(req.Context())
	m.seen(peer)

	m.mu.Lock()
	source := m.certificateSource
	table := m.imported[peer]
	m.mu.Unlock()

	answer := certificatesAnswer{Certificates: []Certificate{}}

	if source != nil && table != nil {
		for resolver, domains := range wantedCertificates(table.routes) {
			certificates, err := source(resolver)
			if err != nil {
				m.logger.Debug().Err(err).Str("resolver", resolver).Msg("Unable to read the certificates of a resolver for a satellite")
				continue
			}

			for _, certificate := range certificates {
				if coversAny(certificate.Domains, domains) {
					certificate.Resolver = resolver
					answer.Certificates = append(answer.Certificates, certificate)
				}
			}
		}
	}

	slices.SortFunc(answer.Certificates, func(a, b Certificate) int { return strings.Compare(certificateKey(a), certificateKey(b)) })

	body, err := json.Marshal(answer)
	if err != nil {
		http.Error(rw, "unable to encode the certificates", http.StatusInternalServerError)
		return
	}

	sum := sha256.Sum256(body)
	etag := `"` + hex.EncodeToString(sum[:12]) + `"`

	rw.Header().Set("ETag", etag)

	if req.Header.Get("If-None-Match") == etag {
		rw.WriteHeader(http.StatusNotModified)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	_, _ = rw.Write(body)
}

// wantedCertificates returns, by resolver, the domains of the routers that name a certificate resolver:
// the domains of their TLS configuration, or the ones of their rule.
func wantedCertificates(routes []Route) map[string][]string {
	wanted := map[string][]string{}

	for _, route := range routes {
		if route.TLS == nil || route.TLS.CertResolver == "" || route.TLS.Passthrough {
			continue
		}

		var domains []string

		for _, domain := range route.TLS.Domains {
			domains = append(domains, domain.ToStrArray()...)
		}

		if len(domains) == 0 {
			var err error

			switch route.Kind {
			case KindHTTP:
				domains, err = httpmuxer.ParseDomains(route.Rule)
			case KindTCP:
				domains, err = tcpmuxer.ParseHostSNI(route.Rule)
			}

			if err != nil {
				continue
			}
		}

		for _, domain := range domains {
			domain = strings.ToLower(strings.TrimSpace(domain))
			if domain != "" && domain != "*" && !slices.Contains(wanted[route.TLS.CertResolver], domain) {
				wanted[route.TLS.CertResolver] = append(wanted[route.TLS.CertResolver], domain)
			}
		}
	}

	return wanted
}

// coversAny tells whether a certificate with these names serves one of the domains.
func coversAny(names, domains []string) bool {
	for _, domain := range domains {
		for _, name := range names {
			if covers(strings.ToLower(name), domain) {
				return true
			}
		}
	}

	return false
}

// covers tells whether a name of a certificate serves a domain: the same name, or a wildcard of one label.
func covers(name, domain string) bool {
	if name == domain {
		return true
	}

	suffix, wildcard := strings.CutPrefix(name, "*.")
	if !wildcard || strings.HasPrefix(domain, "*.") {
		return false
	}

	label, rest, ok := strings.Cut(domain, ".")

	return ok && label != "" && rest == suffix
}

func certificateKey(c Certificate) string {
	return c.Resolver + "\x00" + strings.Join(c.Domains, ",")
}

// certificatesLoop asks an instance, now and then, for the certificates of the routers this satellite announces,
// until the context is done.
// A router that is new to the instance gets its certificate within seconds, when the CA answers: the satellite asks
// after 1 second, then twice as late each time, up to the interval, and starts over when its routers change.
func (l *link) certificatesLoop(ctx context.Context, nodeID string) {
	etag := ""
	first := min(time.Second, l.mesh.certificatesInterval)
	wait := first

	for ctx.Err() == nil {
		l.mesh.mu.Lock()
		changed := l.mesh.exportChanged
		l.mesh.mu.Unlock()

		if s := l.current(); s != nil {
			next, err := l.mesh.askCertificates(ctx, s, nodeID, etag)
			if err != nil && ctx.Err() == nil {
				l.logger.Debug().Err(err).Str("peer", nodeID).Msg("Unable to ask an instance of the mesh for the certificates of this satellite")
			} else if err == nil {
				etag = next
			}
		}

		select {
		case <-time.After(wait):
			wait = min(2*wait, l.mesh.certificatesInterval)
		case <-changed:
			wait = first
		case <-ctx.Done():
		}
	}
}

// askCertificates asks the other end of a session for the certificates of this satellite, unless they did not change,
// and records them. It returns the version of the answer.
func (m *Mesh) askCertificates(ctx context.Context, s *session, nodeID, etag string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, m.linkTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+controlHost+pathCertificates, nil)
	if err != nil {
		return etag, err
	}

	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := s.RoundTrip(req)
	if err != nil {
		return etag, err
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusNotModified:
		return etag, nil
	case http.StatusOK:
	default:
		return etag, fmt.Errorf("unexpected answer to a request for the certificates of a satellite: %s", resp.Status)
	}

	var answer certificatesAnswer
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxCertificatesBody)).Decode(&answer); err != nil {
		return etag, fmt.Errorf("reading the certificates of an instance: %w", err)
	}

	m.setReceivedCertificates(nodeID, answer.Certificates)

	return resp.Header.Get("ETag"), nil
}

// setReceivedCertificates records the certificates an instance gave this satellite, and gives them to the routing
// when the ones in use change. They are kept when the instance goes away: the satellite serves them until they expire.
func (m *Mesh) setReceivedCertificates(nodeID string, certificates []Certificate) {
	m.mu.Lock()
	before := m.certificatesInUseLocked()
	m.received[nodeID] = certificates
	after := m.certificatesInUseLocked()
	m.mu.Unlock()

	if !slices.EqualFunc(before, after, func(a, b Certificate) bool { return a.Certificate == b.Certificate && a.Key == b.Key }) {
		m.logger.Info().Int("certificates", len(after)).Str("peer", nodeID).Msg("This satellite serves new certificates from an instance of the mesh")
		m.publish()
	}
}

// certificatesInUseLocked returns the certificates this satellite serves: for each set of domains of a resolver, the one
// that expires last, among the ones of all the instances, that has not expired yet.
func (m *Mesh) certificatesInUseLocked() []Certificate {
	now := m.now()

	latest := map[string]Certificate{}

	for _, certificates := range m.received {
		for _, certificate := range certificates {
			if !certificate.NotAfter.IsZero() && certificate.NotAfter.Before(now) {
				continue
			}

			key := certificateKey(certificate)
			if current, ok := latest[key]; !ok || certificate.NotAfter.After(current.NotAfter) {
				latest[key] = certificate
			}
		}
	}

	inUse := make([]Certificate, 0, len(latest))
	for _, key := range slices.Sorted(maps.Keys(latest)) {
		inUse = append(inUse, latest[key])
	}

	return inUse
}

// tlsCertificates returns the certificates this satellite serves, for the routing configuration.
func (m *Mesh) tlsCertificates() []*traefiktls.CertAndStores {
	m.mu.Lock()
	inUse := m.certificatesInUseLocked()
	m.mu.Unlock()

	certificates := make([]*traefiktls.CertAndStores, 0, len(inUse))

	for _, certificate := range inUse {
		served := &traefiktls.CertAndStores{Stores: []string{traefiktls.DefaultTLSStoreName}}
		served.CertFile = types.FileOrContent(certificate.Certificate)
		served.KeyFile = types.FileOrContent(certificate.Key)

		certificates = append(certificates, served)
	}

	return certificates
}
