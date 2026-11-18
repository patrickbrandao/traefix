package coraza

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	coreruleset "github.com/corazawaf/coraza-coreruleset/v4"
)

// rulesFS resolves the Include paths: the @ names to the embedded Core Rule Set when crsEnabled is set,
// as Traefik Hub does, and the other paths to the files of the host, so that the rules can live on disk.
type rulesFS struct {
	crs fs.FS
}

func newRulesFS(crsEnabled bool) fs.FS {
	if crsEnabled {
		return rulesFS{crs: coreruleset.FS}
	}
	return rulesFS{}
}

func (f rulesFS) Open(name string) (fs.File, error) {
	if strings.HasPrefix(name, "@") {
		if f.crs == nil {
			return nil, &fs.PathError{Op: "open", Path: name, Err: errors.New("the embedded Core Rule Set requires crsEnabled")}
		}
		return f.crs.Open(name)
	}

	file, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("opening rules file: %w", err)
	}
	return file, nil
}
