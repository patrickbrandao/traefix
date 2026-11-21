package tcp

import (
	"context"

	"github.com/traefik/traefik/v3/pkg/tcp"
)

// TraefixBuilder builds the handlers of the services that have no configuration of their own,
// like the services of the routers learned from the mesh (Traefix).
// It returns nothing for a service that is not one of its own.
type TraefixBuilder interface {
	BuildTCP(ctx context.Context, serviceName string) (tcp.Handler, error)
}

// SetTraefixBuilder makes the manager ask the builder for a service before it looks for its configuration.
func (m *Manager) SetTraefixBuilder(builder TraefixBuilder) {
	m.traefixBuilder = builder
}

func (m *Manager) buildTraefix(ctx context.Context, serviceName string) (tcp.Handler, error) {
	if m.traefixBuilder == nil {
		return nil, nil
	}

	return m.traefixBuilder.BuildTCP(ctx, serviceName)
}
