package acme

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
)

const (
	// StoreFile is the single JSON file of the storage option, acme.json by default.
	StoreFile = "file"
	// StoreDir is the directory with one sub-directory of PEM files per domain.
	StoreDir = "dir"
	// StoreRedis is the Valkey or Redis server of the cluster option, shared by the instances.
	StoreRedis = "redis"
)

// CertStore configures where a certificate resolver keeps its account and certificates (Traefix).
type CertStore struct {
	Primary string        `description:"Source of truth of the ACME account and certificates: file (the storage option), dir, or redis (the server of the cluster option)." json:"primary,omitempty" toml:"primary,omitempty" yaml:"primary,omitempty" export:"true"`
	Dir     string        `description:"Directory of the per-domain certificate files, required when dir is the primary store or an export." json:"dir,omitempty" toml:"dir,omitempty" yaml:"dir,omitempty" export:"true"`
	Export  []string      `description:"Stores the primary store is copied to: dir, file." json:"export,omitempty" toml:"export,omitempty" yaml:"export,omitempty" export:"true"`
	Tcerts  *TcertsExport `description:"Copy of the certificates in the Redis server, in the format of the traefik-certs project. Used with the redis primary store." json:"tcerts,omitempty" toml:"tcerts,omitempty" yaml:"tcerts,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
}

// TcertsExport configures the copy of the certificates that the tools of the traefik-certs project read.
type TcertsExport struct {
	Enabled bool   `description:"Writes each certificate to the <prefix>:cert:<domain> keys, and its domains to the <prefix>:events channel." json:"enabled" toml:"enabled" yaml:"enabled" export:"true"`
	Prefix  string `description:"Prefix of the keys and of the channel." json:"prefix,omitempty" toml:"prefix,omitempty" yaml:"prefix,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (t *TcertsExport) SetDefaults() {
	t.Enabled = true
	t.Prefix = defaultTcertsPrefix
}

const defaultTcertsPrefix = "tcerts"

// SetDefaults sets the default values.
func (c *CertStore) SetDefaults() {
	c.Primary = StoreFile
}

// Validate checks that the stores are known, used once, and have their location.
func (c *CertStore) Validate() error {
	if c == nil {
		return nil
	}

	if !isLocalStore(c.Primary) && c.Primary != StoreRedis {
		return fmt.Errorf("unknown certStore primary %q, expected %q, %q or %q", c.Primary, StoreFile, StoreDir, StoreRedis)
	}

	if c.Tcerts != nil && c.Tcerts.Enabled && !tcertsPrefixRegexp.MatchString(c.Tcerts.Prefix) {
		return fmt.Errorf("certStore tcerts prefix %q must start with a letter or a digit, followed by letters, digits, dots, underscores or hyphens", c.Tcerts.Prefix)
	}

	for i, export := range c.Export {
		if !isLocalStore(export) {
			return fmt.Errorf("unknown certStore export %q, expected %q or %q", export, StoreFile, StoreDir)
		}

		if export == c.Primary {
			return fmt.Errorf("certStore export %q is already the primary store", export)
		}

		if slices.Contains(c.Export[:i], export) {
			return fmt.Errorf("certStore export %q is listed twice", export)
		}
	}

	if c.Dir == "" && (c.Primary == StoreDir || slices.Contains(c.Export, StoreDir)) {
		return errors.New("certStore dir is required when dir is the primary store or an export")
	}

	return nil
}

// tcertsPrefix returns the prefix of the traefik-certs copy, or an empty string when the copy is disabled.
// The copy is on by default: the option is only needed to turn it off or to change its prefix.
func (c *CertStore) tcertsPrefix() string {
	switch {
	case c.Tcerts == nil:
		return defaultTcertsPrefix
	case !c.Tcerts.Enabled:
		return ""
	default:
		return c.Tcerts.Prefix
	}
}

var tcertsPrefixRegexp = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func isLocalStore(kind string) bool {
	return kind == StoreFile || kind == StoreDir
}
