package acme

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
)

// watchStore makes the resolver follow the certificates that other instances write to its store,
// and take part in their election.
func (p *Provider) watchStore() {
	if watcher, ok := p.Store.(storeWatcher); ok {
		watcher.Watch(p.ResolverName, p.reloadCertificates)
	}

	if coord := p.coordinator(); coord != nil {
		coord.start(p, p.pool)
	}
}

// reloadCertificates replaces the certificates of the resolver with the ones of its store, and serves them.
func (p *Provider) reloadCertificates() {
	p.certificatesMu.Lock()
	defer p.certificatesMu.Unlock()

	// The store is read with the lock held: a certificate this resolver is saving is either in the store already,
	// or added to the list after this reload.
	certificates, err := p.Store.GetCertificates(p.ResolverName)
	if err != nil {
		log.Error().Err(err).Str(logs.ProviderName, p.ResolverName+resolverSuffix).Msg("Unable to reload the ACME certificates")
		return
	}

	p.certificates = certificates

	p.configurationChan <- p.buildMessage()
}

// checkStore tells whether the resolver may use the ACME server, and takes the account another instance registered.
// When no instance registered the account, this one does so under a lock, which the returned function frees.
// The client mutex must be held.
func (p *Provider) checkStore(ctx context.Context) (func(), error) {
	noop := func() {}

	gate, ok := p.Store.(storeGate)
	if !ok {
		return noop, nil
	}

	if err := gate.Ready(p.ResolverName); err != nil {
		return noop, err
	}

	if p.account != nil {
		return noop, nil
	}

	usable := func(account *Account) bool {
		return account != nil && account.Registration != nil && isAccountMatchingCaServer(ctx, account.Registration.URI, p.CAServer)
	}

	coord := p.coordinator()
	if coord == nil {
		account, err := p.Store.GetAccount(p.ResolverName)
		if err != nil {
			return noop, err
		}

		if usable(account) {
			p.account = account
		}

		return noop, nil
	}

	account, lock, err := coord.lockAccount(ctx, usable)
	if err != nil {
		return noop, err
	}

	if lock == nil {
		p.account = account
		return noop, nil
	}

	return func() { _ = lock.Release(ctx) }, nil
}
