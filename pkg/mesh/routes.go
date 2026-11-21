package mesh

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/types"
)

const (
	// ProviderName is the name of the provider of the routers learned from the other instances.
	ProviderName = "mesh"
	// internalProviderName is the provider of the routers of the instance itself (dashboard, ping, ...), which stay local.
	internalProviderName = "internal"

	// KindHTTP is an HTTP router.
	KindHTTP = "http"
	// KindTCP is a TCP router.
	KindTCP = "tcp"
	// KindUDP is a UDP router.
	KindUDP = "udp"

	pathRoutes = "/v1/routes"

	// routesPollTimeout is how long an instance holds a request for its routes when they have not changed.
	routesPollTimeout = 30 * time.Second
	// publishDelay gathers the changes of several instances into one change of the routing configuration.
	publishDelay = 200 * time.Millisecond

	// maxRoutes bounds the table an instance takes from another one.
	maxRoutes = 20000
	// maxRoutesBody bounds what an instance reads of the table of another one.
	maxRoutesBody = 64 << 20

	// maxMeshPriority is the highest priority of a router in the mesh.
	maxMeshPriority = 1000
)

// Route is a router that an instance announces to the others: what they need to match the requests for it.
// The middlewares, the service and the servers of the router stay with the instance that owns it.
type Route struct {
	Kind string `json:"kind"`
	// Name is the name of the router on the instance that owns it, with its provider.
	Name        string    `json:"name"`
	Rule        string    `json:"rule,omitempty"`
	RuleSyntax  string    `json:"ruleSyntax,omitempty"`
	Priority    int       `json:"priority,omitempty"`
	EntryPoints []string  `json:"entryPoints,omitempty"`
	TLS         *RouteTLS `json:"tls,omitempty"`
	// MeshPriority chooses between the instances that have the router.
	MeshPriority int `json:"meshPriority,omitempty"`
	// Down is true when the service of the router has no server to answer.
	Down bool `json:"down,omitempty"`
}

// RouteTLS is the TLS configuration of an announced router.
type RouteTLS struct {
	Passthrough  bool           `json:"passthrough,omitempty"`
	Options      string         `json:"options,omitempty"`
	CertResolver string         `json:"certResolver,omitempty"`
	Domains      []types.Domain `json:"domains,omitempty"`
}

// signature identifies what makes two announcements the same route: the same requests reach it, in the same way.
// The priority in the mesh is not part of it: it is what tells the instances of one route apart.
func (r Route) signature() string {
	entryPoints := slices.Clone(r.EntryPoints)
	slices.Sort(entryPoints)

	data, err := json.Marshal(struct {
		Kind        string    `json:"kind"`
		Rule        string    `json:"rule"`
		RuleSyntax  string    `json:"ruleSyntax"`
		Priority    int       `json:"priority"`
		EntryPoints []string  `json:"entryPoints"`
		TLS         *RouteTLS `json:"tls"`
	}{r.Kind, r.Rule, r.RuleSyntax, r.Priority, entryPoints, r.TLS})
	if err != nil {
		return ""
	}

	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:6])
}

func routeKey(kind, name string) string {
	return kind + "/" + name
}

// routesAnswer is the table of the routers of an instance.
type routesAnswer struct {
	// Instance changes when the instance restarts, and its generations start again.
	Instance string `json:"instance"`
	// Generation changes with the table.
	Generation uint64  `json:"generation"`
	Routes     []Route `json:"routes"`
}

// importedTable is the table of another instance, as last received.
type importedTable struct {
	instance   string
	generation uint64
	routes     []Route
}

// Transform reads the routers of the local providers from the configurations on their way to the routing,
// and returns the configurations as they are.
// It sees the routers as the providers wrote them, before the middlewares of the entry points are added to them.
func (m *Mesh) Transform(_ context.Context, configurations dynamic.Configurations) dynamic.Configurations {
	routes, local := m.localRoutes(configurations)

	// The routers that are announced are the ones the other instances may send requests for.
	served := map[string]*dynamic.Router{}

	for providerName, conf := range configurations {
		if conf == nil || conf.HTTP == nil {
			continue
		}

		for name, router := range conf.HTTP.Routers {
			qualified := qualifiedName(name, providerName)

			if slices.ContainsFunc(routes, func(r Route) bool { return r.Kind == KindHTTP && r.Name == qualified }) {
				served[qualified] = router.DeepCopy()
			}
		}
	}

	m.mu.Lock()
	m.localBase = routes
	exportChanged := m.refreshExportedLocked()

	localChanged := !reflect.DeepEqual(local, m.localRouters)
	m.localRouters = local
	m.localHTTP = served
	m.mu.Unlock()

	// A router that appears, goes or changes locally changes what is made of the routes of the other instances.
	if localChanged || exportChanged {
		m.publish()
	}

	return configurations
}

// localRoutes returns the routers to announce, and the names of all the routers of the local providers.
func (m *Mesh) localRoutes(configurations dynamic.Configurations) ([]Route, map[string]struct{}) {
	routes := []Route{}
	local := map[string]struct{}{}

	for providerName, conf := range configurations {
		if providerName == ProviderName || providerName == internalProviderName || conf == nil {
			continue
		}

		if conf.HTTP != nil {
			for name, router := range conf.HTTP.Routers {
				// A router with a parent is reached through it, and one without a service only routes to its children:
				// neither can be served by its name alone.
				if router == nil || len(router.ParentRefs) > 0 || router.Service == "" {
					continue
				}

				route := Route{
					Kind:        KindHTTP,
					Name:        qualifiedName(name, providerName),
					Rule:        router.Rule,
					RuleSyntax:  router.RuleSyntax,
					Priority:    router.Priority,
					EntryPoints: m.entryPointsOf(router.EntryPoints, m.defaultEntryPoints),
				}

				if router.TLS != nil {
					route.TLS = &RouteTLS{Options: router.TLS.Options, CertResolver: router.TLS.CertResolver, Domains: router.TLS.Domains}
				}

				routes = m.addLocalRoute(routes, local, route, router.Mesh)
			}
		}

		if conf.TCP != nil {
			for name, router := range conf.TCP.Routers {
				if router == nil {
					continue
				}

				route := Route{
					Kind:        KindTCP,
					Name:        qualifiedName(name, providerName),
					Rule:        router.Rule,
					RuleSyntax:  router.RuleSyntax,
					Priority:    router.Priority,
					EntryPoints: m.entryPointsOf(router.EntryPoints, m.defaultEntryPoints),
				}

				if router.TLS != nil {
					route.TLS = &RouteTLS{
						Passthrough:  router.TLS.Passthrough,
						Options:      router.TLS.Options,
						CertResolver: router.TLS.CertResolver,
						Domains:      router.TLS.Domains,
					}
				}

				routes = m.addLocalRoute(routes, local, route, router.Mesh)
			}
		}

		if conf.UDP != nil {
			for name, router := range conf.UDP.Routers {
				if router == nil {
					continue
				}

				route := Route{
					Kind:        KindUDP,
					Name:        qualifiedName(name, providerName),
					EntryPoints: m.entryPointsOf(router.EntryPoints, m.udpEntryPoints),
				}

				routes = m.addLocalRoute(routes, local, route, router.Mesh)
			}
		}
	}

	slices.SortFunc(routes, func(a, b Route) int { return strings.Compare(routeKey(a.Kind, a.Name), routeKey(b.Kind, b.Name)) })

	return routes, local
}

func (m *Mesh) addLocalRoute(routes []Route, local map[string]struct{}, route Route, options *dynamic.RouterMesh) []Route {
	local[routeKey(route.Kind, route.Name)] = struct{}{}

	export := m.conf.Export == cluster.MeshExportAll

	if options != nil {
		if options.Export != nil {
			export = *options.Export
		}

		route.MeshPriority = options.Priority

		if route.MeshPriority < 0 || route.MeshPriority > maxMeshPriority {
			m.warnOnce("priority/"+route.Kind+"/"+route.Name, func() {
				m.logger.Warn().Str("router", route.Name).Int("priority", options.Priority).
					Msgf("The mesh priority of a router must be between 0 and %d: the router is announced with the default priority", maxMeshPriority)
			})

			route.MeshPriority = 0
		}
	}

	if !export {
		return routes
	}

	return append(routes, route)
}

// entryPointsOf returns the entry points of a router, which are the default ones when it names none.
func (m *Mesh) entryPointsOf(entryPoints, defaults []string) []string {
	if len(entryPoints) == 0 {
		entryPoints = defaults
	}

	sorted := slices.Clone(entryPoints)
	slices.Sort(sorted)

	return slices.Compact(sorted)
}

func qualifiedName(name, providerName string) string {
	if strings.Contains(name, "@") {
		return name
	}

	return name + "@" + providerName
}

// serveRoutes answers a request for the table of this instance.
// An instance that already has the current table gets the next one when it changes, or nothing after a while.
func (m *Mesh) serveRoutes(rw http.ResponseWriter, req *http.Request) {
	m.seen(peerFrom(req.Context()))

	after, _ := strconv.ParseUint(req.URL.Query().Get("after"), 10, 64)
	instance := req.URL.Query().Get("instance")

	m.mu.Lock()
	generation, changed := m.exportGeneration, m.exportChanged
	m.mu.Unlock()

	if instance == m.member.Instance() && after == generation {
		timer := time.NewTimer(m.routesPollTimeout)
		defer timer.Stop()

		select {
		case <-changed:
		case <-timer.C:
			rw.WriteHeader(http.StatusNoContent)
			return
		case <-req.Context().Done():
			return
		}
	}

	m.mu.Lock()
	answer := routesAnswer{Instance: m.member.Instance(), Generation: m.exportGeneration, Routes: m.exported}
	m.mu.Unlock()

	rw.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(rw).Encode(answer); err != nil {
		m.logger.Debug().Err(err).Msg("Unable to answer a request for the routes of this instance")
	}
}

// askRoutes asks the other end of a session for its table, unless it is still the one this instance has.
// It returns nil when the table has not changed.
func (m *Mesh) askRoutes(ctx context.Context, s *session, known *importedTable) (*importedTable, error) {
	ctx, cancel := context.WithTimeout(ctx, m.routesPollTimeout+m.linkTimeout)
	defer cancel()

	query := url.Values{}
	if known != nil {
		query.Set("instance", known.instance)
		query.Set("after", strconv.FormatUint(known.generation, 10))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+controlHost+pathRoutes+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil, nil
	case http.StatusOK:
	default:
		return nil, fmt.Errorf("unexpected answer to a request for the routes of an instance: %s", resp.Status)
	}

	var answer routesAnswer
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxRoutesBody)).Decode(&answer); err != nil {
		return nil, fmt.Errorf("reading the routes of an instance: %w", err)
	}

	routes := make([]Route, 0, min(len(answer.Routes), maxRoutes))

	for _, route := range answer.Routes[:min(len(answer.Routes), maxRoutes)] {
		switch route.Kind {
		case KindHTTP, KindTCP, KindUDP:
		default:
			// A kind of a later version.
			continue
		}

		if route.Name == "" {
			continue
		}

		routes = append(routes, route)
	}

	return &importedTable{instance: answer.Instance, generation: answer.Generation, routes: routes}, nil
}

// setImported records the table of an instance, or forgets it with a nil table, when its link is lost.
func (m *Mesh) setImported(nodeID string, table *importedTable) {
	m.mu.Lock()
	if table == nil {
		_, known := m.imported[nodeID]
		delete(m.imported, nodeID)
		m.mu.Unlock()

		if known {
			m.publish()
		}

		return
	}

	m.imported[nodeID] = table
	m.mu.Unlock()

	m.publish()
}

func (m *Mesh) importedFrom(nodeID string) *importedTable {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.imported[nodeID]
}

// warnOnce runs a warning the first time it is asked for under a key.
func (m *Mesh) warnOnce(key string, warn func()) {
	m.mu.Lock()
	_, warned := m.warned[key]
	m.warned[key] = struct{}{}
	m.mu.Unlock()

	if !warned {
		warn()
	}
}
