// Package mesh links the Traefix instances of a cluster: each instance announces the routers it discovered,
// and forwards to the instance that owns it a request for a router it does not have.
package mesh

import (
	"crypto/ed25519"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"slices"
	"strings"
	"time"
)

const (
	// defaultAuthorityIterations makes the search for a token from a certificate of the cluster expensive.
	// It is the PBKDF2-HMAC-SHA256 count OWASP recommends, and costs a fraction of a second, once, at startup.
	defaultAuthorityIterations = 600_000
	// authoritySalt is versioned: another derivation would need another one.
	authoritySalt = "traefix-mesh-authority/v1/"

	nodeURIScheme = "traefix"
	nodeURIHost   = "mesh"
)

// authorityIterations is a variable for the tests, which derive many authorities.
// The instances of a cluster must all use the same count: another one gives another authority.
var authorityIterations = defaultAuthorityIterations

// The certificates have a fixed validity, and are verified at a fixed time:
// what admits an instance is the token it holds, which no date revokes, and the clocks of the instances do not have to agree.
var (
	certificatesNotBefore = time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	certificatesNotAfter  = time.Date(2125, time.January, 1, 0, 0, 0, 0, time.UTC)
	certificatesCheckedAt = certificatesNotBefore.Add(24 * time.Hour)
)

// authority signs the certificates of the instances that hold a token.
// The instances that hold the same token derive the same one, so none has to be distributed.
type authority struct {
	key  ed25519.PrivateKey
	cert *x509.Certificate
}

func deriveAuthority(token, prefix string) (*authority, error) {
	seed, err := pbkdf2.Key(sha256.New, token, []byte(authoritySalt+prefix), authorityIterations, ed25519.SeedSize)
	if err != nil {
		return nil, fmt.Errorf("deriving the mesh authority: %w", err)
	}

	key := ed25519.NewKeyFromSeed(seed)

	public, ok := key.Public().(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("deriving the mesh authority: unexpected public key type")
	}

	keyID := sha256.Sum256(public)

	// Nothing here is random, and Ed25519 signatures are deterministic: every instance builds the same certificate.
	template := &x509.Certificate{
		SerialNumber:          new(big.Int).SetBytes(keyID[:16]),
		Subject:               pkix.Name{Organization: []string{prefix}, CommonName: "Traefix mesh authority"},
		NotBefore:             certificatesNotBefore,
		NotAfter:              certificatesNotAfter,
		IsCA:                  true,
		BasicConstraintsValid: true,
		MaxPathLenZero:        true,
		KeyUsage:              x509.KeyUsageCertSign,
		SubjectKeyId:          keyID[:20],
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, public, key)
	if err != nil {
		return nil, fmt.Errorf("creating the certificate of the mesh authority: %w", err)
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("reading the certificate of the mesh authority: %w", err)
	}

	return &authority{key: key, cert: cert}, nil
}

// fingerprint identifies the authority, and so the token, without revealing it.
func (a *authority) fingerprint() string {
	sum := sha256.Sum256(a.cert.Raw)

	return hex.EncodeToString(sum[:8])
}

// issue creates the certificate of an instance, with a key that never leaves the process.
func (a *authority) issue(prefix, nodeID string) (tls.Certificate, error) {
	public, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return tls.Certificate{}, err
	}

	template := &x509.Certificate{
		SerialNumber:   serial,
		Subject:        pkix.Name{Organization: []string{prefix}, CommonName: nodeID},
		NotBefore:      certificatesNotBefore,
		NotAfter:       certificatesNotAfter,
		KeyUsage:       x509.KeyUsageDigitalSignature,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		URIs:           []*url.URL{nodeURI(prefix, nodeID)},
		AuthorityKeyId: a.cert.SubjectKeyId,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, a.cert, public, a.key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("creating the mesh certificate of the instance: %w", err)
	}

	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("reading the mesh certificate of the instance: %w", err)
	}

	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}, nil
}

// nodeURI names an instance in its certificate. The identifiers of the instances are not DNS names,
// and neither are the prefixes of the clusters, so both go in the path.
func nodeURI(prefix, nodeID string) *url.URL {
	return &url.URL{Scheme: nodeURIScheme, Host: nodeURIHost, Path: nodeURIPath(prefix) + nodeID}
}

func nodeURIPath(prefix string) string {
	return "/" + prefix + "/node/"
}

// Identity is what an instance shows to the others, and what it checks of them:
// a certificate signed by the authority of its token, verified against the authorities of the tokens it accepts.
type Identity struct {
	prefix      string
	nodeID      string
	certificate tls.Certificate
	roots       *x509.CertPool
	// fingerprints are the ones of the accepted authorities, the one of the token of this instance first.
	fingerprints []string
}

// NewIdentity creates the identity of an instance from the token of the cluster.
// The additional tokens are accepted from the other instances, e.g. while the token of the cluster is being changed.
func NewIdentity(prefix, nodeID, token string, additionalTokens []string) (*Identity, error) {
	if token == "" {
		return nil, errors.New("the mesh needs the token of the cluster")
	}

	own, err := deriveAuthority(token, prefix)
	if err != nil {
		return nil, err
	}

	certificate, err := own.issue(prefix, nodeID)
	if err != nil {
		return nil, err
	}

	identity := &Identity{
		prefix:       prefix,
		nodeID:       nodeID,
		certificate:  certificate,
		roots:        x509.NewCertPool(),
		fingerprints: []string{own.fingerprint()},
	}

	identity.roots.AddCert(own.cert)

	for _, other := range additionalTokens {
		accepted, err := deriveAuthority(other, prefix)
		if err != nil {
			return nil, err
		}

		identity.roots.AddCert(accepted.cert)
		identity.fingerprints = append(identity.fingerprints, accepted.fingerprint())
	}

	return identity, nil
}

// NodeID returns the identifier of the instance.
func (i *Identity) NodeID() string {
	return i.nodeID
}

// Fingerprint identifies the token of the instance without revealing it: two instances with the same one hold the same token.
func (i *Identity) Fingerprint() string {
	return i.fingerprints[0]
}

// Accepts tells whether the instance accepts the instances whose token has this fingerprint.
func (i *Identity) Accepts(fingerprint string) bool {
	return slices.Contains(i.fingerprints, fingerprint)
}

// ServerTLS returns the TLS configuration of the side that accepts a link.
func (i *Identity) ServerTLS(protocols ...string) *tls.Config {
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{i.certificate},
		NextProtos:   protocols,
		// The certificate is checked by verifyConnection, against the authorities and at the fixed time.
		ClientAuth:       tls.RequireAnyClientCert,
		VerifyConnection: i.verifyConnection,
	}
}

// ClientTLS returns the TLS configuration of the side that opens a link.
func (i *Identity) ClientTLS(protocols ...string) *tls.Config {
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{i.certificate},
		NextProtos:   protocols,
		// The standard verification wants a DNS name and the current time. The certificate is checked by verifyConnection instead.
		InsecureSkipVerify: true, //nolint:gosec // See the comment above.
		VerifyConnection:   i.verifyConnection,
	}
}

// PeerNode returns the identifier of the instance at the other end of a connection,
// after checking that an authority this instance accepts signed its certificate.
func (i *Identity) PeerNode(state tls.ConnectionState) (string, error) {
	if len(state.PeerCertificates) == 0 {
		return "", errors.New("the other instance showed no certificate")
	}

	leaf := state.PeerCertificates[0]

	intermediates := x509.NewCertPool()
	for _, cert := range state.PeerCertificates[1:] {
		intermediates.AddCert(cert)
	}

	_, err := leaf.Verify(x509.VerifyOptions{
		Roots:         i.roots,
		Intermediates: intermediates,
		CurrentTime:   certificatesCheckedAt,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		return "", fmt.Errorf("the other instance does not hold a token of this cluster: %w", err)
	}

	for _, uri := range leaf.URIs {
		if uri.Scheme != nodeURIScheme || uri.Host != nodeURIHost {
			continue
		}

		if nodeID, ok := strings.CutPrefix(uri.Path, nodeURIPath(i.prefix)); ok && nodeID != "" {
			return nodeID, nil
		}
	}

	return "", errors.New("the certificate of the other instance names no instance of this cluster")
}

func (i *Identity) verifyConnection(state tls.ConnectionState) error {
	_, err := i.PeerNode(state)

	return err
}
