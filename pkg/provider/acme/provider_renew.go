package acme

import (
	"context"

	"github.com/go-acme/lego/v5/acme"
	"github.com/go-acme/lego/v5/certificate"
	"github.com/rs/zerolog/log"
)

type renewFunc func(ctx context.Context, res certificate.Resource, opts *certificate.RenewOptions) (*certificate.Resource, error)

// renewCertificate renews a certificate, and does so again without naming the certificate it replaces when the ACME
// server refuses that link (Traefix).
// The server only accepts the link from the account that obtained the certificate (RFC 9773, section 5),
// which is not the account of the resolver for a certificate that came from the storage of another instance.
func renewCertificate(ctx context.Context, renew renewFunc, res certificate.Resource, opts *certificate.RenewOptions) (*certificate.Resource, error) {
	renewed, err := renew(ctx, res, opts)
	if err == nil || !opts.UseARICertID || !isOrderRefused(err) {
		return renewed, err
	}

	log.Ctx(ctx).Info().Err(err).Strs("domains", res.Domains).
		Msg("The ACME server refuses to replace the certificate by its renewal, renewing it as a new certificate")

	plain := *opts
	plain.UseARICertID = false

	return renew(ctx, res, &plain)
}

// isOrderRefused tells whether the ACME server refused the order itself.
// Such a refusal comes as a bare problem, before any validation, so another order costs nothing.
// A failed validation, which can have the same problem type, comes wrapped with its domain:
// it must not be tried twice, as the ACME server limits the failed validations.
func isOrderRefused(err error) bool {
	_, ok := err.(*acme.ProblemDetails) //nolint:errorlint // A wrapped problem is not a refusal of the order.

	return ok
}
