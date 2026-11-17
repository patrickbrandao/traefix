package crd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	traefikv1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Test_serviceOptionsHash(t *testing.T) {
	t.Parallel()

	identityOnly := traefikv1alpha1.LoadBalancerSpec{
		Name:      "whoami",
		Kind:      "Service",
		Namespace: "default",
		Port:      intstr.FromInt(80),
		Weight:    new(1),
	}

	stickyFoo := identityOnly
	stickyFoo.Sticky = &dynamic.Sticky{
		Cookie: &dynamic.Cookie{
			Name:     "cookie",
			Secure:   true,
			HTTPOnly: true,
			SameSite: "none",
			MaxAge:   42,
			Path:     new("/foo"),
		},
	}

	stickyRoot := stickyFoo
	stickyRoot.Sticky = &dynamic.Sticky{
		Cookie: &dynamic.Cookie{
			Name:     "cookie",
			Secure:   true,
			HTTPOnly: true,
			SameSite: "none",
			MaxAge:   42,
		},
	}

	hash, err := serviceOptionsHash(identityOnly)
	require.NoError(t, err)
	assert.Empty(t, hash)

	fooHash, err := serviceOptionsHash(stickyFoo)
	require.NoError(t, err)
	assert.Equal(t, "7122ba6b63cac426", fooHash)

	rootHash, err := serviceOptionsHash(stickyRoot)
	require.NoError(t, err)
	assert.Equal(t, "2e2d91e1e1ae5b98", rootHash)
	assert.NotEqual(t, fooHash, rootHash)

	assert.Equal(t, "default-whoami-80-7122ba6b63cac426", appendServiceOptionsHash("default-whoami-80", stickyFoo))
	assert.Equal(t, "default-whoami-80", appendServiceOptionsHash("default-whoami-80", identityOnly))

	traefikService := stickyFoo
	traefikService.Kind = "TraefikService"
	traefikService.Port = intstr.IntOrString{}
	assert.Equal(t, "default-whoami", appendServiceOptionsHash("default-whoami", traefikService))
}
