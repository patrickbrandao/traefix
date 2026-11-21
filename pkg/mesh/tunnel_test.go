package mesh

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
)

const testLinkTimeout = 5 * time.Second

// freeAddress returns a loopback address whose port is free in TCP and in UDP.
func freeAddress(t *testing.T) string {
	t.Helper()

	for range 20 {
		tcp, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)

		address := tcp.Addr().String()
		_ = tcp.Close()

		udp, err := net.ListenPacket("udp", address)
		if err != nil {
			continue
		}
		_ = udp.Close()

		return address
	}

	require.Fail(t, "no port free in TCP and in UDP")

	return ""
}

// serveTest runs a listener of the mesh with a handler, and returns its address.
func serveTest(t *testing.T, identity *Identity, transport string, handler http.Handler) string {
	t.Helper()

	address := freeAddress(t)

	l, err := newListener(address, transport, identity, testLinkTimeout)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})

	go func() {
		defer close(done)
		l.serve(ctx, handler)
	}()

	t.Cleanup(func() {
		cancel()
		<-done
	})

	return address
}

func dialTest(t *testing.T, identity *Identity, transport, address string) *session {
	t.Helper()

	dial := dialQUIC
	if transport == TransportTCP {
		dial = dialTCP
	}

	ctx, cancel := context.WithTimeout(t.Context(), testLinkTimeout)
	defer cancel()

	s, err := dial(ctx, identity, address, testLinkTimeout)
	require.NoError(t, err)
	t.Cleanup(s.Close)

	return s
}

// echoHandler answers what the tests of the tunnels need: a request, a stream and a flow that send back what they get.
func echoHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.Header.Get("Test-Kind") {
		case "stream":
			stream, err := acceptStream(rw, req)
			if err != nil {
				return
			}
			defer func() { _ = stream.Close() }()

			_, _ = io.Copy(stream, stream)

		case "flow":
			flow, err := acceptFlow(rw, req)
			if err != nil {
				return
			}
			defer func() { _ = flow.Close() }()

			for {
				p, err := flow.Receive(req.Context())
				if err != nil {
					return
				}

				if err := flow.Send(p); err != nil {
					return
				}
			}

		case "refuse":
			http.Error(rw, "no", http.StatusServiceUnavailable)

		default:
			body, _ := io.ReadAll(req.Body)
			rw.Header().Set("Test-Peer", peerFrom(req.Context()))
			_, _ = rw.Write(body)
		}
	})
}

func transports() []string {
	return []string{TransportQUIC, TransportTCP}
}

func TestSession_RoundTrip(t *testing.T) {
	for _, transport := range transports() {
		t.Run(transport, func(t *testing.T) {
			server := newTestIdentity(t, "srv1", testToken)
			client := newTestIdentity(t, "srv2", testToken)

			address := serveTest(t, server, cluster.MeshTransportAuto, echoHandler(t))
			s := dialTest(t, client, transport, address)

			assert.Equal(t, transport, s.transport)
			assert.Equal(t, "srv1", s.peer)

			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://"+controlHost+"/v1/test", bytes.NewReader([]byte("hello")))
			require.NoError(t, err)

			resp, err := s.RoundTrip(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "hello", string(body))
			// The listener names the instance that sent the request from its certificate.
			assert.Equal(t, "srv2", resp.Header.Get("Test-Peer"))
		})
	}
}

func TestSession_OpenStream(t *testing.T) {
	for _, transport := range transports() {
		t.Run(transport, func(t *testing.T) {
			server := newTestIdentity(t, "srv1", testToken)
			client := newTestIdentity(t, "srv2", testToken)

			address := serveTest(t, server, cluster.MeshTransportAuto, echoHandler(t))
			s := dialTest(t, client, transport, address)

			stream, err := s.OpenStream(t.Context(), http.Header{"Test-Kind": {"stream"}})
			require.NoError(t, err)

			// More than the flow control windows, in both directions at once.
			sent := make([]byte, 4<<20)
			_, err = rand.Read(sent)
			require.NoError(t, err)

			writeErr := make(chan error, 1)
			go func() {
				_, err := stream.Write(sent)
				if err == nil {
					err = stream.CloseWrite()
				}
				writeErr <- err
			}()

			received, err := io.ReadAll(stream)
			require.NoError(t, err)
			require.NoError(t, <-writeErr)

			assert.True(t, bytes.Equal(sent, received))
			require.NoError(t, stream.Close())

			// Several streams live side by side on one link.
			first, err := s.OpenStream(t.Context(), http.Header{"Test-Kind": {"stream"}})
			require.NoError(t, err)
			second, err := s.OpenStream(t.Context(), http.Header{"Test-Kind": {"stream"}})
			require.NoError(t, err)

			_, err = second.Write([]byte("second"))
			require.NoError(t, err)
			_, err = first.Write([]byte("first"))
			require.NoError(t, err)

			buf := make([]byte, 6)
			_, err = io.ReadFull(second, buf)
			require.NoError(t, err)
			assert.Equal(t, "second", string(buf))

			_, err = io.ReadFull(first, buf[:5])
			require.NoError(t, err)
			assert.Equal(t, "first", string(buf[:5]))

			require.NoError(t, first.Close())
			require.NoError(t, second.Close())
		})
	}
}

func TestSession_OpenStream_refused(t *testing.T) {
	for _, transport := range transports() {
		t.Run(transport, func(t *testing.T) {
			server := newTestIdentity(t, "srv1", testToken)
			client := newTestIdentity(t, "srv2", testToken)

			address := serveTest(t, server, cluster.MeshTransportAuto, echoHandler(t))
			s := dialTest(t, client, transport, address)

			_, err := s.OpenStream(t.Context(), http.Header{"Test-Kind": {"refuse"}})
			require.ErrorContains(t, err, "503")
		})
	}
}

func TestSession_OpenStream_endsWithItsContext(t *testing.T) {
	for _, transport := range transports() {
		t.Run(transport, func(t *testing.T) {
			server := newTestIdentity(t, "srv1", testToken)
			client := newTestIdentity(t, "srv2", testToken)

			address := serveTest(t, server, cluster.MeshTransportAuto, echoHandler(t))
			s := dialTest(t, client, transport, address)

			ctx, cancel := context.WithCancel(t.Context())

			stream, err := s.OpenStream(ctx, http.Header{"Test-Kind": {"stream"}})
			require.NoError(t, err)

			cancel()

			_, err = io.ReadAll(stream)
			require.Error(t, err)
		})
	}
}

func TestSession_OpenFlow(t *testing.T) {
	for _, transport := range transports() {
		t.Run(transport, func(t *testing.T) {
			server := newTestIdentity(t, "srv1", testToken)
			client := newTestIdentity(t, "srv2", testToken)

			address := serveTest(t, server, cluster.MeshTransportAuto, echoHandler(t))
			s := dialTest(t, client, transport, address)

			flow, err := s.OpenFlow(t.Context(), http.Header{"Test-Kind": {"flow"}})
			require.NoError(t, err)

			// From a small packet to the largest UDP payload: each one comes back whole.
			for _, size := range []int{1, 100, 1100, 1400, 9000, maxPacketSize} {
				sent := make([]byte, size)
				_, err = rand.Read(sent)
				require.NoError(t, err)

				require.NoError(t, flow.Send(sent))

				ctx, cancel := context.WithTimeout(t.Context(), testLinkTimeout)
				received, err := flow.Receive(ctx)
				cancel()

				require.NoError(t, err, "packet of %d bytes", size)
				assert.True(t, bytes.Equal(sent, received), "packet of %d bytes", size)
			}

			if transport == TransportTCP {
				assert.Zero(t, flow.sentDatagrams.Load())
				assert.Equal(t, uint64(6), flow.sentInStream.Load())
			} else {
				// The small packets travel as datagrams, the ones that do not fit in the stream.
				assert.Positive(t, flow.sentDatagrams.Load())
				assert.Positive(t, flow.sentInStream.Load())
				assert.Equal(t, uint64(6), flow.sentDatagrams.Load()+flow.sentInStream.Load())
			}

			require.Error(t, flow.Send(make([]byte, maxPacketSize+1)))

			require.NoError(t, flow.Close())

			_, err = flow.Receive(t.Context())
			require.Error(t, err)
		})
	}
}

// Test_datagramSize records the largest packet that travels as a QUIC datagram on the loopback interface,
// which the documentation of the UDP forwarding quotes.
func Test_datagramSize(t *testing.T) {
	server := newTestIdentity(t, "srv1", testToken)
	client := newTestIdentity(t, "srv2", testToken)

	address := serveTest(t, server, cluster.MeshTransportQUIC, echoHandler(t))
	s := dialTest(t, client, TransportQUIC, address)

	flow, err := s.OpenFlow(t.Context(), http.Header{"Test-Kind": {"flow"}})
	require.NoError(t, err)
	defer func() { _ = flow.Close() }()

	err = flow.datagrams.SendDatagram(make([]byte, maxPacketSize))

	var tooLarge *quic.DatagramTooLargeError
	require.ErrorAs(t, err, &tooLarge)

	t.Logf("largest packet in a QUIC datagram right after the handshake: %d bytes", tooLarge.MaxDatagramPayloadSize)

	// A packet of the smallest size every IPv6 path carries, minus the QUIC and UDP headers, must fit.
	assert.GreaterOrEqual(t, tooLarge.MaxDatagramPayloadSize, int64(1100))
}
