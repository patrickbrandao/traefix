package middleware

import (
	"reflect"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	corazamw "github.com/traefik/traefik/v3/pkg/middlewares/coraza"
)

// corazaConfig returns the configuration of the native Coraza middleware, given as coraza, or as plugin.coraza
// when no plugin of that name is declared, as Traefik Hub reads it. The second value reports the plugin form.
func (b *Builder) corazaConfig(config *dynamic.Middleware) (*dynamic.Coraza, bool, error) {
	if config.Coraza != nil {
		return config.Coraza, false, nil
	}

	raw, ok := config.Plugin[corazamw.PluginName]
	if !ok || len(config.Plugin) != 1 {
		return nil, false, nil
	}

	// The plugin builder is an interface that may hold a nil pointer, as the plugin block checks.
	if b.pluginBuilder != nil && !reflect.ValueOf(b.pluginBuilder).IsNil() {
		if pb, ok := b.pluginBuilder.(interface{ HasPlugin(name string) bool }); ok && pb.HasPlugin(corazamw.PluginName) {
			return nil, false, nil
		}
	}

	cfg, err := corazamw.ConfigFromPlugin(raw)
	if err != nil {
		return nil, false, err
	}

	return &cfg, true, nil
}
