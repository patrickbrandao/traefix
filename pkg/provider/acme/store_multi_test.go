package acme

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v3/pkg/safe"
)

// readStoreTestFile waits for the storage file, which LocalStore writes in the background, to hold the certificates.
func readStoreTestFile(t *testing.T, filename, resolverName string, certificates int) *StoredData {
	t.Helper()

	var stored *StoredData
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(filename)
		if err != nil {
			return false
		}

		content := map[string]*StoredData{}
		//nolint:musttag // StoredData has the untagged encoding of the storage file.
		if err := json.Unmarshal(data, &content); err != nil {
			return false
		}

		stored = content[resolverName]

		return stored != nil && len(stored.Certificates) == certificates
	}, 5*time.Second, 10*time.Millisecond)

	return stored
}

func writeStoreTestFile(t *testing.T, filename string, content map[string]*StoredData) {
	t.Helper()

	//nolint:musttag // StoredData has the untagged encoding of the storage file.
	data, err := json.Marshal(content)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filename, data, 0o600))
}

func (b *StoreBuilder) mustBuild(t *testing.T, conf *Configuration) Store {
	t.Helper()

	store, err := b.Build("le", conf)
	require.NoError(t, err)

	return store
}

func TestStoreBuilder_Build_default(t *testing.T) {
	builder := NewStoreBuilder(safe.NewPool(t.Context()), nil)

	first := builder.mustBuild(t, &Configuration{Storage: filepath.Join(t.TempDir(), "acme.json")})
	assert.IsType(t, &LocalStore{}, first)

	// The resolvers with the same storage file share the store, as in Traefik.
	shared := filepath.Join(t.TempDir(), "acme.json")
	second := builder.mustBuild(t, &Configuration{Storage: shared})
	third := builder.mustBuild(t, &Configuration{Storage: shared})
	assert.Same(t, second, third)
}

func TestStoreBuilder_Build_fileWithDirExport(t *testing.T) {
	tmp := t.TempDir()
	storage := filepath.Join(tmp, "acme.json")
	dir := filepath.Join(tmp, "certs")

	existing := newStoreTestCertificate(t, in90Days(), "example.com")
	account := &Account{Email: "ops@example.com", PrivateKey: []byte("key")}
	writeStoreTestFile(t, storage, map[string]*StoredData{
		"le": {Account: account, Certificates: []*CertAndStore{existing}},
	})

	builder := NewStoreBuilder(safe.NewPool(t.Context()), nil)
	store := builder.mustBuild(t, &Configuration{
		Storage:   storage,
		CertStore: &CertStore{Primary: StoreFile, Dir: dir, Export: []string{StoreDir}},
	})

	// The first read brings the directory in line with the storage file.
	certificates, err := store.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{existing}, certificates)
	assert.FileExists(t, filepath.Join(dir, "example.com", "fullchain.pem"))

	// The account stays in the storage file.
	assert.NoDirExists(t, filepath.Join(dir, "_traefix"))

	issued := newStoreTestCertificate(t, in90Days(), "example.org")
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{existing, issued}))

	assert.FileExists(t, filepath.Join(dir, "example.org", "privkey.pem"))
	readStoreTestFile(t, storage, "le", 2)
}

func TestStoreBuilder_Build_dirOnly(t *testing.T) {
	tmp := t.TempDir()
	storage := filepath.Join(tmp, "acme.json")
	dir := filepath.Join(tmp, "certs")

	builder := NewStoreBuilder(safe.NewPool(t.Context()), nil)
	conf := &Configuration{Storage: storage, CertStore: &CertStore{Primary: StoreDir, Dir: dir}}
	store := builder.mustBuild(t, conf)

	account := &Account{Email: "ops@example.com", PrivateKey: []byte("key")}
	cert := newStoreTestCertificate(t, in90Days(), "example.com")

	require.NoError(t, store.SaveAccount("le", account))
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{cert}))

	// The storage file is not created when no store uses it.
	assert.NoFileExists(t, storage)

	store = NewStoreBuilder(safe.NewPool(t.Context()), nil).mustBuild(t, conf)

	gotAccount, err := store.GetAccount("le")
	require.NoError(t, err)
	assert.Equal(t, account, gotAccount)

	certificates, err := store.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{cert}, certificates)
}

func TestStoreBuilder_Build_dirWithFileExport(t *testing.T) {
	tmp := t.TempDir()
	storage := filepath.Join(tmp, "acme.json")
	dir := filepath.Join(tmp, "certs")

	builder := NewStoreBuilder(safe.NewPool(t.Context()), nil)
	store := builder.mustBuild(t, &Configuration{
		Storage:   storage,
		CertStore: &CertStore{Primary: StoreDir, Dir: dir, Export: []string{StoreFile}},
	})

	account := &Account{Email: "ops@example.com", PrivateKey: []byte("key")}
	cert := newStoreTestCertificate(t, in90Days(), "example.com")

	require.NoError(t, store.SaveAccount("le", account))
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{cert}))

	stored := readStoreTestFile(t, storage, "le", 1)
	assert.Equal(t, account, stored.Account)
	assert.Equal(t, []*CertAndStore{cert}, stored.Certificates)
}

// Moving from the storage file to the directory must not ask the ACME server for the certificates again.
func TestStoreBuilder_Build_dirImportsStorageFile(t *testing.T) {
	tmp := t.TempDir()
	storage := filepath.Join(tmp, "acme.json")
	dir := filepath.Join(tmp, "certs")

	account := &Account{Email: "ops@example.com", PrivateKey: []byte("key")}
	cert := newStoreTestCertificate(t, in90Days(), "example.com", "www.example.com")
	writeStoreTestFile(t, storage, map[string]*StoredData{
		"le": {Account: account, Certificates: []*CertAndStore{cert}},
	})

	conf := &Configuration{Storage: storage, CertStore: &CertStore{Primary: StoreDir, Dir: dir}}
	store := NewStoreBuilder(safe.NewPool(t.Context()), nil).mustBuild(t, conf)

	certificates, err := store.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{cert}, certificates)

	gotAccount, err := store.GetAccount("le")
	require.NoError(t, err)
	assert.Equal(t, account, gotAccount)

	assert.FileExists(t, filepath.Join(dir, "www.example.com", "cert.json"))
	assert.FileExists(t, filepath.Join(dir, "_traefix", "le", "account.json"))

	// Once the directory has data, the storage file is not read anymore.
	renewed := newStoreTestCertificate(t, in90Days().Add(24*time.Hour), "example.com", "www.example.com")
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{renewed}))

	store = NewStoreBuilder(safe.NewPool(t.Context()), nil).mustBuild(t, conf)

	certificates, err = store.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{renewed}, certificates)
}

func TestStoreBuilder_Build_dirImportFailsOnUnreadableStorageFile(t *testing.T) {
	tmp := t.TempDir()
	storage := filepath.Join(tmp, "acme.json")

	require.NoError(t, os.WriteFile(storage, []byte("{not json"), 0o600))

	store := NewStoreBuilder(safe.NewPool(t.Context()), nil).mustBuild(t, &Configuration{
		Storage:   storage,
		CertStore: &CertStore{Primary: StoreDir, Dir: filepath.Join(tmp, "certs")},
	})

	// An empty list here would make the provider ask for every certificate again.
	_, err := store.GetCertificates("le")
	require.Error(t, err)
}

type failingStore struct {
	Store
}

func (failingStore) SaveAccount(string, *Account) error {
	return errors.New("failing")
}

func (failingStore) SaveCertificates(string, []*CertAndStore) error {
	return errors.New("failing")
}

func TestMultiStore_exportFailureDoesNotFailTheSave(t *testing.T) {
	primary := NewDirStore(t.TempDir())

	store := &MultiStore{
		primary: primary,
		exports: []storeExport{{kind: StoreFile, store: failingStore{}, account: true}},
	}

	cert := newStoreTestCertificate(t, in90Days(), "example.com")

	require.NoError(t, store.SaveAccount("le", &Account{Email: "ops@example.com"}))
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{cert}))

	certificates, err := primary.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{cert}, certificates)
}

func TestMultiStore_primaryFailureFailsTheSave(t *testing.T) {
	store := &MultiStore{primary: failingStore{Store: NewDirStore(t.TempDir())}}

	require.Error(t, store.SaveAccount("le", &Account{}))
	require.Error(t, store.SaveCertificates("le", nil))
}

func TestCertStore_Validate(t *testing.T) {
	testCases := []struct {
		desc      string
		store     *CertStore
		expectErr bool
	}{
		{desc: "no certStore"},
		{desc: "file", store: &CertStore{Primary: StoreFile}},
		{desc: "file with dir export", store: &CertStore{Primary: StoreFile, Dir: "/certs", Export: []string{StoreDir}}},
		{desc: "dir", store: &CertStore{Primary: StoreDir, Dir: "/certs"}},
		{desc: "dir with file export", store: &CertStore{Primary: StoreDir, Dir: "/certs", Export: []string{StoreFile}}},
		{desc: "no primary", store: &CertStore{}, expectErr: true},
		{desc: "redis", store: &CertStore{Primary: StoreRedis}},
		{desc: "redis with exports", store: &CertStore{Primary: StoreRedis, Dir: "/certs", Export: []string{StoreDir, StoreFile}}},
		{desc: "redis with a tcerts prefix", store: &CertStore{Primary: StoreRedis, Tcerts: &TcertsExport{Enabled: true, Prefix: "certs"}}},
		{desc: "redis without tcerts", store: &CertStore{Primary: StoreRedis, Tcerts: &TcertsExport{}}},
		{desc: "tcerts without prefix", store: &CertStore{Primary: StoreRedis, Tcerts: &TcertsExport{Enabled: true}}, expectErr: true},
		{desc: "tcerts prefix with a colon", store: &CertStore{Primary: StoreRedis, Tcerts: &TcertsExport{Enabled: true, Prefix: "a:b"}}, expectErr: true},
		{desc: "election", store: &CertStore{Primary: StoreRedis, Election: &Election{Step: ptypes.Duration(time.Second), LockTTL: ptypes.Duration(30 * time.Second)}}},
		{desc: "election without step", store: &CertStore{Primary: StoreRedis, Election: &Election{LockTTL: ptypes.Duration(30 * time.Second)}}, expectErr: true},
		{desc: "election with a short lockTTL", store: &CertStore{Primary: StoreRedis, Election: &Election{Step: ptypes.Duration(time.Second), LockTTL: ptypes.Duration(time.Second)}}, expectErr: true},
		{desc: "redis as an export", store: &CertStore{Primary: StoreFile, Export: []string{StoreRedis}}, expectErr: true},
		{desc: "unknown primary", store: &CertStore{Primary: "s3"}, expectErr: true},
		{desc: "unknown export", store: &CertStore{Primary: StoreFile, Export: []string{"s3"}}, expectErr: true},
		{desc: "export of the primary", store: &CertStore{Primary: StoreFile, Export: []string{StoreFile}}, expectErr: true},
		{desc: "export listed twice", store: &CertStore{Primary: StoreFile, Dir: "/certs", Export: []string{StoreDir, StoreDir}}, expectErr: true},
		{desc: "dir primary without dir", store: &CertStore{Primary: StoreDir}, expectErr: true},
		{desc: "dir export without dir", store: &CertStore{Primary: StoreFile, Export: []string{StoreDir}}, expectErr: true},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			err := test.store.Validate()
			if test.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
