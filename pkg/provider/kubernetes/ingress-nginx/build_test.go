package ingressnginx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func Test_basicAuthUsers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		desc           string
		authSecretType string
		data           map[string][]byte
		expected       []secretUser
	}{
		{
			desc:           "auth-file users are read from the auth key, without comments and blank lines",
			authSecretType: "auth-file",
			data:           map[string][]byte{"auth": []byte("# comment\nuser1:hash1\n\n user2:hash2 \n")},
			expected: []secretUser{
				{key: "auth", value: "user1:hash1"},
				{key: "auth", value: "user2:hash2"},
			},
		},
		{
			desc:           "auth-map users are sorted by key",
			authSecretType: "auth-map",
			data:           map[string][]byte{"user2": []byte("hash2"), "user1": []byte("hash1"), "user3": []byte("hash3")},
			expected: []secretUser{
				{key: "user1", value: "user1:hash1"},
				{key: "user2", value: "user2:hash2"},
				{key: "user3", value: "user3:hash3"},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			users, err := basicAuthUsers(&corev1.Secret{Data: test.data}, test.authSecretType)
			require.NoError(t, err)

			assert.Equal(t, test.expected, users)
		})
	}
}

func TestProvider_resolveBasicAuth_authMap(t *testing.T) {
	t.Parallel()

	kubeClient := kubefake.NewClientset(&corev1.Secret{
		Name: "basic-auth", Namespace: "default",
		Data: map[string][]byte{"user2": []byte("hash2"), "user1": []byte("hash1")},
	})
	client := newClient(kubeClient)

	_, err := client.WatchAll(t.Context(), "", "")
	require.NoError(t, err)

	p := Provider{k8sClient: client}

	basic, digest, urnRefs, err := p.resolveBasicAuth("default", IngressConfig{
		AuthType:       new("basic"),
		AuthSecret:     new("basic-auth"),
		AuthSecretType: new("auth-map"),
	})
	require.NoError(t, err)

	assert.Equal(t, &dynamic.BasicAuth{Users: dynamic.Users{"urn:k8s:secret:basic-auth:user1", "urn:k8s:secret:basic-auth:user2"}}, basic)
	assert.Nil(t, digest)
	assert.Equal(t, map[string]string{"/basicAuth/users/0": "user1:hash1", "/basicAuth/users/1": "user2:hash2"}, urnRefs)
}

func TestBackendEndpointsCacheKey(t *testing.T) {
	backend := netv1.IngressBackend{
		Service: &netv1.IngressServiceBackend{
			Name: "whoami",
			Port: netv1.ServiceBackendPort{Name: "http"},
		},
	}

	base := backendEndpointsCacheKey("testing", backend, IngressConfig{})

	// The same service and config must reuse the same cache entry.
	assert.Equal(t, base, backendEndpointsCacheKey("testing", backend, IngressConfig{}))

	// getBackendEndpoints branches on ServiceUpstream and DefaultBackend, so those must not
	// collide with the base entry for the same service.
	assert.NotEqual(t, base, backendEndpointsCacheKey("testing", backend, IngressConfig{ServiceUpstream: new(true)}))
	assert.NotEqual(t, base, backendEndpointsCacheKey("testing", backend, IngressConfig{DefaultBackend: new("fallback")}))

	// A different port of the same service is a distinct backend.
	otherPort := netv1.IngressBackend{
		Service: &netv1.IngressServiceBackend{
			Name: "whoami",
			Port: netv1.ServiceBackendPort{Name: "https"},
		},
	}
	assert.NotEqual(t, base, backendEndpointsCacheKey("testing", otherPort, IngressConfig{}))

	// A backend without a Service cannot be keyed and must bypass the cache.
	assert.Empty(t, backendEndpointsCacheKey("testing", netv1.IngressBackend{}, IngressConfig{}))
}

func TestLoadCertificatesContinuesAfterMissingSecret(t *testing.T) {
	t.Parallel()

	const namespace = "default"

	validFirst := &corev1.Secret{
		Name: "valid-first", Namespace: namespace,
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("first-certificate"),
			corev1.TLSPrivateKeyKey: []byte("first-key"),
		},
	}
	validLast := &corev1.Secret{
		Name: "valid-last", Namespace: namespace,
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("last-certificate"),
			corev1.TLSPrivateKeyKey: []byte("last-key"),
		},
	}

	kubeClient := kubefake.NewClientset(validFirst, validLast)
	client := newClient(kubeClient)
	_, err := client.WatchAll(t.Context(), "", "")
	require.NoError(t, err)

	provider := Provider{k8sClient: client}
	ingress := &netv1.Ingress{
		Name: "testing", Namespace: namespace,
		Spec: netv1.IngressSpec{
			TLS: []netv1.IngressTLS{
				{SecretName: validFirst.Name},
				{SecretName: "missing"},
				{SecretName: validLast.Name},
			},
		},
	}

	certs := make(map[string]certPair)
	loaded := make(map[string]bool)
	err = provider.loadCertificates(t.Context(), ingress, certs, loaded)

	require.ErrorContains(t, err, "default/missing")
	assert.Equal(t, map[string]certPair{
		"default/valid-first": {Cert: "first-certificate", Key: "first-key"},
		"default/valid-last":  {Cert: "last-certificate", Key: "last-key"},
	}, certs)
	assert.Equal(t, map[string]bool{
		"default/valid-first": true,
		"default/valid-last":  true,
	}, loaded)
}
