package coraza

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/label"
)

// The labels split a list on commas, which the SecRule directives contain: the plugin form, whose values are
// untyped, takes an indexed list instead.
func TestConfigFromPlugin_labels(t *testing.T) {
	cfg, err := label.DecodeConfiguration(map[string]string{
		"traefik.http.middlewares.waf.plugin.coraza.crsEnabled":    "true",
		"traefik.http.middlewares.waf.plugin.coraza.directives[0]": "SecRuleEngine On",
		"traefik.http.middlewares.waf.plugin.coraza.directives[1]": `SecRule REQUEST_URI "@streq /admin" "id:101,phase:1,deny,status:403"`,
	})
	require.NoError(t, err)

	coraza, err := ConfigFromPlugin(cfg.HTTP.Middlewares["waf"].Plugin["coraza"])
	require.NoError(t, err)
	assert.Equal(t, dynamic.Coraza{
		CRSEnabled: true,
		Directives: []string{"SecRuleEngine On", `SecRule REQUEST_URI "@streq /admin" "id:101,phase:1,deny,status:403"`},
	}, coraza)
}
