package dynamic

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/paerser/file"
	ptypes "github.com/traefik/paerser/types"
)

func TestForwardAuth_defaultTimeout(t *testing.T) {
	testCases := []struct {
		desc     string
		content  string
		expected ptypes.Duration
	}{
		{
			desc:     "unset",
			content:  "http:\n  middlewares:\n    auth:\n      forwardAuth:\n        address: http://auth\n",
			expected: ForwardAuthDefaultTimeout,
		},
		{
			desc:     "set",
			content:  "http:\n  middlewares:\n    auth:\n      forwardAuth:\n        address: http://auth\n        timeout: 2m\n",
			expected: ptypes.Duration(2 * time.Minute),
		},
		{
			desc:     "disabled",
			content:  "http:\n  middlewares:\n    auth:\n      forwardAuth:\n        address: http://auth\n        timeout: 0\n",
			expected: 0,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			cfg := &Configuration{}
			require.NoError(t, file.DecodeContent(test.content, ".yaml", cfg))

			assert.Equal(t, test.expected, cfg.HTTP.Middlewares["auth"].ForwardAuth.Timeout)
		})
	}
}

func Test_GetStrategy_ipv6Subnet(t *testing.T) {
	testCases := []struct {
		desc        string
		expectError bool
		ipv6Subnet  *int
	}{
		{
			desc: "Nil subnet",
		},
		{
			desc:        "Zero subnet",
			expectError: true,
			ipv6Subnet:  new(0),
		},
		{
			desc:        "Subnet greater that 128",
			expectError: true,
			ipv6Subnet:  new(129),
		},
		{
			desc:       "Valid subnet",
			ipv6Subnet: new(128),
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			strategy := IPStrategy{
				IPv6Subnet: test.ipv6Subnet,
			}

			get, err := strategy.Get()
			if test.expectError {
				require.Error(t, err)
				assert.Nil(t, get)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, get)
			}
		})
	}
}

func TestHasSecureHeadersDefined(t *testing.T) {
	testCases := []struct {
		desc     string
		headers  *Headers
		expected bool
	}{
		{
			desc:     "Nil headers",
			headers:  nil,
			expected: false,
		},
		{
			desc:     "Empty headers",
			headers:  &Headers{},
			expected: false,
		},
		{
			desc: "STSSeconds set to non-zero",
			headers: &Headers{
				STSSeconds: new(int64(42)),
			},
			expected: true,
		},
		{
			desc: "STSSeconds set to zero",
			headers: &Headers{
				STSSeconds: new(int64(0)),
			},
			expected: true,
		},
		{
			desc: "STSSeconds nil (not set)",
			headers: &Headers{
				FrameDeny: true,
			},
			expected: true,
		},
		{
			desc: "Only ForceSTSHeader",
			headers: &Headers{
				ForceSTSHeader: true,
			},
			expected: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			result := test.headers.HasSecureHeadersDefined()
			assert.Equal(t, test.expected, result)
		})
	}
}
