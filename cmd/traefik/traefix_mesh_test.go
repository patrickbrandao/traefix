package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/config/static"
	"github.com/traefik/traefik/v3/pkg/provider/acme"
	"github.com/traefik/traefik/v3/pkg/types"
)

func TestSatelliteResolvers(t *testing.T) {
	resolvers := map[string]static.CertificateResolver{
		"le":      {ACME: &acme.Configuration{Email: "ops@example.com"}},
		"staging": {ACME: &acme.Configuration{}},
		"ts":      {Tailscale: &struct{}{}},
	}

	member := &static.Configuration{
		CertificatesResolvers: resolvers,
		Cluster:               &cluster.Configuration{Mesh: &cluster.Mesh{}},
	}

	conf, names := satelliteResolvers(member)
	assert.Same(t, member, conf)
	assert.Empty(t, names)

	satellite := &static.Configuration{
		CertificatesResolvers: resolvers,
		Cluster:               &cluster.Configuration{Mesh: &cluster.Mesh{Satellite: true}},
	}

	// A satellite keeps its other resolvers, and the names of the ACME ones for its routers.
	conf, names = satelliteResolvers(satellite)
	assert.Equal(t, []string{"le", "staging"}, names)
	assert.Equal(t, map[string]static.CertificateResolver{"ts": resolvers["ts"]}, conf.CertificatesResolvers)
	assert.Len(t, satellite.CertificatesResolvers, 3)

	withoutCluster := &static.Configuration{CertificatesResolvers: resolvers}

	conf, names = satelliteResolvers(withoutCluster)
	assert.Same(t, withoutCluster, conf)
	assert.Empty(t, names)
}

// fakeACMEStore holds the certificates of one resolver.
type fakeACMEStore struct {
	certificates []*acme.CertAndStore
}

func (s *fakeACMEStore) GetAccount(string) (*acme.Account, error) { return nil, nil }
func (s *fakeACMEStore) SaveAccount(string, *acme.Account) error  { return nil }
func (s *fakeACMEStore) GetCertificates(string) ([]*acme.CertAndStore, error) {
	return s.certificates, nil
}
func (s *fakeACMEStore) SaveCertificates(string, []*acme.CertAndStore) error { return nil }

func TestACMECertificates(t *testing.T) {
	expires := time.Date(2027, 3, 1, 12, 0, 0, 0, time.UTC)

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{SerialNumber: big.NewInt(1), DNSNames: []string{"nas.example.com"}, NotBefore: expires.Add(-90 * 24 * time.Hour), NotAfter: expires}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	chain := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	store := &fakeACMEStore{certificates: []*acme.CertAndStore{
		{Certificate: acme.Certificate{Domain: types.Domain{Main: "nas.example.com", SANs: []string{"files.example.com"}}, Certificate: chain, Key: []byte("KEY")}},
		// A certificate without its key is of no use to a satellite.
		{Certificate: acme.Certificate{Domain: types.Domain{Main: "broken.example.com"}, Certificate: chain}},
	}}

	source := acmeCertificates([]*acme.Provider{{ResolverName: "le", Store: store}})

	certificates, err := source("le")
	require.NoError(t, err)
	require.Len(t, certificates, 1)

	assert.Equal(t, []string{"nas.example.com", "files.example.com"}, certificates[0].Domains)
	assert.Equal(t, string(chain), certificates[0].Certificate)
	assert.Equal(t, "KEY", certificates[0].Key)
	assert.True(t, expires.Equal(certificates[0].NotAfter))

	// A resolver this instance does not have gives nothing.
	certificates, err = source("staging")
	require.NoError(t, err)
	assert.Empty(t, certificates)
}
