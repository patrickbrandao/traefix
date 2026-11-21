package acme

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	traefiktls "github.com/traefik/traefik/v3/pkg/tls"
	"github.com/traefik/traefik/v3/pkg/types"
)

// newStoreTestCertificate creates a leaf certificate signed by a test CA, as a bundle of the leaf and the CA.
func newStoreTestCertificate(t *testing.T, notAfter time.Time, main string, sans ...string) *CertAndStore {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA", Organization: []string{"Traefix Tests"}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, caKey.Public(), caKey)
	require.NoError(t, err)

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: main},
		DNSNames:     append([]string{main}, sans...),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, caTemplate, key.Public(), caKey)
	require.NoError(t, err)

	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	bundle := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	bundle = append(bundle, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})...)

	return &CertAndStore{
		Certificate: Certificate{
			Domain:      types.Domain{Main: main, SANs: sans},
			Certificate: bundle,
			Key:         pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}),
		},
		Store: traefiktls.DefaultTLSStoreName,
	}
}

func readStoreTestCertJSON(t *testing.T, dir, name string) *certJSON {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, name, "cert.json"))
	require.NoError(t, err)

	doc := &certJSON{}
	require.NoError(t, json.Unmarshal(data, doc))

	return doc
}

func in90Days() time.Time {
	return time.Now().Add(90 * 24 * time.Hour).Truncate(time.Second)
}

func TestDirStore_SaveCertificates_layout(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "certs")
	cert := newStoreTestCertificate(t, in90Days(), "example.com", "*.example.com", "www.example.org")

	store := NewDirStore(dir)
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{cert}))

	// The wildcard shares the directory of its domain.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	assert.Equal(t, []string{"example.com", "www.example.org"}, names)

	expectedPerms := map[string]os.FileMode{
		"fullchain.pem": 0o644,
		"cert.pem":      0o644,
		"chain.pem":     0o644,
		"privkey.pem":   0o600,
		"cert.md5":      0o644,
		"cert.json":     0o600,
	}

	for _, name := range names {
		files, err := os.ReadDir(filepath.Join(dir, name))
		require.NoError(t, err)
		require.Len(t, files, len(expectedPerms))

		for _, file := range files {
			info, err := file.Info()
			require.NoError(t, err)

			if runtime.GOOS != "windows" {
				assert.Equal(t, expectedPerms[file.Name()], info.Mode().Perm(), file.Name())
			}
		}

		fullchain, err := os.ReadFile(filepath.Join(dir, name, "fullchain.pem"))
		require.NoError(t, err)
		assert.Equal(t, cert.Certificate.Certificate, fullchain)

		leaf, err := os.ReadFile(filepath.Join(dir, name, "cert.pem"))
		require.NoError(t, err)

		chain, err := os.ReadFile(filepath.Join(dir, name, "chain.pem"))
		require.NoError(t, err)
		assert.Equal(t, fullchain, append(leaf, chain...))

		privkey, err := os.ReadFile(filepath.Join(dir, name, "privkey.pem"))
		require.NoError(t, err)
		assert.Equal(t, cert.Key, privkey)

		sum := md5.Sum(leaf)
		certMD5, err := os.ReadFile(filepath.Join(dir, name, "cert.md5"))
		require.NoError(t, err)
		assert.Equal(t, hex.EncodeToString(sum[:])+"\n", string(certMD5))

		doc := readStoreTestCertJSON(t, dir, name)
		assert.Equal(t, name, doc.FQDN)
		assert.Equal(t, []string{"example.com", "*.example.com", "www.example.org"}, doc.SANs)
		assert.True(t, doc.IsWildcard)
		assert.True(t, doc.IsCurrentlyValid)
		assert.Equal(t, "example.com", doc.SubjectCN)
		assert.Equal(t, "Test CA", doc.IssuerCN)
		assert.Equal(t, "Traefix Tests", doc.IssuerOrg)
		assert.Equal(t, "ECDSA", doc.KeyAlgorithm)
		assert.Equal(t, 256, doc.KeyBits)
		assert.Equal(t, "acme", doc.Source)
		assert.Equal(t, "le", doc.AcmeResolver)
		assert.Equal(t, hex.EncodeToString(sum[:]), doc.CertMD5)
		assert.Equal(t, doc.CertMD5, doc.Hash.Cert)
		assert.Equal(t, string(fullchain), doc.PEM.Fullchain)
		assert.Equal(t, string(privkey), doc.PEM.Privkey)
		assert.Equal(t, &certJSONTraefix{
			Resolver: "le",
			Main:     "example.com",
			SANs:     []string{"*.example.com", "www.example.org"},
			Store:    "default",
		}, doc.Traefix)
	}
}

// The consumers of traefik-certs decode cert.json with these keys.
func TestDirStore_certJSON_keys(t *testing.T) {
	dir := t.TempDir()

	store := NewDirStore(dir)
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{newStoreTestCertificate(t, in90Days(), "example.com")}))

	data, err := os.ReadFile(filepath.Join(dir, "example.com", "cert.json"))
	require.NoError(t, err)

	var doc map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &doc))

	var keys []string
	for key := range doc {
		keys = append(keys, key)
	}

	assert.ElementsMatch(t, []string{
		"fqdn", "sans", "is_wildcard", "subject_cn", "issuer_cn", "issuer_org", "serial",
		"not_before", "not_before_unix", "not_after", "not_after_unix", "seconds_to_expiry",
		"is_currently_valid", "fingerprint_sha256", "cert_md5", "key_algorithm", "key_bits",
		"signature_algorithm", "source", "acme_resolver", "updated_at", "pem", "hash", "traefix",
	}, keys)

	var pemKeys map[string]string
	require.NoError(t, json.Unmarshal(doc["pem"], &pemKeys))
	assert.Len(t, pemKeys, 4)
	assert.Contains(t, pemKeys, "cert")
	assert.Contains(t, pemKeys, "chain")
	assert.Contains(t, pemKeys, "fullchain")
	assert.Contains(t, pemKeys, "privkey")
}

func TestDirStore_GetCertificates(t *testing.T) {
	dir := t.TempDir()

	first := newStoreTestCertificate(t, in90Days(), "example.com", "www.example.com")
	second := newStoreTestCertificate(t, in90Days(), "*.example.org")
	second.Store = "other"
	third := newStoreTestCertificate(t, in90Days(), "example.net")

	store := NewDirStore(dir)
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{first, second}))
	require.NoError(t, store.SaveCertificates("zerossl", []*CertAndStore{third}))

	store = NewDirStore(dir)

	certificates, err := store.GetCertificates("le")
	require.NoError(t, err)
	assert.ElementsMatch(t, []*CertAndStore{first, second}, certificates)

	certificates, err = store.GetCertificates("zerossl")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{third}, certificates)

	certificates, err = store.GetCertificates("unknown")
	require.NoError(t, err)
	assert.Empty(t, certificates)
}

func TestDirStore_GetCertificates_missingDirectory(t *testing.T) {
	store := NewDirStore(filepath.Join(t.TempDir(), "missing"))

	certificates, err := store.GetCertificates("le")
	require.NoError(t, err)
	assert.Empty(t, certificates)

	account, err := store.GetAccount("le")
	require.NoError(t, err)
	assert.Nil(t, account)
}

// Two certificates for a domain and for its wildcard cannot share the directory.
func TestDirStore_SaveCertificates_wildcardCollision(t *testing.T) {
	dir := t.TempDir()

	wildcard := newStoreTestCertificate(t, in90Days().Add(24*time.Hour), "*.example.com")
	apex := newStoreTestCertificate(t, in90Days(), "example.com")

	store := NewDirStore(dir)

	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{wildcard}))
	assert.Equal(t, []string{"*.example.com"}, readStoreTestCertJSON(t, dir, "example.com").SANs)
	assert.NoDirExists(t, filepath.Join(dir, "_wildcard.example.com"))

	// The certificate with the exact name takes the directory, although it expires first.
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{wildcard, apex}))
	assert.Equal(t, []string{"example.com"}, readStoreTestCertJSON(t, dir, "example.com").SANs)
	assert.Equal(t, []string{"*.example.com"}, readStoreTestCertJSON(t, dir, "_wildcard.example.com").SANs)

	certificates, err := NewDirStore(dir).GetCertificates("le")
	require.NoError(t, err)
	assert.ElementsMatch(t, []*CertAndStore{wildcard, apex}, certificates)
}

func TestDirStore_SaveCertificates_wildcardCollisionBetweenResolvers(t *testing.T) {
	dir := t.TempDir()

	wildcard := newStoreTestCertificate(t, in90Days(), "*.example.com")
	apex := newStoreTestCertificate(t, in90Days(), "example.com")

	store := NewDirStore(dir)
	require.NoError(t, store.SaveCertificates("dns", []*CertAndStore{wildcard}))
	require.NoError(t, store.SaveCertificates("http", []*CertAndStore{apex}))

	store = NewDirStore(dir)

	certificates, err := store.GetCertificates("dns")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{wildcard}, certificates)

	certificates, err = store.GetCertificates("http")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{apex}, certificates)
}

func TestDirStore_SaveCertificates_writesOnlyChanges(t *testing.T) {
	dir := t.TempDir()

	first := newStoreTestCertificate(t, in90Days(), "example.com")
	second := newStoreTestCertificate(t, in90Days(), "example.org")

	now := time.Now()
	store := NewDirStore(dir)
	store.now = func() time.Time { return now }

	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{first}))
	firstUpdatedAt := readStoreTestCertJSON(t, dir, "example.com").UpdatedAt

	now = now.Add(time.Hour)
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{first, second}))

	assert.Equal(t, firstUpdatedAt, readStoreTestCertJSON(t, dir, "example.com").UpdatedAt)
	assert.NotEqual(t, firstUpdatedAt, readStoreTestCertJSON(t, dir, "example.org").UpdatedAt)

	// A new store on the same directory reads what is there before writing.
	now = now.Add(time.Hour)
	store = NewDirStore(dir)
	store.now = func() time.Time { return now }

	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{first, second}))
	assert.Equal(t, firstUpdatedAt, readStoreTestCertJSON(t, dir, "example.com").UpdatedAt)
}

func TestDirStore_SaveCertificates_renewal(t *testing.T) {
	dir := t.TempDir()

	old := newStoreTestCertificate(t, in90Days(), "example.com")
	renewed := newStoreTestCertificate(t, in90Days().Add(60*24*time.Hour), "example.com")

	store := NewDirStore(dir)
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{old}))
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{renewed}))

	privkey, err := os.ReadFile(filepath.Join(dir, "example.com", "privkey.pem"))
	require.NoError(t, err)
	assert.Equal(t, renewed.Key, privkey)

	// A certificate never replaces a more recent one for the same names.
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{old}))

	privkey, err = os.ReadFile(filepath.Join(dir, "example.com", "privkey.pem"))
	require.NoError(t, err)
	assert.Equal(t, renewed.Key, privkey)
}

// A directory written by the cert-watcher of traefik-certs has no traefix object.
func TestDirStore_GetCertificates_traefikCertsFile(t *testing.T) {
	dir := t.TempDir()

	cert := newStoreTestCertificate(t, in90Days(), "example.com", "www.example.com")

	doc, err := newCertJSON("www.example.com", "le", cert, time.Now())
	require.NoError(t, err)
	doc.Traefix = nil
	// The order of the names of a certificate is not the order the ACME provider tracks.
	doc.SANs = []string{"www.example.com", "example.com"}

	data, err := json.Marshal(doc)
	require.NoError(t, err)

	require.NoError(t, os.Mkdir(filepath.Join(dir, "www.example.com"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "www.example.com", "cert.json"), data, 0o600))

	certificates, err := NewDirStore(dir).GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{cert}, certificates)
}

func TestDirStore_GetCertificates_ignoresUnusableDirectories(t *testing.T) {
	dir := t.TempDir()

	cert := newStoreTestCertificate(t, in90Days(), "example.com")
	require.NoError(t, NewDirStore(dir).SaveCertificates("le", []*CertAndStore{cert}))

	require.NoError(t, os.Mkdir(filepath.Join(dir, "empty.example.com"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "broken.example.com"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.example.com", "cert.json"), []byte("{"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "example.com", ".tmp-cert.json-123"), []byte("{"), 0o600))

	store := NewDirStore(dir)

	certificates, err := store.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{cert}, certificates)

	assert.Equal(t, []string{"broken.example.com", "empty.example.com"}, store.unreadable)
	assert.NoFileExists(t, filepath.Join(dir, "example.com", ".tmp-cert.json-123"))
}

func TestDirStore_Account(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "certs")

	account := &Account{Email: "ops@example.com", PrivateKey: []byte("key")}

	store := NewDirStore(dir)
	require.NoError(t, store.SaveAccount("le", account))

	path := filepath.Join(dir, "_traefix", "le", "account.json")
	info, err := os.Stat(path)
	require.NoError(t, err)

	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}

	store = NewDirStore(dir)

	got, err := store.GetAccount("le")
	require.NoError(t, err)
	assert.Equal(t, account, got)

	got, err = store.GetAccount("other")
	require.NoError(t, err)
	assert.Nil(t, got)

	// The directory of the accounts is not a certificate.
	certificates, err := store.GetCertificates("le")
	require.NoError(t, err)
	assert.Empty(t, certificates)
	assert.Empty(t, store.unreadable)
}

func TestDirStore_Account_rejectsUnsafeResolverName(t *testing.T) {
	store := NewDirStore(t.TempDir())

	for _, name := range []string{"", "../le", "a/b", "_le", ".le"} {
		require.Error(t, store.SaveAccount(name, &Account{}), name)

		_, err := store.GetAccount(name)
		require.Error(t, err, name)
	}
}
