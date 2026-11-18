package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/runtime"
)

// declaredPluginsBuilder is a plugins builder declaring the plugins of the given names.
type declaredPluginsBuilder struct {
	fakePluginsBuilder

	declared map[string]bool
}

func (b *declaredPluginsBuilder) HasPlugin(name string) bool {
	return b.declared[name]
}

func TestBuilder_coraza(t *testing.T) {
	directives := []any{"SecRuleEngine On", `SecRule REQUEST_URI "@streq /admin" "id:101,phase:1,deny,status:403,log"`}

	testCases := []struct {
		desc           string
		middleware     *dynamic.Middleware
		pluginsBuilder PluginsBuilder
		expected       int
		pluginBuilt    bool
	}{
		{
			desc:       "native",
			middleware: &dynamic.Middleware{Coraza: &dynamic.Coraza{Directives: []string{"SecRuleEngine On", `SecRule REQUEST_URI "@streq /admin" "id:101,phase:1,deny,status:403,log"`}}},
			expected:   http.StatusForbidden,
		},
		{
			desc:       "plugin form without plugins",
			middleware: &dynamic.Middleware{Plugin: map[string]dynamic.PluginConf{"coraza": {"directives": directives}}},
			expected:   http.StatusForbidden,
		},
		{
			desc:           "plugin form with other plugins",
			middleware:     &dynamic.Middleware{Plugin: map[string]dynamic.PluginConf{"coraza": {"directives": directives}}},
			pluginsBuilder: &declaredPluginsBuilder{declared: map[string]bool{"other": true}},
			expected:       http.StatusForbidden,
		},
		{
			desc:           "a declared coraza plugin keeps the plugin",
			middleware:     &dynamic.Middleware{Plugin: map[string]dynamic.PluginConf{"coraza": {"directives": directives}}},
			pluginsBuilder: &declaredPluginsBuilder{declared: map[string]bool{"coraza": true}},
			expected:       http.StatusOK,
			pluginBuilt:    true,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			rtConf := runtime.NewConfig(dynamic.Configuration{
				HTTP: &dynamic.HTTPConfiguration{Middlewares: map[string]*dynamic.Middleware{"waf@file": test.middleware}},
			})

			builder := NewBuilder(rtConf.Middlewares, nil, test.pluginsBuilder)
			handler, err := builder.BuildMiddlewareChain(t.Context(), []string{"waf@file"}).
				Then(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) { rw.WriteHeader(http.StatusOK) }))
			require.NoError(t, err)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://example.com/admin", nil))
			assert.Equal(t, test.expected, rec.Code)

			if pb, ok := test.pluginsBuilder.(*declaredPluginsBuilder); ok {
				assert.Equal(t, test.pluginBuilt, pb.config != nil)
			}
		})
	}
}
