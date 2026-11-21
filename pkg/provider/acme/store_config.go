package acme

import (
	"errors"
	"fmt"
	"slices"
)

const (
	// StoreFile is the single JSON file of the storage option, acme.json by default.
	StoreFile = "file"
	// StoreDir is the directory with one sub-directory of PEM files per domain.
	StoreDir = "dir"
)

// CertStore configures where a certificate resolver keeps its account and certificates (Traefix).
type CertStore struct {
	Primary string   `description:"Source of truth of the ACME account and certificates: file (the storage option) or dir." json:"primary,omitempty" toml:"primary,omitempty" yaml:"primary,omitempty" export:"true"`
	Dir     string   `description:"Directory of the per-domain certificate files, required when dir is the primary store or an export." json:"dir,omitempty" toml:"dir,omitempty" yaml:"dir,omitempty" export:"true"`
	Export  []string `description:"Stores the primary store is copied to: dir, file." json:"export,omitempty" toml:"export,omitempty" yaml:"export,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (c *CertStore) SetDefaults() {
	c.Primary = StoreFile
}

// Validate checks that the stores are known, used once, and have their location.
func (c *CertStore) Validate() error {
	if c == nil {
		return nil
	}

	if !isStoreKind(c.Primary) {
		return fmt.Errorf("unknown certStore primary %q, expected %q or %q", c.Primary, StoreFile, StoreDir)
	}

	for i, export := range c.Export {
		if !isStoreKind(export) {
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

func isStoreKind(kind string) bool {
	return kind == StoreFile || kind == StoreDir
}
