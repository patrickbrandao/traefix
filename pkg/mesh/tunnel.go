package mesh

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

// Stream is a stream of bytes through a link, opened with a CONNECT request: a TCP connection travels in one.
type Stream interface {
	io.ReadWriteCloser

	// CloseWrite tells the other end that nothing more is written, and keeps reading.
	// The accepting end of a link over TCP cannot do it: its stream ends when it is closed.
	CloseWrite() error

	// Done is closed once nothing more can be written to the stream: the other end gave it up, the link was lost,
	// or this end closed its writing side.
	Done() <-chan struct{}
}

// deadliner is a stream whose pending reads and writes can be bounded, as the ones of a connection.
// The accepting end of a link has it, over both transports.
type deadliner interface {
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

// streamCanceled is the HTTP/3 code of a stream given up by its user.
const streamCanceled = quic.StreamErrorCode(http3.ErrCodeRequestCanceled)

func newTunnelRequest(ctx context.Context, header http.Header, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodConnect, "https://"+controlHost, body)
	if err != nil {
		return nil, err
	}

	maps.Copy(req.Header, header)

	return req, nil
}

// OpenStream opens a stream through the link. The header tells the other end what the stream is for.
// The stream ends with the context.
func (s *session) OpenStream(ctx context.Context, header http.Header) (Stream, error) {
	stream, _, err := s.openTunnel(ctx, header)

	return stream, err
}

// OpenFlow opens a flow of packets through the link. The flow ends with the context.
func (s *session) OpenFlow(ctx context.Context, header http.Header) (*Flow, error) {
	stream, datagrams, err := s.openTunnel(ctx, header)
	if err != nil {
		return nil, err
	}

	return newFlow(ctx, stream, datagrams), nil
}

func (s *session) openTunnel(ctx context.Context, header http.Header) (Stream, datagrammer, error) {
	if s.h3 == nil {
		stream, err := s.openTunnelTCP(ctx, header)
		return stream, nil, err
	}

	req, err := newTunnelRequest(ctx, header, nil)
	if err != nil {
		return nil, nil, err
	}

	str, err := s.h3.OpenRequestStream(ctx)
	if err != nil {
		return nil, nil, err
	}

	stream := &quicClientStream{str: str}

	if err := str.SendRequestHeader(req); err != nil {
		_ = stream.Close()
		return nil, nil, err
	}

	resp, err := str.ReadResponse()
	if err != nil {
		_ = stream.Close()
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		_ = stream.Close()
		return nil, nil, fmt.Errorf("the other instance refused the stream: %s", resp.Status)
	}

	// The context only bounds the opening of an HTTP/3 stream: the stream is made to end with it.
	stream.stop = context.AfterFunc(ctx, func() { _ = stream.Close() })

	return stream, str, nil
}

func (s *session) openTunnelTCP(ctx context.Context, header http.Header) (Stream, error) {
	reader, writer := io.Pipe()

	req, err := newTunnelRequest(ctx, header, reader)
	if err != nil {
		return nil, err
	}

	resp, err := s.h2.RoundTrip(req)
	if err != nil {
		_ = writer.Close()
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		_ = writer.Close()
		_ = resp.Body.Close()

		return nil, fmt.Errorf("the other instance refused the stream: %s", resp.Status)
	}

	stream := &tcpClientStream{body: resp.Body, writer: writer, done: make(chan struct{})}

	// The HTTP/2 client does not look at the context while it waits for the next bytes to send:
	// the stream is made to end with it.
	stream.stop = context.AfterFunc(ctx, func() { _ = stream.Close() })

	return stream, nil
}

// acceptStream answers the CONNECT request of a stream. The handler must not return before it is done with the stream.
func acceptStream(rw http.ResponseWriter, req *http.Request) (Stream, error) {
	stream, _, err := acceptTunnel(rw, req)

	return stream, err
}

// acceptFlow answers the CONNECT request of a flow of packets. The handler must not return before it is done with the flow.
func acceptFlow(rw http.ResponseWriter, req *http.Request) (*Flow, error) {
	stream, datagrams, err := acceptTunnel(rw, req)
	if err != nil {
		return nil, err
	}

	return newFlow(req.Context(), stream, datagrams), nil
}

func acceptTunnel(rw http.ResponseWriter, req *http.Request) (Stream, datagrammer, error) {
	if req.Method != http.MethodConnect {
		return nil, nil, errors.New("a stream of the mesh needs a CONNECT request")
	}

	rw.WriteHeader(http.StatusOK)

	if streamer, ok := rw.(http3.HTTPStreamer); ok {
		str, err := streamer.HTTPStream()
		if err != nil {
			return nil, nil, err
		}

		return &quicServerStream{str: str}, str, nil
	}

	controller := http.NewResponseController(rw)

	if err := controller.Flush(); err != nil {
		return nil, nil, err
	}

	return &tcpServerStream{body: req.Body, rw: rw, controller: controller, done: req.Context().Done()}, nil, nil
}

type quicClientStream struct {
	str  *http3.RequestStream
	stop func() bool
}

func (s *quicClientStream) Read(p []byte) (int, error)  { return s.str.Read(p) }
func (s *quicClientStream) Write(p []byte) (int, error) { return s.str.Write(p) }
func (s *quicClientStream) CloseWrite() error           { return s.str.Close() }
func (s *quicClientStream) Done() <-chan struct{}       { return s.str.Context().Done() }

func (s *quicClientStream) Close() error {
	if s.stop != nil {
		s.stop()
	}

	s.str.CancelRead(streamCanceled)

	return s.str.Close()
}

type quicServerStream struct {
	str *http3.Stream
}

func (s *quicServerStream) Read(p []byte) (int, error)  { return s.str.Read(p) }
func (s *quicServerStream) Write(p []byte) (int, error) { return s.str.Write(p) }
func (s *quicServerStream) CloseWrite() error           { return s.str.Close() }
func (s *quicServerStream) Done() <-chan struct{}       { return s.str.Context().Done() }

func (s *quicServerStream) SetReadDeadline(t time.Time) error  { return s.str.SetReadDeadline(t) }
func (s *quicServerStream) SetWriteDeadline(t time.Time) error { return s.str.SetWriteDeadline(t) }

func (s *quicServerStream) Close() error {
	s.str.CancelRead(streamCanceled)

	return s.str.Close()
}

type tcpClientStream struct {
	body   io.ReadCloser
	writer *io.PipeWriter
	stop   func() bool

	// The other end cannot stop sending and keep receiving: the stream is over when nothing more comes from it.
	done chan struct{}
	once sync.Once
}

func (s *tcpClientStream) Read(p []byte) (int, error) {
	n, err := s.body.Read(p)
	if err != nil {
		s.once.Do(func() { close(s.done) })
	}

	return n, err
}

func (s *tcpClientStream) Write(p []byte) (int, error) { return s.writer.Write(p) }
func (s *tcpClientStream) CloseWrite() error           { return s.writer.Close() }
func (s *tcpClientStream) Done() <-chan struct{}       { return s.done }

func (s *tcpClientStream) Close() error {
	if s.stop != nil {
		s.stop()
	}

	s.once.Do(func() { close(s.done) })

	_ = s.writer.Close()

	return s.body.Close()
}

type tcpServerStream struct {
	body       io.ReadCloser
	rw         http.ResponseWriter
	controller *http.ResponseController
	done       <-chan struct{}
}

func (s *tcpServerStream) Read(p []byte) (int, error) { return s.body.Read(p) }

func (s *tcpServerStream) Write(p []byte) (int, error) {
	n, err := s.rw.Write(p)
	if err != nil {
		return n, err
	}

	// Nothing reaches the other end before a flush.
	return n, s.controller.Flush()
}

// CloseWrite cannot end the response and keep the request open: the stream is closed when its handler returns.
func (s *tcpServerStream) CloseWrite() error { return nil }

func (s *tcpServerStream) Close() error { return s.body.Close() }

func (s *tcpServerStream) Done() <-chan struct{} { return s.done }

func (s *tcpServerStream) SetReadDeadline(t time.Time) error { return s.controller.SetReadDeadline(t) }

func (s *tcpServerStream) SetWriteDeadline(t time.Time) error {
	return s.controller.SetWriteDeadline(t)
}

// datagrammer sends packets without retransmission nor order, as UDP does: the QUIC datagrams of a stream.
type datagrammer interface {
	SendDatagram(p []byte) error
	ReceiveDatagram(ctx context.Context) ([]byte, error)
}

const (
	// maxPacketSize is the largest UDP payload.
	maxPacketSize = 65535
	// flowBacklog is how many received packets a flow keeps for its reader. Like a socket buffer, it drops the rest.
	flowBacklog = 256
)

// Flow carries packets through a link: the packets of a UDP session travel in one.
// Over QUIC, a packet goes in a datagram when it fits, without retransmission nor order.
// A packet that does not fit, and every packet over TCP, goes in the stream of the flow, with its length first.
type Flow struct {
	stream    Stream
	datagrams datagrammer

	ctx    context.Context
	cancel context.CancelCauseFunc

	writeMu  sync.Mutex
	incoming chan []byte

	sentDatagrams     atomic.Uint64
	sentInStream      atomic.Uint64
	receivedDatagrams atomic.Uint64
	receivedInStream  atomic.Uint64
}

// flowCounts are the packets a flow carried, by path and direction.
type flowCounts struct {
	sentDatagrams, sentInStream, receivedDatagrams, receivedInStream uint64
}

func newFlow(ctx context.Context, stream Stream, datagrams datagrammer) *Flow {
	ctx, cancel := context.WithCancelCause(ctx)

	f := &Flow{stream: stream, datagrams: datagrams, ctx: ctx, cancel: cancel, incoming: make(chan []byte, flowBacklog)}

	go f.readStream()

	if datagrams != nil {
		go f.readDatagrams()
	}

	// The readers are blocked in the stream: closing it is what stops them.
	context.AfterFunc(ctx, func() { _ = stream.Close() })

	return f
}

// Send sends a packet.
func (f *Flow) Send(p []byte) error {
	if len(p) > maxPacketSize {
		return fmt.Errorf("packet of %d bytes is larger than a UDP packet", len(p))
	}

	if f.datagrams != nil {
		err := f.datagrams.SendDatagram(p)
		if err == nil {
			f.sentDatagrams.Add(1)
			return nil
		}

		if _, tooLarge := errors.AsType[*quic.DatagramTooLargeError](err); !tooLarge {
			return err
		}
	}

	frame := make([]byte, 2+len(p))
	binary.BigEndian.PutUint16(frame, uint16(len(p))) //nolint:gosec // The length is checked above.
	copy(frame[2:], p)

	f.writeMu.Lock()
	defer f.writeMu.Unlock()

	if _, err := f.stream.Write(frame); err != nil {
		return err
	}

	f.sentInStream.Add(1)

	return nil
}

// Receive returns the next packet.
func (f *Flow) Receive(ctx context.Context) ([]byte, error) {
	select {
	case p := <-f.incoming:
		return p, nil
	case <-f.ctx.Done():
		return nil, context.Cause(f.ctx)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close ends the flow.
func (f *Flow) Close() error {
	f.cancel(io.EOF)

	return nil
}

func (f *Flow) counts() flowCounts {
	return flowCounts{
		sentDatagrams:     f.sentDatagrams.Load(),
		sentInStream:      f.sentInStream.Load(),
		receivedDatagrams: f.receivedDatagrams.Load(),
		receivedInStream:  f.receivedInStream.Load(),
	}
}

func (f *Flow) readStream() {
	header := make([]byte, 2)

	for {
		if _, err := io.ReadFull(f.stream, header); err != nil {
			f.cancel(err)
			return
		}

		p := make([]byte, binary.BigEndian.Uint16(header))

		if _, err := io.ReadFull(f.stream, p); err != nil {
			f.cancel(err)
			return
		}

		f.receivedInStream.Add(1)

		// The stream is reliable: its packets wait for the reader instead of being dropped.
		select {
		case f.incoming <- p:
		case <-f.ctx.Done():
			return
		}
	}
}

func (f *Flow) readDatagrams() {
	for {
		p, err := f.datagrams.ReceiveDatagram(f.ctx)
		if err != nil {
			f.cancel(err)
			return
		}

		f.receivedDatagrams.Add(1)

		select {
		case f.incoming <- p:
		default:
		}
	}
}
