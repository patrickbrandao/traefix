package acme

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
	traefiktls "github.com/traefik/traefik/v3/pkg/tls"
	"github.com/traefik/traefik/v3/pkg/types"
)

const (
	// dirStoreMeta is the sub-directory of the data that is not a certificate. Its name cannot be a domain.
	dirStoreMeta = "_traefix"
	// dirStoreWildcardPrefix names the directory of a wildcard-only certificate
	// when another certificate holds the exact name.
	dirStoreWildcardPrefix = "_wildcard."

	dirStoreCertJSON = "cert.json"
	dirStoreTmp      = ".tmp-"
)

var _ Store = (*DirStore)(nil)

// DirStore keeps each certificate as PEM files in one directory per domain,
// with the layout of the traefik-certs project:
// fullchain.pem, cert.pem, chain.pem, privkey.pem, cert.md5 and cert.json.
type DirStore struct {
	dir string
	now func() time.Time

	mu     sync.Mutex
	loaded bool
	// certs holds the certificates of each resolver, as read from the directory or last saved.
	certs map[string][]*CertAndStore
	// written describes the cert.json of each directory, to skip the writes that change nothing.
	written map[string]dirEntry
	// unreadable lists the directories with no usable cert.json.
	unreadable []string
}

type dirEntry struct {
	signature string
	sans      string
	notAfter  int64
}

// NewDirStore creates a DirStore on a directory, which is created on the first write.
func NewDirStore(dir string) *DirStore {
	return &DirStore{dir: dir, now: time.Now}
}

// GetAccount returns the ACME account of a resolver.
func (s *DirStore) GetAccount(resolverName string) (*Account, error) {
	path, err := s.accountPath(resolverName)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	account := &Account{}
	//nolint:musttag // Account keeps the untagged encoding it has in the storage file.
	if err := json.Unmarshal(data, account); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return account, nil
}

// SaveAccount stores the ACME account of a resolver.
func (s *DirStore) SaveAccount(resolverName string, account *Account) error {
	path, err := s.accountPath(resolverName)
	if err != nil {
		return err
	}

	//nolint:musttag // Account keeps the untagged encoding it has in the storage file.
	data, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}

	// The account holds its private key: no access for group and others, on the way down too.
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	return writeFileAtomic(filepath.Dir(path), filepath.Base(path), data, 0o600, true)
}

// GetCertificates returns the certificates of a resolver.
func (s *DirStore) GetCertificates(resolverName string) ([]*CertAndStore, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.load(); err != nil {
		return nil, err
	}

	return slices.Clone(s.certs[resolverName]), nil
}

// SaveCertificates stores the certificates of a resolver.
func (s *DirStore) SaveCertificates(resolverName string, certificates []*CertAndStore) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.load(); err != nil {
		return err
	}

	s.certs[resolverName] = slices.Clone(certificates)

	return s.sync()
}

// ReportUnmanaged logs the content of the directory that no configured resolver manages.
func (s *DirStore) ReportUnmanaged(resolverNames []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	logger := log.With().Str(logs.ProviderName, "acme").Str("dir", s.dir).Logger()

	if err := s.load(); err != nil {
		logger.Error().Err(err).Msg("Unable to read the certificates directory")
		return
	}

	for _, resolverName := range slices.Sorted(maps.Keys(s.certs)) {
		if slices.Contains(resolverNames, resolverName) || len(s.certs[resolverName]) == 0 {
			continue
		}

		logger.Warn().Str("resolver", resolverName).Int("certificates", len(s.certs[resolverName])).
			Msg("Certificates of an unknown certificate resolver are neither served nor renewed")
	}

	if len(s.unreadable) > 0 {
		logger.Warn().Strs("directories", s.unreadable).Msg("Directories without a usable cert.json are ignored")
	}
}

func (s *DirStore) accountPath(resolverName string) (string, error) {
	if !isDirStoreName(resolverName) {
		return "", fmt.Errorf("certificate resolver name %q cannot be used as a directory name", resolverName)
	}

	return filepath.Join(s.dir, dirStoreMeta, resolverName, "account.json"), nil
}

// load reads the directory once. The mutex must be held.
func (s *DirStore) load() error {
	if s.loaded {
		return nil
	}

	entries, err := os.ReadDir(s.dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	certs := map[string][]*CertAndStore{}
	written := map[string]dirEntry{}
	var unreadable []string

	docs := map[string]*certJSON{}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || name == dirStoreMeta || strings.HasPrefix(name, ".") {
			continue
		}

		s.removeTemporaryFiles(name)

		doc, err := s.readCertJSON(name)
		if err != nil {
			unreadable = append(unreadable, name)
			continue
		}

		docs[name] = doc
		written[name] = newDirEntry(doc)
	}

	// The files written by Traefix come first: they know the resolver, the domain and the TLS store,
	// which the files of traefik-certs for the same certificate can only guess.
	names := slices.SortedFunc(maps.Keys(docs), func(a, b string) int {
		if byTraefix := cmp.Compare(boolToInt(docs[b].Traefix != nil), boolToInt(docs[a].Traefix != nil)); byTraefix != 0 {
			return byTraefix
		}

		return cmp.Compare(a, b)
	})

	// A certificate with several names is in several directories: it is read once.
	seen := map[string]struct{}{}

	for _, name := range names {
		doc := docs[name]

		resolverName, cert, err := doc.toCertAndStore()
		if err != nil {
			unreadable = append(unreadable, name)
			continue
		}

		key := resolverName + "|" + doc.Hash.Fullchain + "|" + doc.Hash.Privkey
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		certs[resolverName] = append(certs[resolverName], cert)
	}

	s.certs = certs
	s.written = written
	s.unreadable = unreadable
	s.loaded = true

	return nil
}

func (s *DirStore) readCertJSON(name string) (*certJSON, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, name, dirStoreCertJSON))
	if err != nil {
		return nil, err
	}

	doc := &certJSON{}
	if err := json.Unmarshal(data, doc); err != nil {
		return nil, err
	}

	return doc, nil
}

// removeTemporaryFiles deletes what an interrupted write left behind.
func (s *DirStore) removeTemporaryFiles(name string) {
	files, err := os.ReadDir(filepath.Join(s.dir, name))
	if err != nil {
		return
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), dirStoreTmp) {
			_ = os.Remove(filepath.Join(s.dir, name, file.Name()))
		}
	}
}

// toCertAndStore rebuilds the record of the ACME provider from a cert.json.
func (c *certJSON) toCertAndStore() (string, *CertAndStore, error) {
	if c.PEM.Fullchain == "" || c.PEM.Privkey == "" {
		return "", nil, errors.New("no certificate or no private key")
	}

	if _, _, _, err := decomposePEM(c.PEM.Fullchain); err != nil {
		return "", nil, err
	}

	cert := &CertAndStore{
		Certificate: Certificate{
			Certificate: []byte(c.PEM.Fullchain),
			Key:         []byte(c.PEM.Privkey),
		},
		Store: traefiktls.DefaultTLSStoreName,
	}

	if c.Traefix != nil {
		cert.Domain = types.Domain{Main: c.Traefix.Main, SANs: c.Traefix.SANs}
		if c.Traefix.Store != "" {
			cert.Store = c.Traefix.Store
		}

		return c.Traefix.Resolver, cert, nil
	}

	// A file written by traefik-certs only knows the names of the certificate.
	if len(c.SANs) == 0 {
		return "", nil, errors.New("no domain")
	}

	sans := slices.Clone(c.SANs)
	if i := slices.Index(sans, c.SubjectCN); i > 0 {
		sans = slices.Insert(slices.Delete(sans, i, i+1), 0, c.SubjectCN)
	}
	cert.Domain.Set(sans)

	return c.AcmeResolver, cert, nil
}

// dirClaim is a certificate that wants a directory.
type dirClaim struct {
	resolverName string
	cert         *CertAndStore
	notAfter     time.Time
}

// sync writes the directories that differ from the certificates. The mutex must be held.
func (s *DirStore) sync() error {
	var errs []error

	layout := certLayout(s.certs)
	for _, name := range slices.Sorted(maps.Keys(layout)) {
		claim := layout[name]

		doc, err := newCertJSON(name, claim.resolverName, claim.cert, s.now())
		if err != nil {
			errs = append(errs, fmt.Errorf("certificate of %s: %w", name, err))
			continue
		}

		next := newDirEntry(doc)

		current, ok := s.written[name]
		if ok && current.signature == next.signature {
			continue
		}

		// The rule of traefik-certs: a certificate never replaces a more recent one for the same names.
		if ok && current.sans == next.sans && current.notAfter > next.notAfter {
			continue
		}

		if err := s.writeCertFiles(name, doc); err != nil {
			errs = append(errs, fmt.Errorf("writing the files of %s: %w", name, err))
			continue
		}

		s.written[name] = next
	}

	return errors.Join(errs...)
}

// certLayout gives each directory its certificate, among the certificates of every resolver.
// A domain and its wildcard share a directory, as in traefik-certs.
// When two certificates want it, the one with the exact name keeps it
// and the wildcard-only one goes to a directory of its own, so that none is lost.
func certLayout(certs map[string][]*CertAndStore) map[string]*dirClaim {
	exact := map[string]*dirClaim{}
	wildcard := map[string]*dirClaim{}

	for _, resolverName := range slices.Sorted(maps.Keys(certs)) {
		for _, cert := range certs[resolverName] {
			_, _, leaf, err := decomposePEM(string(cert.Certificate.Certificate))
			if err != nil {
				log.Error().Err(err).Str(logs.ProviderName, "acme").Strs("domains", cert.Domain.ToStrArray()).
					Msg("Unable to read a certificate, it is not written to the per-domain certificate files or keys")
				continue
			}

			claim := &dirClaim{resolverName: resolverName, cert: cert, notAfter: leaf.NotAfter}

			for _, domain := range cert.Domain.ToStrArray() {
				name := strings.TrimPrefix(domain, "*.")
				if !isDirStoreName(name) {
					continue
				}

				claims := exact
				if isWildcard(domain) {
					claims = wildcard
				}

				if current, ok := claims[name]; !ok || claim.notAfter.After(current.notAfter) {
					claims[name] = claim
				}
			}
		}
	}

	layout := exact
	for name, claim := range wildcard {
		holder, ok := layout[name]
		switch {
		case !ok:
			layout[name] = claim
		case holder.cert != claim.cert:
			layout[dirStoreWildcardPrefix+name] = claim
		}
	}

	return layout
}

// writeCertFiles writes the files of a directory.
// cert.json comes last: it is what the consumers watch, and it must never describe files that are not there yet.
func (s *DirStore) writeCertFiles(name string, doc *certJSON) error {
	dir := filepath.Join(s.dir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	docData, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}

	files := []struct {
		name string
		data string
		perm os.FileMode
	}{
		{name: "fullchain.pem", data: doc.PEM.Fullchain, perm: 0o644},
		{name: "cert.pem", data: doc.PEM.Cert, perm: 0o644},
		{name: "chain.pem", data: doc.PEM.Chain, perm: 0o644},
		{name: "privkey.pem", data: doc.PEM.Privkey, perm: 0o600},
		{name: "cert.md5", data: doc.CertMD5 + "\n", perm: 0o644},
	}

	for _, file := range files {
		if err := writeFileAtomic(dir, file.name, []byte(file.data), file.perm, false); err != nil {
			return err
		}
	}

	return writeFileAtomic(dir, dirStoreCertJSON, docData, 0o600, true)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}

	return 0
}

func newDirEntry(doc *certJSON) dirEntry {
	sans := slices.Clone(doc.SANs)
	slices.Sort(sans)

	return dirEntry{signature: doc.signature(), sans: strings.Join(sans, ","), notAfter: doc.NotAfterUnix}
}

// isDirStoreName tells whether a domain or resolver name is safe as a directory name.
// The leading underscore and dot are kept for the directories of the store itself.
func isDirStoreName(name string) bool {
	return name != "" &&
		!strings.HasPrefix(name, "_") &&
		!strings.HasPrefix(name, ".") &&
		!strings.ContainsAny(name, `/\`) &&
		name == filepath.Base(name)
}

// writeFileAtomic writes a file through a temporary file and a rename,
// so that a reader never sees half of a PEM file.
func writeFileAtomic(dir, name string, data []byte, perm os.FileMode, durable bool) error {
	tmp, err := os.CreateTemp(dir, dirStoreTmp+name+"-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}

	if durable {
		if err := tmp.Sync(); err != nil {
			_ = tmp.Close()
			return err
		}
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), filepath.Join(dir, name))
}
