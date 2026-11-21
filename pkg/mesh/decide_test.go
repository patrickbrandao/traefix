package mesh

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/runtime"
)

// places names the places of a plan: "local" for the instance itself.
func places(choices []choice) []string {
	var names []string

	for _, c := range choices {
		if c.local {
			names = append(names, "local")
			continue
		}

		names = append(names, c.nodeID)
	}

	return names
}

func TestMesh_plan(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	second := newTestNode(t, "srv2", testToken, nil, first.address)
	third := newTestNode(t, "srv3", testToken, nil, first.address)

	// The instance of the tests, linked to the two others.
	first.waitLinked("srv2", "srv3")
	second.waitLinked("srv1", "srv3")
	third.waitLinked("srv1", "srv2")

	m := first.mesh

	// The order of the client among the two other instances, which the hash decides.
	client := "198.51.100.7"
	byClient := []string{"srv2", "srv3"}
	if rendezvous(client, "srv3") > rendezvous(client, "srv2") {
		byClient = []string{"srv3", "srv2"}
	}

	testCases := []struct {
		desc     string
		target   forwardTarget
		expected []string
	}{
		{
			desc:     "default priorities: this instance, the other one only if this one cannot",
			target:   forwardTarget{local: true, candidates: []forwardCandidate{{nodeID: "srv2"}}},
			expected: []string{"local", "srv2"},
		},
		{
			desc:     "no local router: the other instances share the requests",
			target:   forwardTarget{candidates: []forwardCandidate{{nodeID: "srv2"}, {nodeID: "srv3"}}},
			expected: byClient,
		},
		{
			desc:     "another instance has a priority: it gets the requests",
			target:   forwardTarget{local: true, candidates: []forwardCandidate{{nodeID: "srv2", priority: 10}}},
			expected: []string{"srv2", "local"},
		},
		{
			desc:     "this instance has the highest priority",
			target:   forwardTarget{local: true, localPriority: 10, candidates: []forwardCandidate{{nodeID: "srv2", priority: 5}, {nodeID: "srv3"}}},
			expected: []string{"local", "srv2", "srv3"},
		},
		{
			desc:     "a lower priority than another instance",
			target:   forwardTarget{local: true, localPriority: 5, candidates: []forwardCandidate{{nodeID: "srv2", priority: 10}}},
			expected: []string{"srv2", "local"},
		},
		{
			desc:     "an instance says its servers are down",
			target:   forwardTarget{local: true, candidates: []forwardCandidate{{nodeID: "srv2", priority: 10, down: true}, {nodeID: "srv3", priority: 5}}},
			expected: []string{"srv3", "local"},
		},
		{
			desc:     "an instance this one has no link to",
			target:   forwardTarget{candidates: []forwardCandidate{{nodeID: "srv9", priority: 10}, {nodeID: "srv2"}}},
			expected: []string{"srv2"},
		},
		{
			desc:     "nobody can take the request",
			target:   forwardTarget{candidates: []forwardCandidate{{nodeID: "srv9"}, {nodeID: "srv2", down: true}}},
			expected: nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			test.target.router = "app@docker"

			assert.Equal(t, test.expected, places(m.plan(&test.target, client)))
		})
	}

	t.Run("a tie on a priority: the client decides, this instance among the others", func(t *testing.T) {
		target := &forwardTarget{
			router: "app@docker", local: true, localPriority: 10,
			candidates: []forwardCandidate{{nodeID: "srv2", priority: 10}, {nodeID: "srv3"}},
		}

		counts := map[string]int{}

		for i := range 200 {
			plan := places(m.plan(target, fmt.Sprintf("198.51.100.%d", i)))

			require.Len(t, plan, 3)
			assert.Equal(t, "srv3", plan[2])
			counts[plan[0]]++

			// Asked again, the same client gets the same place.
			assert.Equal(t, plan, places(m.plan(target, fmt.Sprintf("198.51.100.%d", i))))
		}

		assert.Greater(t, counts["local"], 60)
		assert.Greater(t, counts["srv2"], 60)
	})

	t.Run("the local router is down: this instance comes last", func(t *testing.T) {
		down := map[string]struct{}{routeKey(KindHTTP, "app@docker"): {}}
		m.localDown.Store(&down)
		t.Cleanup(func() { m.localDown.Store(&map[string]struct{}{}) })

		target := &forwardTarget{kind: KindHTTP, router: "app@docker", local: true, localPriority: 10, candidates: []forwardCandidate{{nodeID: "srv2"}}}
		assert.Equal(t, []string{"srv2", "local"}, places(m.plan(target, client)))

		// Alone, it still answers, as it would without the mesh.
		alone := &forwardTarget{kind: KindHTTP, router: "app@docker", local: true}
		assert.Equal(t, []string{"local"}, places(m.plan(alone, client)))
	})

	t.Run("an instance in quarantine", func(t *testing.T) {
		target := &forwardTarget{router: "quarantined@docker", local: true, candidates: []forwardCandidate{{nodeID: "srv2", priority: 10}}}

		for range quarantineAfter - 1 {
			m.failed(target.key(), "srv2")
		}

		assert.Equal(t, []string{"srv2", "local"}, places(m.plan(target, client)))

		m.failed(target.key(), "srv2")
		assert.Equal(t, []string{"local"}, places(m.plan(target, client)))

		// The other routes of the instance are not concerned.
		other := &forwardTarget{router: "other@docker", candidates: []forwardCandidate{{nodeID: "srv2"}}}
		assert.Equal(t, []string{"srv2"}, places(m.plan(other, client)))

		first.advance(quarantineFirst + time.Second)
		assert.Equal(t, []string{"srv2", "local"}, places(m.plan(target, client)))

		// It fails again right away: it stays away twice as long.
		for range quarantineAfter {
			m.failed(target.key(), "srv2")
		}

		first.advance(quarantineFirst + time.Second)
		assert.Equal(t, []string{"local"}, places(m.plan(target, client)))

		first.advance(quarantineFirst)
		assert.Equal(t, []string{"srv2", "local"}, places(m.plan(target, client)))

		// A request that gets through clears the count.
		m.failed(target.key(), "srv2")
		m.succeeded(target.key(), "srv2")

		for range quarantineAfter - 1 {
			m.failed(target.key(), "srv2")
		}

		assert.Equal(t, []string{"srv2", "local"}, places(m.plan(target, client)))
	})
}

func TestMesh_plan_roundRobin(t *testing.T) {
	roundRobin := func(conf *cluster.Configuration) { conf.Mesh.Balance = cluster.MeshBalanceRoundRobin }

	first := newTestNode(t, "srv1", testToken, roundRobin)
	second := newTestNode(t, "srv2", testToken, roundRobin, first.address)

	first.waitLinked("srv2")
	second.waitLinked("srv1")

	tied := &forwardTarget{router: "app@docker", local: true, localPriority: 10, candidates: []forwardCandidate{{nodeID: "srv2", priority: 10}}}

	counts := map[string]int{}
	for range 100 {
		counts[places(first.mesh.plan(tied, "198.51.100.7"))[0]]++
	}

	assert.Equal(t, map[string]int{"local": 50, "srv2": 50}, counts)

	// With the default priority, this instance takes its own requests, round robin or not.
	byDefault := &forwardTarget{router: "app@docker", local: true, candidates: []forwardCandidate{{nodeID: "srv2"}}}

	for range 10 {
		assert.Equal(t, []string{"local", "srv2"}, places(first.mesh.plan(byDefault, "198.51.100.7")))
	}
}

// decided returns the handler of the router of the tests on an instance that has it:
// the choice of the mesh in front of the local handler.
func (n *testNode) decided(local http.Handler) http.Handler {
	return n.mesh.Decide("app@docker", local)
}

func TestMesh_sharedRoute(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	first.owns(named("srv1"), 0)

	second := newTestNode(t, "srv2", testToken, nil, first.address)
	second.owns(named("srv2"), 0)

	first.waitLinked("srv2")
	second.waitLinked("srv1")

	fromFirst, fromSecond := first.decided(named("srv1")), second.decided(named("srv2"))

	require.Eventually(t, func() bool { return first.candidates() == 1 && second.candidates() == 1 }, 10*time.Second, 10*time.Millisecond)

	// Each instance has the router and no priority is set: each one serves what it receives.
	assert.Equal(t, "srv1", ask(t, fromFirst, "198.51.100.7:1000"))
	assert.Equal(t, "srv2", ask(t, fromSecond, "198.51.100.7:1000"))

	// The first instance gets a priority: the second one sends its requests there.
	first.owns(named("srv1"), 10)

	require.Eventually(t, func() bool { return ask(t, fromSecond, "198.51.100.7:1000") == "srv1" }, 10*time.Second, 10*time.Millisecond)
	assert.Equal(t, "srv1", ask(t, fromFirst, "198.51.100.7:1000"))

	// A request that came through the mesh is served where it arrives: it never goes back.
	assert.Equal(t, uint64(0), first.mesh.turn.Load())

	// The first instance goes away: the second one serves its requests itself again.
	first.stop()

	require.Eventually(t, func() bool { return ask(t, fromSecond, "198.51.100.7:1000") == "srv2" }, 10*time.Second, 10*time.Millisecond)
}

func TestMesh_sharedRoute_anotherRule(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	first.owns(named("srv1"), 10)

	second := newTestNode(t, "srv2", testToken, nil, first.address)
	second.mesh.SwitchHandlers(Handlers{HTTP: map[string]http.Handler{"app@docker": named("srv2")}}, nil)
	second.mesh.Transform(t.Context(), dynamic.Configurations{
		"docker": {HTTP: &dynamic.HTTPConfiguration{Routers: map[string]*dynamic.Router{
			"app": {Rule: "Host(`other.example.com`)", EntryPoints: []string{"web"}, Service: "app"},
		}}},
	})

	second.waitLinked("srv1")

	require.Eventually(t, func() bool {
		table := second.mesh.importedFrom("srv1")
		return table != nil && len(table.routes) == 1
	}, 10*time.Second, 10*time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	// The same name with another rule is another route: the priority of the other instance does not take the requests.
	assert.Equal(t, 0, second.candidates())
	assert.Equal(t, "srv2", ask(t, second.decided(named("srv2")), "198.51.100.7:1000"))
}

func TestMesh_retry(t *testing.T) {
	var served atomic.Int32

	primary := newTestNode(t, "srv1", testToken, nil)
	primary.owns(named("srv1"), 10)

	backup := newTestNode(t, "srv2", testToken, nil, primary.address)
	backup.owns(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		served.Add(1)
		_, _ = rw.Write([]byte("srv2"))
	}), 0)

	ingress := newTestNode(t, "srv3", testToken, nil, primary.address)
	ingress.waitLinked("srv1", "srv2")

	handler := ingress.entry()

	require.Eventually(t, func() bool { return ingress.candidates() == 2 }, 10*time.Second, 10*time.Millisecond)
	require.Equal(t, "srv1", ask(t, handler, "198.51.100.7:1000"))

	// The preferred instance lost the router, and the others have not heard of it yet.
	primary.mesh.routing.Store(&routing{})

	// It says so without doing anything with the request: the next instance serves it.
	assert.Equal(t, "srv2", ask(t, handler, "198.51.100.7:1000"))

	// A request with a body is not sent a second time: what was read of it is gone.
	req := httptest.NewRequest(http.MethodPost, "http://app.example.com/", strings.NewReader("payload"))
	req.RemoteAddr = "198.51.100.7:1000"

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Contains(t, recorder.Header().Get("Proxy-Status"), "traefix-mesh")

	// After a few failures the instance is left alone for the route: the requests go straight to the other one.
	require.Equal(t, "srv2", ask(t, handler, "198.51.100.7:1000"))
	require.True(t, ingress.mesh.quarantined(routeKey(KindHTTP, "app@docker"), "srv1"))

	before := served.Load()

	req = httptest.NewRequest(http.MethodPost, "http://app.example.com/", strings.NewReader("payload"))
	req.RemoteAddr = "198.51.100.7:1000"

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, before+1, served.Load())
}

func Test_serviceUp(t *testing.T) {
	loadBalancer := func(servers int, statuses ...string) *runtime.ServiceInfo {
		service := &runtime.ServiceInfo{Service: &dynamic.Service{LoadBalancer: &dynamic.ServersLoadBalancer{}}}

		for i := range servers {
			service.LoadBalancer.Servers = append(service.LoadBalancer.Servers, dynamic.Server{URL: fmt.Sprintf("http://10.0.0.%d", i)})
		}

		for i, status := range statuses {
			service.UpdateServerStatus(fmt.Sprintf("http://10.0.0.%d", i), status)
		}

		return service
	}

	conf := &runtime.Configuration{Services: map[string]*runtime.ServiceInfo{
		"up@docker":       loadBalancer(2, runtime.StatusUp, runtime.StatusDown),
		"down@docker":     loadBalancer(2, runtime.StatusDown, runtime.StatusDown),
		"empty@docker":    loadBalancer(0),
		"unknown@docker":  loadBalancer(2),
		"weighted@docker": {Service: &dynamic.Service{Weighted: &dynamic.WeightedRoundRobin{}}},
	}}

	assert.True(t, serviceUp(conf, "up@docker"))
	assert.False(t, serviceUp(conf, "down@docker"))
	assert.False(t, serviceUp(conf, "empty@docker"))
	assert.True(t, serviceUp(conf, "unknown@docker"))
	assert.True(t, serviceUp(conf, "weighted@docker"))
	assert.False(t, serviceUp(conf, "missing@docker"))
	assert.True(t, serviceUp(conf, "api@internal"))
	assert.True(t, serviceUp(nil, "anything@docker"))
}

func TestMesh_health(t *testing.T) {
	owner := newTestNode(t, "srv1", testToken, nil)
	owner.owns(named("srv1"), 0)

	ingress := newTestNode(t, "srv2", testToken, nil, owner.address)
	handler := ingress.entry()

	require.Equal(t, "srv1", ask(t, handler, "198.51.100.7:1000"))

	service := &runtime.ServiceInfo{Service: &dynamic.Service{LoadBalancer: &dynamic.ServersLoadBalancer{Servers: []dynamic.Server{{URL: "http://10.0.0.1"}}}}}
	service.UpdateServerStatus("http://10.0.0.1", runtime.StatusUp)

	conf := &runtime.Configuration{Services: map[string]*runtime.ServiceInfo{"app@docker": service}}

	owner.mesh.SwitchHandlers(Handlers{HTTP: map[string]http.Handler{"app@docker": named("srv1")}}, conf)

	require.Equal(t, "srv1", ask(t, handler, "198.51.100.7:1000"))

	answer := func() int {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://app.example.com/", nil))

		return recorder.Code
	}

	// The health check of the owner takes its only server down: it says so, and the requests stop coming.
	service.UpdateServerStatus("http://10.0.0.1", runtime.StatusDown)
	owner.mesh.checkHealth()

	assert.True(t, owner.mesh.isLocalDown(routeKey(KindHTTP, "app@docker")))

	require.Eventually(t, func() bool { return answer() == http.StatusServiceUnavailable }, 10*time.Second, 10*time.Millisecond)

	// The server is back.
	service.UpdateServerStatus("http://10.0.0.1", runtime.StatusUp)
	owner.mesh.checkHealth()

	require.Eventually(t, func() bool { return answer() == http.StatusOK }, 10*time.Second, 10*time.Millisecond)
}

func TestMesh_LocalRoute(t *testing.T) {
	node := newTestNode(t, "srv1", testToken, nil)

	node.mesh.Transform(t.Context(), dynamic.Configurations{
		"docker": {HTTP: &dynamic.HTTPConfiguration{Routers: map[string]*dynamic.Router{
			"app":     {Rule: "Host(`app.example.com`)", Service: "app", Middlewares: []string{"auth", "compress@file"}},
			"plain":   {Rule: "Host(`plain.example.com`)", Service: "plain"},
			"private": {Rule: "Host(`private.example.com`)", Service: "private", Mesh: &dynamic.RouterMesh{Export: new(false)}},
		}}},
	})

	testCases := []struct {
		desc         string
		router       string
		entryPoint   string
		middlewares  []string
		expectedName string
		expectedOwn  int
		expectedOK   bool
	}{
		{
			desc:   "router as its provider wrote it",
			router: "app@docker", entryPoint: "web", middlewares: []string{"auth@docker", "compress@file"},
			expectedName: "app@docker", expectedOwn: 2, expectedOK: true,
		},
		{
			desc:   "with the middlewares of its entry point first",
			router: "app@docker", entryPoint: "web", middlewares: []string{"waf@file", "auth@docker", "compress@file"},
			expectedName: "app@docker", expectedOwn: 2, expectedOK: true,
		},
		{
			desc:   "copy of a router of several entry points",
			router: "websecure-app@docker", entryPoint: "websecure", middlewares: []string{"waf@file", "auth@docker", "compress@file"},
			expectedName: "app@docker", expectedOwn: 2, expectedOK: true,
		},
		{
			desc:   "router without middlewares",
			router: "plain@docker", entryPoint: "web", middlewares: []string{"waf@file"},
			expectedName: "plain@docker", expectedOwn: 0, expectedOK: true,
		},
		{desc: "router that is not announced", router: "private@docker", entryPoint: "web"},
		{desc: "router of another provider", router: "app@file", entryPoint: "web", middlewares: []string{"auth@docker", "compress@file"}},
		{desc: "middlewares of a configuration that changed since", router: "app@docker", entryPoint: "web", middlewares: []string{"auth@docker"}},
		{desc: "prefix that is not the entry point", router: "web-app@docker", entryPoint: "websecure", middlewares: []string{"auth@docker", "compress@file"}},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			name, own, ok := node.mesh.LocalRoute(test.router, test.entryPoint, test.middlewares)

			assert.Equal(t, test.expectedOK, ok)
			assert.Equal(t, test.expectedName, name)
			assert.Equal(t, test.expectedOwn, own)
		})
	}
}
