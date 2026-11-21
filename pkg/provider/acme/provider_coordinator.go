package acme

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"

	"github.com/go-acme/lego/v5/certificate"
	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/types"
)

// storeCoordinated is a store that several instances share, which have to agree on the one that asks the ACME server.
type storeCoordinated interface {
	coordinatorFor(resolverName string) *coordinator
}

// coordinator returns the coordinator of the resolver, nil when its store is not shared with other instances.
func (p *Provider) coordinator() *coordinator {
	if coordinated, ok := p.Store.(storeCoordinated); ok {
		return coordinated.coordinatorFor(p.ResolverName)
	}

	return nil
}

// resolveCertificate obtains a certificate for the domains that have none.
// With a store shared by several instances, it only does so when this instance is the one elected for the certificate:
// otherwise it returns nothing, and the certificate comes from the store once the elected instance has it.
func (p *Provider) resolveCertificate(ctx context.Context, domain types.Domain, tlsStore string) (types.Domain, *certificate.Resource, error) {
	coord := p.coordinator()
	if coord == nil {
		return p.obtainCertificate(ctx, domain, tlsStore)
	}

	domains, err := p.sanitizeDomains(ctx, domain)
	if err != nil {
		return types.Domain{}, nil, err
	}

	id := certID(domains)

	lock, err := p.elect(ctx, coord, id, domains, tlsStore)
	if lock == nil {
		return types.Domain{}, nil, err
	}

	obtain := p.obtainCertificate
	if coord.obtain != nil {
		obtain = coord.obtain
	}

	obtainCtx, stop := coord.hold(ctx, lock)
	dom, cert, err := obtain(obtainCtx, domain, tlsStore)
	stop()

	countIssuance(p.ResolverName, issuanceObtain, cert != nil, err)

	p.elected(ctx, coord, id, lock, cert != nil, err)

	return dom, cert, err
}

// resolveDefaultCertificate obtains the default certificate of a TLS store, under the same election as resolveCertificate.
func (p *Provider) resolveDefaultCertificate(ctx context.Context, domains []string) (*certificate.Resource, error) {
	coord := p.coordinator()
	if coord == nil {
		return p.obtainDefaultCertificate(ctx, domains)
	}

	id := certID(domains)

	// The caller checked that no certificate has exactly these domains, which is what the default certificate needs.
	lock, err := p.electFor(ctx, coord, id, domains, "", func() bool { return p.certExists(domains) })
	if lock == nil {
		return nil, err
	}

	obtainCtx, stop := coord.hold(ctx, lock)
	cert, err := p.obtainDefaultCertificate(obtainCtx, domains)
	stop()

	countIssuance(p.ResolverName, issuanceObtain, cert != nil, err)

	p.elected(ctx, coord, id, lock, cert != nil, err)

	return cert, err
}

// elect returns the lock of a certificate request when this instance is the one to ask the ACME server.
func (p *Provider) elect(ctx context.Context, coord *coordinator, id string, domains []string, tlsStore string) (*cluster.Lock, error) {
	return p.electFor(ctx, coord, id, domains, tlsStore, func() bool {
		return len(p.uncoveredDomains(domains, tlsStore)) == 0
	})
}

func (p *Provider) electFor(ctx context.Context, coord *coordinator, id string, domains []string, tlsStore string, served func() bool) (*cluster.Lock, error) {
	logger := log.Ctx(ctx)

	if served() {
		coord.forget(ctx, id)
		return nil, nil
	}

	// The other instances learn about the certificate: this one may not be allowed to obtain it, or not be the best one to.
	if ctx.Value(coordinatorAttemptKey{}) == nil {
		coord.request(ctx, id, domains, tlsStore)
	}

	// Without the server, no instance can be elected: the request stays pending, without a wait on each call to the server.
	if err := coord.store.Ready(p.ResolverName); err != nil {
		logger.Debug().Err(err).Strs("domains", domains).Msg("The certificate is requested once the cluster Redis server is back")
		return nil, nil
	}

	lock, err := coord.acquire(ctx, id)
	if err != nil {
		// The request stays pending, and is tried again when the cluster answers.
		logger.Debug().Err(err).Strs("domains", domains).Msg("Unable to take the certificate request in the cluster")
		return nil, nil
	}

	if lock == nil {
		logger.Debug().Strs("domains", domains).Msg("The certificate is not requested by this instance for now")
		return nil, nil
	}

	// Another instance may have stored the certificate while this one waited for its turn.
	coord.store.refresh(p.ResolverName)

	if served() {
		coord.forget(ctx, id)
		_ = lock.Release(ctx)

		return nil, nil
	}

	logger.Info().Strs("domains", domains).Msg("This instance requests the certificate for the cluster")

	return lock, nil
}

// elected closes the request of an elected instance.
func (p *Provider) elected(ctx context.Context, coord *coordinator, id string, lock *cluster.Lock, obtained bool, err error) {
	switch {
	case err != nil:
		coord.failed(ctx, id, err, true)
		_ = lock.Release(ctx)
	case !obtained:
		// Nothing was asked to the ACME server, e.g. the domains were being resolved by another routine of this instance.
		_ = lock.Release(ctx)
	default:
		// The lock is left to expire: the caller stores the certificate right after this,
		// and no instance must take the request in between.
		coord.succeeded(ctx, id)
	}
}

// electRenewals returns the certificates that this instance is elected to renew: the ones it found due and took,
// and the ones it took from the announcement of an instance with a lower priority.
// The returned function must be called once the renewals are done.
func (p *Provider) electRenewals(ctx context.Context, certificates []*CertAndStore) ([]*CertAndStore, func()) {
	coord := p.coordinator()
	if coord == nil {
		return certificates, func() {}
	}

	if coord.member.Priority() == 0 {
		return nil, func() {}
	}

	logger := log.Ctx(ctx)

	if err := coord.store.Ready(p.ResolverName); err != nil {
		if len(certificates) > 0 {
			logger.Info().Int("certificates", len(certificates)).Msg("The certificates are renewed once the cluster Redis server is back")
		}

		return nil, func() {}
	}

	type renewal struct {
		id       string
		cert     *CertAndStore
		previous []byte
		lock     *cluster.Lock
		stop     context.CancelFunc
	}

	var renewals []*renewal

	reserved := coord.takeReserved()
	for id, reservation := range reserved {
		cert, previous := p.certificateByID(id)
		if cert == nil {
			reservation.stop()
			_ = reservation.lock.Release(ctx)

			continue
		}

		renewals = append(renewals, &renewal{id: id, cert: cert, previous: previous, lock: reservation.lock, stop: reservation.stop})
	}

	due := map[string]*CertAndStore{}
	var ids, items []string
	for _, cert := range certificates {
		id := certID(cert.Domain.ToStrArray())
		if _, ok := reserved[id]; ok {
			continue
		}

		due[id] = cert
		ids = append(ids, id)
		items = append(items, renewItem(id, certificateSum(cert.Certificate.Certificate)))
	}

	if len(ids) > 0 {
		// The instances above this one take what they can: this one waits for its turn, and renews the rest.
		coord.announceRenewals(ctx, items)

		for id, lock := range coord.acquireAll(ctx, ids) {
			_, stop := coord.hold(ctx, lock)
			renewals = append(renewals, &renewal{id: id, cert: due[id], previous: due[id].Certificate.Certificate, lock: lock, stop: stop})
		}
	}

	if len(renewals) == 0 {
		return nil, func() {}
	}

	// Another instance may have renewed a certificate while this one waited for its turn.
	coord.store.refresh(p.ResolverName)

	var elected []*CertAndStore
	var held []*renewal
	for _, renewal := range renewals {
		if !p.hasCertificate(renewal.cert.Domain, renewal.previous) {
			renewal.stop()
			_ = renewal.lock.Release(ctx)

			continue
		}

		elected = append(elected, renewal.cert)
		held = append(held, renewal)
	}

	return elected, func() {
		for _, renewal := range held {
			renewal.stop()

			if p.hasCertificate(renewal.cert.Domain, renewal.previous) {
				countIssuance(p.ResolverName, issuanceRenew, false, errNotRenewed)

				// A renewal is never abandoned: it is tried again, at most every few hours.
				coord.failed(ctx, renewal.id, errNotRenewed, false)
			} else {
				countIssuance(p.ResolverName, issuanceRenew, true, nil)

				coord.succeeded(ctx, renewal.id)
			}

			_ = renewal.lock.Release(ctx)
		}
	}
}

var errNotRenewed = errors.New("the certificate was not renewed")

// certificateByID returns the certificate the resolver holds for a set of domains, and its content at this time.
func (p *Provider) certificateByID(id string) (*CertAndStore, []byte) {
	p.certificatesMu.RLock()
	defer p.certificatesMu.RUnlock()

	for _, cert := range p.certificates {
		if certID(cert.Domain.ToStrArray()) == id {
			return cert, cert.Certificate.Certificate
		}
	}

	return nil, nil
}

// certificateSum identifies the certificate the resolver holds for a set of domains, empty when it holds none.
func (p *Provider) certificateSum(id string) string {
	_, certificate := p.certificateByID(id)
	if certificate == nil {
		return ""
	}

	return certificateSum(certificate)
}

func certificateSum(certificate []byte) string {
	sum := sha256.Sum256(certificate)

	return hex.EncodeToString(sum[:8])
}

// hasCertificate tells whether the resolver holds this exact certificate for the domain.
func (p *Provider) hasCertificate(domain types.Domain, certificate []byte) bool {
	p.certificatesMu.RLock()
	defer p.certificatesMu.RUnlock()

	for _, cert := range p.certificates {
		if reflect.DeepEqual(domain, cert.Domain) {
			return bytes.Equal(certificate, cert.Certificate.Certificate)
		}
	}

	return false
}

// uncoveredDomains returns the domains that no certificate of the TLS store or of the resolver covers.
// Unlike getUncheckedDomains, it does not mark them as being resolved.
func (p *Provider) uncoveredDomains(domains []string, tlsStore string) []string {
	var existing []string

	if p.tlsManager != nil {
		if store := p.tlsManager.GetStore(tlsStore); store != nil {
			existing = append(existing, store.GetAllDomains()...)
		}
	}

	p.certificatesMu.RLock()
	for _, cert := range p.certificates {
		existing = append(existing, strings.Join(cert.Domain.ToStrArray(), ","))
	}
	p.certificatesMu.RUnlock()

	var uncovered []string
	for _, domain := range domains {
		if !isDomainAlreadyChecked(domain, existing) {
			uncovered = append(uncovered, domain)
		}
	}

	return uncovered
}
