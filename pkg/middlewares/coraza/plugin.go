package coraza

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
)

// PluginName is the plugin.coraza key of Traefik Hub and of the Coraza WASM plugin, read as the native middleware
// when no plugin of that name is declared.
const PluginName = "coraza"

// ConfigFromPlugin reads the plugin.coraza configuration, whose values are untyped: the providers give strings,
// e.g. the labels, or lists and booleans, e.g. the files.
func ConfigFromPlugin(raw dynamic.PluginConf) (dynamic.Coraza, error) {
	var cfg dynamic.Coraza

	for key, value := range raw {
		var err error
		switch strings.ToLower(key) {
		case "directives":
			cfg.Directives, err = toStrings(value)
		case "crsenabled":
			cfg.CRSEnabled, err = toBool(value)
		case "failopen":
			cfg.FailOpen, err = toBool(value)
		default:
			err = fmt.Errorf("unknown option %q", key)
		}
		if err != nil {
			return dynamic.Coraza{}, fmt.Errorf("plugin.coraza: %s: %w", key, err)
		}
	}

	return cfg, nil
}

func toStrings(value any) ([]string, error) {
	switch v := value.(type) {
	case []string:
		return v, nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected %T in the list", item)
			}
			out = append(out, s)
		}
		return out, nil
	case map[string]any:
		// The labels give a list as a map indexed by position.
		out := make([]string, len(v))
		for k, item := range v {
			i, err := strconv.Atoi(k)
			if err != nil || i < 0 || i >= len(v) {
				return nil, fmt.Errorf("unexpected index %q", k)
			}
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected %T in the list", item)
			}
			out[i] = s
		}
		return out, nil
	case string:
		// A single string holds one directive per line.
		var out []string
		for line := range strings.SplitSeq(v, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				out = append(out, line)
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unexpected %T", value)
	}
}

func toBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	default:
		return false, fmt.Errorf("unexpected %T", value)
	}
}
