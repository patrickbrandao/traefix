package mesh

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/types"
)

func Test_wantedCertificates(t *testing.T) {
	routes := []Route{
		{Kind: KindHTTP, Name: "nas@docker", Rule: "Host(`NAS.example.com`) || Host(`files.example.com`)", TLS: &RouteTLS{CertResolver: "le"}},
		{Kind: KindHTTP, Name: "api@docker", Rule: "Host(`api.example.com`)", TLS: &RouteTLS{CertResolver: "le", Domains: []types.Domain{{Main: "example.com", SANs: []string{"*.example.com"}}}}},
		{Kind: KindHTTP, Name: "plain@docker", Rule: "Host(`plain.example.com`)", TLS: &RouteTLS{}},
		{Kind: KindHTTP, Name: "clear@docker", Rule: "Host(`clear.example.com`)"},
		{Kind: KindTCP, Name: "pg@docker", Rule: "HostSNI(`pg.example.com`)", TLS: &RouteTLS{CertResolver: "staging"}},
		{Kind: KindTCP, Name: "pass@docker", Rule: "HostSNI(`pass.example.com`)", TLS: &RouteTLS{CertResolver: "le", Passthrough: true}},
		{Kind: KindTCP, Name: "any@docker", Rule: "HostSNI(`*`)", TLS: &RouteTLS{CertResolver: "le"}},
	}

	// The domains of the TLS configuration come before the ones of the rule; a passthrough router needs none.
	assert.Equal(t, map[string][]string{
		"le":      {"nas.example.com", "files.example.com", "example.com", "*.example.com"},
		"staging": {"pg.example.com"},
	}, wantedCertificates(routes))
}

func Test_covers(t *testing.T) {
	testCases := []struct {
		name, domain string
		expected     bool
	}{
		{name: "nas.example.com", domain: "nas.example.com", expected: true},
		{name: "*.example.com", domain: "nas.example.com", expected: true},
		{name: "*.example.com", domain: "a.nas.example.com"},
		{name: "*.example.com", domain: "example.com"},
		{name: "*.example.com", domain: "*.example.com", expected: true},
		{name: "nas.example.com", domain: "*.example.com"},
		{name: "other.example.com", domain: "nas.example.com"},
	}

	for _, test := range testCases {
		assert.Equal(t, test.expected, covers(test.name, test.domain), "%s for %s", test.name, test.domain)
	}

	assert.True(t, coversAny([]string{"example.com", "*.Example.com"}, []string{"other.org", "nas.example.com"}))
	assert.False(t, coversAny([]string{"example.com"}, []string{"nas.example.com"}))
}

// fakeCertificates is the store of the resolvers of a central in the tests.
type fakeCertificates struct {
	mu           sync.Mutex
	certificates map[string][]Certificate
}

func (f *fakeCertificates) set(resolver string, certificates ...Certificate) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.certificates[resolver] = certificates
}

func (f *fakeCertificates) source(resolver string) ([]Certificate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.certificates[resolver], nil
}

func testCertificate(serial string, notAfter time.Time, domains ...string) Certificate {
	return Certificate{Domains: domains, Certificate: "CERT " + serial, Key: "KEY " + serial, NotAfter: notAfter}
}

func TestMesh_serveCertificates(t *testing.T) {
	node := newTestNode(t, "srv1", testToken, nil)

	later := time.Now().Add(30 * 24 * time.Hour)

	store := &fakeCertificates{certificates: map[string][]Certificate{}}
	store.set("le",
		testCertificate("nas", later, "nas.example.com"),
		testCertificate("wild", later, "*.wild.example.com"),
		testCertificate("other", later, "other.example.com"),
	)
	node.mesh.SetCertificateSource(store.source)

	node.mesh.mu.Lock()
	node.mesh.imported["casa1"] = &importedTable{routes: []Route{
		{Kind: KindHTTP, Name: "nas@docker", Rule: "Host(`nas.example.com`)", TLS: &RouteTLS{CertResolver: "le"}},
		{Kind: KindHTTP, Name: "box@docker", Rule: "Host(`box.wild.example.com`)", TLS: &RouteTLS{CertResolver: "le"}},
	}}
	node.mesh.mu.Unlock()

	ask := func(peer, etag string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "https://"+controlHost+pathCertificates, nil)
		req = req.WithContext(context.WithValue(req.Context(), peerKey{}, peer))

		if etag != "" {
			req.Header.Set("If-None-Match", etag)
		}

		recorder := httptest.NewRecorder()
		node.mesh.serveCertificates(recorder, req)

		return recorder
	}

	// A satellite gets the certificates of the domains of its routers, and no other.
	first := ask("casa1", "")
	require.Equal(t, http.StatusOK, first.Code)
	assert.Contains(t, first.Body.String(), "CERT nas")
	assert.Contains(t, first.Body.String(), "CERT wild")
	assert.NotContains(t, first.Body.String(), "CERT other")

	// Nothing changed: nothing is sent again.
	etag := first.Header().Get("ETag")
	require.NotEmpty(t, etag)
	assert.Equal(t, http.StatusNotModified, ask("casa1", etag).Code)

	// A renewal changes the answer.
	store.set("le", testCertificate("nas2", later.Add(time.Hour), "nas.example.com"))
	renewed := ask("casa1", etag)
	require.Equal(t, http.StatusOK, renewed.Code)
	assert.Contains(t, renewed.Body.String(), "CERT nas2")

	// An instance that announces nothing gets nothing.
	assert.JSONEq(t, `{"certificates":[]}`, ask("srv9", "").Body.String())
}

func (n *testNode) servedCertificates() []string {
	conf, _ := n.mesh.buildConfiguration()
	if conf.TLS == nil {
		return nil
	}

	var served []string
	for _, certificate := range conf.TLS.Certificates {
		served = append(served, string(certificate.CertFile))
	}

	return served
}

func TestMesh_satellite_certificates(t *testing.T) {
	later := time.Now().Add(30 * 24 * time.Hour)

	store := &fakeCertificates{certificates: map[string][]Certificate{}}
	store.set("le", testCertificate("nas", later, "nas.example.com"), testCertificate("other", later, "other.example.com"))

	central := newTestNode(t, "srv1", testToken, nil)
	central.mesh.SetCertificateSource(store.source)

	satellite := newTestNode(t, "casa1", testToken, asSatellite, central.address)
	satellite.mesh.Transform(t.Context(), dynamic.Configurations{
		"docker": {HTTP: &dynamic.HTTPConfiguration{Routers: map[string]*dynamic.Router{
			"nas": {Rule: "Host(`nas.example.com`)", EntryPoints: []string{"websecure"}, Service: "nas", TLS: &dynamic.RouterTLSConfig{CertResolver: "le"}},
		}}},
	})

	// The central follows the routers of the satellite, and the satellite gets the certificate of its own.
	require.Eventually(t, func() bool {
		served := satellite.servedCertificates()
		return len(served) == 1 && served[0] == "CERT nas"
	}, 10*time.Second, 10*time.Millisecond)

	// A renewal reaches the satellite.
	store.set("le", testCertificate("nas2", later.Add(time.Hour), "nas.example.com"))

	require.Eventually(t, func() bool {
		served := satellite.servedCertificates()
		return len(served) == 1 && served[0] == "CERT nas2"
	}, 10*time.Second, 10*time.Millisecond)

	// The central goes away: the satellite keeps serving what it has.
	central.stop()
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, []string{"CERT nas2"}, satellite.servedCertificates())

	// An expired certificate is not served.
	satellite.mesh.setReceivedCertificates("srv1", []Certificate{testCertificate("old", time.Now().Add(-time.Hour), "nas.example.com")})
	assert.Empty(t, satellite.servedCertificates())
}

func TestMesh_member_servesNoCertificates(t *testing.T) {
	node := newTestNode(t, "srv1", testToken, nil)

	conf, _ := node.mesh.buildConfiguration()
	assert.Nil(t, conf.TLS)
}
