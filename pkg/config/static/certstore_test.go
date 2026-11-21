package static

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/provider/acme"
)

func TestValidateConfiguration_certStore(t *testing.T) {
	testCases := []struct {
		desc      string
		certStore *acme.CertStore
		expectErr bool
	}{
		{desc: "no certStore"},
		{desc: "dir with file export", certStore: &acme.CertStore{Primary: acme.StoreDir, Dir: "/certs", Export: []string{acme.StoreFile}}},
		{desc: "dir without its location", certStore: &acme.CertStore{Primary: acme.StoreDir}, expectErr: true},
		{desc: "unknown store", certStore: &acme.CertStore{Primary: "s3"}, expectErr: true},
		{desc: "redis without the cluster option", certStore: &acme.CertStore{Primary: acme.StoreRedis}, expectErr: true},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			cfg := &Configuration{
				CertificatesResolvers: map[string]CertificateResolver{
					"le": {ACME: &acme.Configuration{Storage: "acme.json", CertStore: test.certStore}},
				},
			}

			err := cfg.ValidateConfiguration()
			if test.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
