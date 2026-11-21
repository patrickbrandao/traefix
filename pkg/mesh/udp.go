package mesh

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/udp"
)

const (
	// forwardUDP is a session of a UDP router: its packets travel in a flow of the link.
	forwardUDP = "udp"

	// defaultUDPTimeout is the idle timeout of the UDP entry points of Traefik, for a session that tells none.
	defaultUDPTimeout = 3 * time.Second
)

// meshUDPService forwards the sessions of a UDP router learned from the mesh to an instance that owns it.
type meshUDPService struct {
	mesh *Mesh
	// name is the name of the service, which is the one of the router of this instance.
	name string
}

func (s *meshUDPService) ServeUDP(conn *udp.Conn) {
	target := s.mesh.tables.Load().udpTargets[s.name]
	if target == nil {
		s.mesh.refuseUDP(conn, "no instance of the mesh announces the router")
		return
	}

	s.mesh.dispatchUDP(conn, target, nil)
}

// sharedUDPRoute decides, for a UDP router that this instance and other instances have, where a session is served.
type sharedUDPRoute struct {
	mesh *Mesh
	name string
	// local is the service of the router on this instance.
	local udp.Handler
}

func (s *sharedUDPRoute) ServeUDP(conn *udp.Conn) {
	// Most routers live on one instance: nothing is decided for them.
	target := s.mesh.tables.Load().udpShared[s.name]
	if target == nil {
		s.local.ServeUDP(conn)
		return
	}

	s.mesh.dispatchUDP(conn, target, s.local)
}

// LocalUDPRouters returns the UDP routers the other instances may send sessions for, as their providers wrote them.
func (m *Mesh) LocalUDPRouters() map[string]*dynamic.UDPRouter {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.localUDP
}

// LocalUDPRoute tells whether a UDP router of the routing configuration is one this instance announces to the mesh.
func (m *Mesh) LocalUDPRoute(routerName string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.localUDP[routerName]

	return ok
}

// DecideUDP returns the handler that serves a session for a UDP router this instance announces:
// on this instance, with the given handler, or on another instance that has the router, as the priorities say.
func (m *Mesh) DecideUDP(name string, local udp.Handler) udp.Handler {
	return &sharedUDPRoute{mesh: m, name: name, local: local}
}

// dispatchUDP serves a session where the priorities say, and falls back to the next place when one cannot take it.
// The packets that come while a flow opens wait in the session: none is lost to a place that refused it.
func (m *Mesh) dispatchUDP(conn *udp.Conn, target *forwardTarget, local udp.Handler) {
	remote := conn.RemoteAddr().String()

	client := remote
	if host, _, err := net.SplitHostPort(remote); err == nil {
		client = host
	}

	info := forwardInfo{Version: 1, Kind: forwardUDP, Router: target.router, Remote: remote, IdleTimeout: conn.Timeout().Milliseconds()}

	for _, next := range m.plan(target, client) {
		if next.local {
			local.ServeUDP(conn)
			return
		}

		session := m.sessionTo(next.nodeID)
		if session == nil {
			// The link went down since the places were chosen.
			continue
		}

		flow, release, err := openTunnel(m, session.OpenFlow, http.Header{headerForward: {info.encode()}})
		if err != nil {
			m.failed(target.key(), next.nodeID)
			m.countRequest(next.nodeID, forwardUDP, resultError)
			m.logger.Debug().Err(err).Str("peer", next.nodeID).Str("router", target.router).
				Msg("Unable to open a flow for a UDP session through the mesh")

			continue
		}

		m.succeeded(target.key(), next.nodeID)
		m.countRequest(next.nodeID, forwardUDP, resultForwarded)

		carryPackets(conn, flow)
		release()

		return
	}

	m.refuseUDP(conn, "no instance that owns the router can be reached")
}

// refuseUDP ends a session that could not be forwarded: its packets are dropped, as UDP does.
func (m *Mesh) refuseUDP(conn *udp.Conn, cause string) {
	m.countRequest("", forwardUDP, resultUnavailable)

	m.logger.Debug().Str("client", conn.RemoteAddr().String()).Str("cause", cause).Msg("Unable to forward a UDP session through the mesh")

	_ = conn.Close()
}

// carryPackets copies the packets between the session of a client and a flow to the instance that serves it,
// until either ends, and ends both.
func carryPackets(conn *udp.Conn, flow *Flow) {
	sent := make(chan struct{})

	go func() {
		defer close(sent)

		// The largest UDP payload: a read gets one packet, and drops what does not fit.
		buf := make([]byte, maxPacketSize)

		for {
			n, err := conn.Read(buf)
			if err != nil {
				break
			}

			// The flow copies the packet before it returns. It only fails once it ended.
			if err := flow.Send(buf[:n]); err != nil {
				break
			}
		}

		// The session ended without a packet for its timeout, or the flow did.
		_ = flow.Close()
	}()

	for {
		p, err := flow.Receive(context.Background())
		if err != nil {
			break
		}

		if _, err := conn.Write(p); err != nil {
			break
		}
	}

	_ = conn.Close()

	<-sent
}

// serveUDP serves a session that another instance received and forwarded: the packets of the flow are the ones
// of the client, for the service of the router, which sends them to its server from a socket of its own.
func (m *Mesh) serveUDP(rw http.ResponseWriter, req *http.Request, info forwardInfo, handler udp.Handler) {
	flow, err := acceptFlow(rw, req)
	if err != nil {
		return
	}

	timeout := defaultUDPTimeout
	if info.IdleTimeout > 0 {
		timeout = time.Duration(info.IdleTimeout) * time.Millisecond
	}

	// The instance that received the session sees the client, and ends the session when it is idle: a packet it gets
	// right then starts another session there. Ending it here too, at the same time, would drop that packet into a flow
	// that is closing. This instance only ends a session that the other one did not end, e.g. when the link is stuck.
	timeout += m.linkTimeout

	conn := udp.NewFlowConn(udpAddr(info.Remote), timeout, flow.Send, func() { _ = flow.Close() })

	received := make(chan struct{})

	go func() {
		defer close(received)

		for {
			p, err := flow.Receive(req.Context())
			if err != nil {
				// The other instance ended the session, or the link was lost.
				_ = conn.Close()
				return
			}

			if !conn.Deliver(p) {
				return
			}
		}
	}()

	// The handler returns once the session is over, and nothing may use the flow afterwards.
	handler.ServeUDP(conn)

	_ = conn.Close()

	<-received
}

// udpAddr reads an address as the one of a UDP client, which is what the handlers expect.
func udpAddr(address string) net.Addr {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return meshAddr(address)
	}

	ip := net.ParseIP(host)
	number, err := strconv.Atoi(port)

	if ip == nil || err != nil {
		return meshAddr(address)
	}

	return &net.UDPAddr{IP: ip, Port: number}
}
