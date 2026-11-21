package mesh

import (
	"context"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/safe"
	"github.com/traefik/traefik/v3/pkg/tcp"
)

const (
	// ImportImported is a route that became a router of this instance.
	ImportImported = "imported"
	// ImportLocal is a route this instance has a router of its own for.
	ImportLocal = "local"
	// ImportSkipped is a route this instance cannot serve.
	ImportSkipped = "skipped"
)

// ImportedRoute is what this instance made of a route that other instances announce.
type ImportedRoute struct {
	Route

	// Peers are the instances that announce the route.
	Peers []string `json:"peers"`
	// Status is imported, local or skipped.
	Status string `json:"status"`
	// Router is the name of the router of this instance for the route, when it is imported.
	Router string `json:"router,omitempty"`
	// Reason tells why the route is not imported.
	Reason string `json:"reason,omitempty"`
}

// Provider gives the routing configuration the routers learned from the other instances of the mesh.
type Provider struct {
	mesh *Mesh
}

// Provider returns the provider of the routers learned from the other instances.
func (m *Mesh) Provider() *Provider {
	return &Provider{mesh: m}
}

// Init does nothing: the mesh is prepared with the cluster.
func (p *Provider) Init() error {
	return nil
}

// Provide records where the configurations are sent. The mesh sends one each time the routes of the other instances change.
func (p *Provider) Provide(configurationChan chan<- dynamic.Message, _ *safe.Pool) error {
	p.mesh.mu.Lock()
	p.mesh.configurationChan = configurationChan
	p.mesh.mu.Unlock()

	p.mesh.publish()

	return nil
}

// ThrottleDuration disables the throttling of the providers: the mesh gathers its changes itself,
// and a route of an instance that went away must leave at once.
func (p *Provider) ThrottleDuration() time.Duration {
	return 0
}

// publish asks for the routers of the other instances to be sent to the routing configuration.
func (m *Mesh) publish() {
	select {
	case m.publishCh <- struct{}{}:
	default:
	}
}

// publishLoop sends the routers of the other instances to the routing configuration when they change.
func (m *Mesh) publishLoop(ctx context.Context) {
	for {
		select {
		case <-m.publishCh:
		case <-ctx.Done():
			return
		}

		// The instances of a mesh often change together, e.g. when a link comes back with every route of an instance.
		select {
		case <-time.After(m.publishDelay):
		case <-ctx.Done():
			return
		}

		select {
		case <-m.publishCh:
		default:
		}

		m.mu.Lock()
		configurationChan := m.configurationChan
		m.mu.Unlock()

		if configurationChan == nil {
			continue
		}

		conf, report := m.buildConfiguration()

		m.mu.Lock()
		m.importReport = report
		m.mu.Unlock()

		select {
		case configurationChan <- dynamic.Message{ProviderName: ProviderName, Configuration: conf}:
		case <-ctx.Done():
			return
		}
	}
}

// routeGroup gathers the instances that announce the same route.
type routeGroup struct {
	route      Route
	signature  string
	candidates []forwardCandidate
}

func (g *routeGroup) peers() []string {
	peers := make([]string, 0, len(g.candidates))
	for _, candidate := range g.candidates {
		peers = append(peers, candidate.nodeID)
	}

	return peers
}

// buildConfiguration makes the routers of this instance for the routes of the other instances,
// and reports what became of each route.
func (m *Mesh) buildConfiguration() (*dynamic.Configuration, []ImportedRoute) {
	m.mu.Lock()
	local := m.localRouters

	// The routers this instance announces, by key: another instance has the same route when its announcement has the same signature.
	announced := map[string]Route{}
	for _, route := range m.exported {
		announced[routeKey(route.Kind, route.Name)] = route
	}

	groups := map[string]map[string]*routeGroup{}

	for nodeID, table := range m.imported {
		for _, route := range table.routes {
			key := routeKey(route.Kind, route.Name)
			signature := route.signature()

			if groups[key] == nil {
				groups[key] = map[string]*routeGroup{}
			}

			group := groups[key][signature]
			if group == nil {
				group = &routeGroup{route: route, signature: signature}
				groups[key][signature] = group
			}

			group.candidates = append(group.candidates, forwardCandidate{nodeID: nodeID, priority: route.MeshPriority, down: route.Down})
		}
	}
	m.mu.Unlock()

	tables := newForwardTables()

	// The maps are never nil: a configuration without any router still has to replace the previous one.
	conf := &dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Routers:  map[string]*dynamic.Router{},
			Services: map[string]*dynamic.Service{},
		},
		TCP: &dynamic.TCPConfiguration{
			Routers:  map[string]*dynamic.TCPRouter{},
			Services: map[string]*dynamic.TCPService{},
		},
		UDP: &dynamic.UDPConfiguration{},
	}

	var report []ImportedRoute

	for _, key := range slices.Sorted(maps.Keys(groups)) {
		// The announcement most instances agree on keeps the plain name. A router that is being changed on its instances,
		// one at a time, shows up as two routes for a while.
		variants := make([]*routeGroup, 0, len(groups[key]))
		for _, group := range groups[key] {
			slices.SortFunc(group.candidates, func(a, b forwardCandidate) int { return strings.Compare(a.nodeID, b.nodeID) })
			variants = append(variants, group)
		}

		slices.SortFunc(variants, func(a, b *routeGroup) int {
			if len(a.candidates) != len(b.candidates) {
				return len(b.candidates) - len(a.candidates)
			}

			return strings.Compare(a.signature, b.signature)
		})

		for i, group := range variants {
			imported := ImportedRoute{Route: group.route, Peers: group.peers()}

			name := routerName(group.route.Name)
			if i > 0 {
				name += "-" + group.signature

				m.warnOnce("variant/"+key+"/"+group.signature, func() {
					m.logger.Warn().Str("router", group.route.Name).Strs("peers", group.peers()).
						Msg("Instances of the mesh announce a router with the same name and another rule, entry points or TLS: it is served as a route of its own")
				})
			}

			switch _, isLocal := local[key]; {
			case isLocal:
				imported.Status = ImportLocal
				imported.Reason = "this instance has a router with the same name"

				// With the same rule, entry points and TLS, it is the same route: the priorities choose where a request goes.
				if own, ok := announced[key]; ok && group.route.Kind != KindUDP && own.signature() == group.signature {
					tables.sharedOf(own.Kind)[own.Name] = &forwardTarget{
						kind:          own.Kind,
						router:        own.Name,
						candidates:    group.candidates,
						local:         true,
						localPriority: own.MeshPriority,
					}

					imported.Reason = "this instance has the router too: the priorities choose where a request is served"
				}

			case group.route.Kind == KindUDP:
				imported.Status = ImportSkipped
				imported.Reason = "the mesh does not forward UDP routers yet"

			default:
				entryPoints := intersect(group.route.EntryPoints, m.entryPoints)

				if len(entryPoints) == 0 {
					imported.Status = ImportSkipped
					imported.Reason = "this instance has none of the entry points of the router"

					break
				}

				if group.route.Kind == KindTCP {
					router := &dynamic.TCPRouter{
						EntryPoints: entryPoints,
						Rule:        group.route.Rule,
						RuleSyntax:  group.route.RuleSyntax,
						Priority:    group.route.Priority,
						Service:     name,
					}

					if tls := group.route.TLS; tls != nil {
						router.TLS = &dynamic.RouterTCPTLSConfig{
							Passthrough:  tls.Passthrough,
							Options:      tls.Options,
							CertResolver: tls.CertResolver,
							Domains:      tls.Domains,
						}
					}

					conf.TCP.Routers[name] = router
					// The service has no configuration: the mesh builds its handler.
					conf.TCP.Services[name] = &dynamic.TCPService{}
				} else {
					router := &dynamic.Router{
						EntryPoints: entryPoints,
						Rule:        group.route.Rule,
						RuleSyntax:  group.route.RuleSyntax,
						Priority:    group.route.Priority,
						Service:     name,
					}

					if tls := group.route.TLS; tls != nil {
						router.TLS = &dynamic.RouterTLSConfig{Options: tls.Options, CertResolver: tls.CertResolver, Domains: tls.Domains}
					}

					conf.HTTP.Routers[name] = router
					// The service has no configuration: the mesh builds its handler.
					conf.HTTP.Services[name] = &dynamic.Service{}
				}

				tables.targetsOf(group.route.Kind)[name+"@"+ProviderName] = &forwardTarget{
					kind:       group.route.Kind,
					router:     group.route.Name,
					candidates: group.candidates,
				}

				imported.Status = ImportImported
				imported.Router = name + "@" + ProviderName
			}

			report = append(report, imported)
		}
	}

	// The tables change with the routers, and also when an instance joins the ones that own a router that is already there,
	// or when the servers of one of them go down: the requests read them each time.
	m.tables.Store(tables)

	return conf, report
}

// routerName makes the name of the router of this instance from the name of the router on the instance that owns it.
func routerName(name string) string {
	return strings.ReplaceAll(name, "@", "-")
}

func intersect(wanted, available []string) []string {
	var common []string

	for _, name := range wanted {
		if slices.Contains(available, name) {
			common = append(common, name)
		}
	}

	return common
}

// ServiceBuilder builds the handlers of the services of the routers learned from the other instances.
type ServiceBuilder struct {
	mesh *Mesh
}

// ServiceBuilder returns the builder of the services of the routers learned from the other instances.
func (m *Mesh) ServiceBuilder() *ServiceBuilder {
	return &ServiceBuilder{mesh: m}
}

// BuildHTTP returns the handler of a service of the mesh, and nothing for the other services.
func (b *ServiceBuilder) BuildHTTP(_ context.Context, serviceName string) (http.Handler, error) {
	if !strings.HasSuffix(serviceName, "@"+ProviderName) {
		return nil, nil
	}

	return &meshService{mesh: b.mesh, name: serviceName}, nil
}

// BuildTCP returns the handler of a TCP service of the mesh, and nothing for the other services.
func (m *Mesh) BuildTCP(_ context.Context, serviceName string) (tcp.Handler, error) {
	if !strings.HasSuffix(serviceName, "@"+ProviderName) {
		return nil, nil
	}

	return &meshTCPService{mesh: m, name: serviceName}, nil
}
