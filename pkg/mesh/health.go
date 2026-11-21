package mesh

import (
	"context"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/traefik/traefik/v3/pkg/config/runtime"
	"github.com/traefik/traefik/v3/pkg/tcp"
	"github.com/traefik/traefik/v3/pkg/udp"
)

// healthInterval is how often an instance looks at the servers of the routers it announces.
const defaultHealthInterval = time.Second

// routing is what serves, by name, the routers this instance announces: the handlers of one routing configuration,
// and the configuration itself, which tells the state of their servers.
type routing struct {
	http map[string]http.Handler
	tcp  map[string]tcp.Handler
	udp  map[string]udp.Handler
	conf *runtime.Configuration
}

// Handlers are the handlers that serve the routers of this instance by name, for what the other instances forward.
type Handlers struct {
	HTTP map[string]http.Handler
	TCP  map[string]tcp.Handler
	UDP  map[string]udp.Handler
}

// SwitchHandlers replaces the handlers that serve the routers of this instance by name,
// with the routing configuration they were built from.
func (m *Mesh) SwitchHandlers(handlers Handlers, conf *runtime.Configuration) {
	m.routing.Store(&routing{http: handlers.HTTP, tcp: handlers.TCP, udp: handlers.UDP, conf: conf})

	m.checkHealth()
}

// healthLoop follows the state of the servers of the routers this instance announces, until the context is done.
func (m *Mesh) healthLoop(ctx context.Context) {
	ticker := time.NewTicker(m.healthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkHealth()
		case <-ctx.Done():
			return
		}
	}
}

// checkHealth finds the announced routers that have no server to answer, and tells the other instances when that changes.
// A router is down when its service is a load balancer whose servers are all down or gone,
// or when it could not be built with the routing configuration in use.
func (m *Mesh) checkHealth() {
	current := m.routing.Load()
	if current == nil {
		// The routing is not built yet.
		return
	}

	m.mu.Lock()
	httpRouters := maps.Clone(m.localHTTP)
	tcpRouters := maps.Clone(m.localTCP)
	udpRouters := maps.Clone(m.localUDP)
	m.mu.Unlock()

	down := map[string]struct{}{}

	for name, router := range httpRouters {
		if current.http[name] == nil || !serviceUp(current.conf, qualify(router.Service, name)) {
			down[routeKey(KindHTTP, name)] = struct{}{}
		}
	}

	for name, router := range tcpRouters {
		if current.tcp[name] == nil || !tcpServiceUp(current.conf, qualify(router.Service, name)) {
			down[routeKey(KindTCP, name)] = struct{}{}
		}
	}

	for name, router := range udpRouters {
		if current.udp[name] == nil || !udpServiceUp(current.conf, qualify(router.Service, name)) {
			down[routeKey(KindUDP, name)] = struct{}{}
		}
	}

	if previous := m.localDown.Load(); previous != nil && reflect.DeepEqual(*previous, down) {
		return
	}

	m.localDown.Store(&down)

	m.mu.Lock()
	changed := m.refreshExportedLocked()
	m.mu.Unlock()

	if changed {
		// What this instance does with its own requests for these routers changes too.
		m.publish()
	}
}

// isLocalDown tells whether the router of this instance, named by the key of its route, has no server to answer.
func (m *Mesh) isLocalDown(key string) bool {
	down := m.localDown.Load()
	if down == nil {
		return false
	}

	_, isDown := (*down)[key]

	return isDown
}

// refreshExportedLocked rebuilds the table this instance announces from its routers and their state,
// and reports whether the table changed.
func (m *Mesh) refreshExportedLocked() bool {
	down := m.localDown.Load()

	routes := make([]Route, 0, len(m.localBase))
	for _, route := range m.localBase {
		if down != nil {
			_, route.Down = (*down)[routeKey(route.Kind, route.Name)]
		}

		routes = append(routes, route)
	}

	if reflect.DeepEqual(routes, m.exported) {
		return false
	}

	m.exported = routes
	m.exportGeneration++
	m.setExportedRoutes(routes)

	close(m.exportChanged)
	m.exportChanged = make(chan struct{})

	return true
}

// serviceUp tells whether a service has a server to answer.
func serviceUp(conf *runtime.Configuration, serviceName string) bool {
	if conf == nil {
		return true
	}

	service, ok := conf.Services[serviceName]
	if !ok || service == nil {
		// The services without a configuration of their own, like the ones of the instance itself, are always there.
		return strings.HasSuffix(serviceName, "@"+internalProviderName)
	}

	if service.Service == nil || service.LoadBalancer == nil {
		// A service made of other services answers for itself.
		return true
	}

	if len(service.LoadBalancer.Servers) == 0 {
		return false
	}

	statuses := service.GetAllStatus()
	if len(statuses) == 0 {
		// Nothing was said of the servers yet.
		return true
	}

	return slices.Contains(slices.Collect(maps.Values(statuses)), runtime.StatusUp)
}

// tcpServiceUp tells whether a TCP service has a server to answer.
func tcpServiceUp(conf *runtime.Configuration, serviceName string) bool {
	if conf == nil {
		return true
	}

	service, ok := conf.TCPServices[serviceName]
	if !ok || service == nil {
		return false
	}

	if service.TCPService == nil || service.LoadBalancer == nil {
		// A service made of other services answers for itself.
		return true
	}

	if len(service.LoadBalancer.Servers) == 0 {
		return false
	}

	statuses := service.GetAllStatus()
	if len(statuses) == 0 {
		// Nothing was said of the servers yet.
		return true
	}

	return slices.Contains(slices.Collect(maps.Values(statuses)), runtime.StatusUp)
}

// udpServiceUp tells whether a UDP service has a server to answer. UDP services have no health check:
// a load balancer is up while it has servers.
func udpServiceUp(conf *runtime.Configuration, serviceName string) bool {
	if conf == nil {
		return true
	}

	service, ok := conf.UDPServices[serviceName]
	if !ok || service == nil {
		return false
	}

	if service.UDPService == nil || service.LoadBalancer == nil {
		// A service made of other services answers for itself.
		return true
	}

	return len(service.LoadBalancer.Servers) > 0
}

// qualify returns the name of an element as the router named by qualified name refers to it: in its provider by default.
func qualify(element, routerName string) string {
	if strings.Contains(element, "@") {
		return element
	}

	if _, providerName, ok := strings.Cut(routerName, "@"); ok {
		return element + "@" + providerName
	}

	return element
}

// LocalRoute tells whether a router of the routing configuration is one this instance announces to the mesh.
// It returns the name it is announced under, and how many of its last middlewares are its own: the ones before them
// were added by its entry point.
func (m *Mesh) LocalRoute(routerName, entryPointName string, middlewares []string) (string, int, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := routerName

	router, ok := m.localHTTP[name]
	if !ok {
		// The routing gives a router of several entry points one copy for each, named after the entry point.
		name, ok = strings.CutPrefix(routerName, entryPointName+"-")
		if !ok {
			return "", 0, false
		}

		if router, ok = m.localHTTP[name]; !ok {
			return "", 0, false
		}
	}

	own := make([]string, 0, len(router.Middlewares))
	for _, middleware := range router.Middlewares {
		own = append(own, qualify(middleware, name))
	}

	// The configuration being built may not be the one the routers were last read from.
	if len(own) > len(middlewares) || !slices.Equal(middlewares[len(middlewares)-len(own):], own) {
		return "", 0, false
	}

	return name, len(own), true
}

// Decide returns the handler that serves a request for a router this instance announces:
// on this instance, with the given handler, or on another instance that has the router, as the priorities say.
func (m *Mesh) Decide(name string, local http.Handler) http.Handler {
	return &sharedRoute{mesh: m, name: name, local: local}
}
