package fingerprint

import (
	"context"
	"crypto/tls"
	"maps"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/quic-go/quic-go"
	h3qlog "github.com/quic-go/quic-go/http3/qlog"
	"github.com/quic-go/quic-go/qlogwriter"
	"golang.org/x/net/ipv4"
)

const (
	// maxSettingsWait bounds the wait of the first HTTP/3 request for the SETTINGS of the client.
	maxSettingsWait = 3 * time.Millisecond

	// quicPendingTTL bounds the life of the ClientHellos of the QUIC handshakes that never become a connection.
	quicPendingTTL = 30 * time.Second

	// maxQUICPending bounds the QUIC handshakes followed at once, e.g. under a flood of Initial packets.
	maxQUICPending = 4096
)

// quicHello is what the QUIC handshake tells of a client, until its connection is identified.
type quicHello struct {
	at        time.Time
	fromInfo  *ClientHello
	raw       []byte
	fromRaw   *ClientHello
	params    []uint64
	assembler *QUICClientHelloAssembler
	truncated bool

	initial quicInitialInfo
}

// quicInitialInfo is what the first Initial datagrams of a client tell, for the transport header.
type quicInitialInfo struct {
	version          uint32
	size             int
	dcidLen, scidLen int
	packets          int
	// firstAt is the arrival of the first Initial datagram, read by the Traefix.
	firstAt time.Time
}

type quicPending struct {
	mu    sync.Mutex
	byKey map[string]*quicHello
}

func (p *quicPending) get(key string, create bool) *quicHello {
	if p.byKey == nil {
		p.byKey = make(map[string]*quicHello)
	}

	h, ok := p.byKey[key]
	if ok || !create {
		return h
	}

	now := time.Now()
	if len(p.byKey) >= maxQUICPending {
		for k, v := range p.byKey {
			if now.Sub(v.at) > quicPendingTTL {
				delete(p.byKey, k)
			}
		}
		if len(p.byKey) >= maxQUICPending {
			return nil
		}
	}

	h = &quicHello{at: now}
	p.byKey[key] = h
	return h
}

func (p *quicPending) take(key string) *quicHello {
	p.mu.Lock()
	defer p.mu.Unlock()

	h := p.byKey[key]
	delete(p.byKey, key)
	return h
}

// QUICGetConfigForClient records the ClientHello of the QUIC handshakes, which the QUIC library only exposes
// through the TLS callback, before calling next.
func (e *EntryPoint) QUICGetConfigForClient(next func(*tls.ClientHelloInfo) (*tls.Config, error)) func(*tls.ClientHelloInfo) (*tls.Config, error) {
	return func(info *tls.ClientHelloInfo) (*tls.Config, error) {
		if info.Conn != nil {
			hello := clientHelloFromInfo(info)
			e.quic.mu.Lock()
			if h := e.quic.get(info.Conn.RemoteAddr().String(), true); h != nil {
				h.fromInfo = hello
			}
			e.quic.mu.Unlock()
		}
		return next(info)
	}
}

func clientHelloFromInfo(info *tls.ClientHelloInfo) *ClientHello {
	h := &ClientHello{
		// QUIC requires TLS 1.3, whose ClientHello carries the TLS 1.2 legacy version.
		LegacyVersion:     tls.VersionTLS12,
		CipherSuites:      slices.Clone(info.CipherSuites),
		Extensions:        slices.Clone(info.Extensions),
		ServerName:        info.ServerName,
		ALPN:              slices.Clone(info.SupportedProtos),
		PointFormats:      slices.Clone(info.SupportedPoints),
		SupportedVersions: slices.Clone(info.SupportedVersions),
	}
	for _, c := range info.SupportedCurves {
		h.SupportedGroups = append(h.SupportedGroups, uint16(c))
	}
	for _, s := range info.SignatureSchemes {
		h.SignatureAlgorithms = append(h.SignatureAlgorithms, uint16(s))
	}
	return h
}

// WrapPacketConn returns a packet connection recording the ClientHello bytes of the Initial packets, when the
// export carries the raw bytes.
func (e *EntryPoint) WrapPacketConn(c net.PacketConn) net.PacketConn {
	if !e.rawBytes {
		return c
	}
	if udp, ok := c.(*net.UDPConn); ok {
		return &initialUDPConn{UDPConn: udp, batch: ipv4.NewPacketConn(udp), ep: e}
	}
	return &initialConn{PacketConn: c, ep: e}
}

// initialUDPConn keeps the methods of the UDP socket, and reads in batches like the QUIC library does: the library
// then keeps its batch reads, its offloads and the control messages that give the TTL and the ECN of the packets,
// which it only uses when the connection it gets has these methods.
type initialUDPConn struct {
	*net.UDPConn

	batch *ipv4.PacketConn
	ep    *EntryPoint
}

// ReadBatch is the read the QUIC library uses on Linux, macOS and FreeBSD.
func (c *initialUDPConn) ReadBatch(ms []ipv4.Message, flags int) (int, error) {
	n, err := c.batch.ReadBatch(ms, flags)
	for i := range n {
		if m := &ms[i]; m.N > 0 && m.Addr != nil && len(m.Buffers) > 0 {
			c.observe(m.Addr, m.Buffers[0][:m.N])
		}
	}
	return n, err
}

// ReadFrom is the read the QUIC library uses on the other systems.
func (c *initialUDPConn) ReadFrom(p []byte) (int, net.Addr, error) {
	n, addr, err := c.UDPConn.ReadFrom(p)
	if n > 0 && addr != nil {
		c.observe(addr, p[:n])
	}
	return n, addr, err
}

func (c *initialUDPConn) observe(addr net.Addr, datagram []byte) {
	// Only the long header packets carry an Initial packet.
	if datagram[0]&0x80 != 0 {
		c.ep.observeInitial(addr.String(), datagram)
	}
}

type initialConn struct {
	net.PacketConn

	ep *EntryPoint
}

func (c *initialConn) ReadFrom(p []byte) (int, net.Addr, error) {
	n, addr, err := c.PacketConn.ReadFrom(p)
	// Only the long header packets carry an Initial packet.
	if n > 0 && addr != nil && p[0]&0x80 != 0 {
		c.ep.observeInitial(addr.String(), p[:n])
	}
	return n, addr, err
}

func (e *EntryPoint) observeInitial(key string, datagram []byte) {
	e.quic.mu.Lock()
	defer e.quic.mu.Unlock()

	h := e.quic.get(key, true)
	if h == nil || h.raw != nil || h.truncated {
		return
	}

	h.initial.packets++
	if h.initial.size == 0 {
		if pkt, _, err := parseQUICLongHeader(datagram); err == nil && pkt.initial {
			h.initial.version = pkt.version
			h.initial.size = len(datagram)
			h.initial.firstAt = time.Now()
			h.initial.dcidLen, h.initial.scidLen = len(pkt.dcid), pkt.scidLen
		}
	}

	_, frames, err := DecryptQUICInitials(datagram)
	if err != nil || len(frames) == 0 {
		return
	}

	if h.assembler == nil {
		h.assembler = &QUICClientHelloAssembler{}
	}
	for _, f := range frames {
		msg, done, err := h.assembler.Add(f)
		if err != nil {
			h.truncated = true
			h.assembler = nil
			e.truncated("client_hello")
			return
		}
		if done {
			h.raw = msg
			h.assembler = nil
			if parsed, err := ParseClientHelloMessage(msg); err == nil {
				h.fromRaw = parsed
			}
			h.params, _ = QUICTransportParameterIDs(msg)
			return
		}
	}
}

// quicTrace follows the HTTP/3 frames the client sends, from the qlog events of the QUIC library,
// for the header order of the requests and the SETTINGS of the connection.
type quicTrace struct {
	fields *fieldMatcher

	mu       sync.Mutex
	pending  []*Request
	streams  map[quic.StreamID]struct{}
	settings string
	// settingsSeen is closed with the first SETTINGS frame of the client.
	settingsSeen chan struct{}
	// waited makes only the first request wait, for a client that never sends SETTINGS.
	waited atomic.Bool
}

// QUICTracer is the qlog tracer of the connections, for the http3HeaderOrder option.
func (e *EntryPoint) QUICTracer(_ context.Context, isClient bool, _ quic.ConnectionID) qlogwriter.Trace {
	if isClient {
		return nil
	}
	return &quicTrace{fields: e.captureCfg.fields, streams: make(map[quic.StreamID]struct{}), settingsSeen: make(chan struct{})}
}

func (t *quicTrace) AddProducer() qlogwriter.Recorder {
	return quicRecorder{t}
}

type quicRecorder struct {
	t *quicTrace
}

func (r quicRecorder) Close() error { return nil }

func (r quicRecorder) RecordEvent(ev qlogwriter.Event) {
	parsed, ok := ev.(h3qlog.FrameParsed)
	if !ok {
		return
	}

	switch f := parsed.Frame.Frame.(type) {
	case h3qlog.HeadersFrame:
		r.t.addHeaders(parsed.StreamID, f)
	case h3qlog.SettingsFrame:
		r.t.setSettings(f)
	}
}

func (t *quicTrace) Settings() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.settings
}

func (t *quicTrace) addHeaders(id quic.StreamID, f h3qlog.HeadersFrame) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// A second header section on a stream is its trailers.
	if _, seen := t.streams[id]; seen {
		return
	}
	if len(t.streams) >= maxPendingRequests {
		clear(t.streams)
	}
	t.streams[id] = struct{}{}

	r := &Request{ProtoMajor: 3, streamID: uint32(id)}
	for _, hf := range f.HeaderFields {
		if strings.HasPrefix(hf.Name, ":") {
			r.PseudoNames = append(r.PseudoNames, hf.Name)
			switch hf.Name {
			case ":method":
				r.Method = hf.Value
			case ":path":
				r.Target = hf.Value
			case ":authority":
				r.Authority = hf.Value
			}
			continue
		}
		if len(r.Names) >= maxH2Fields {
			r.Truncated = true
			continue
		}
		r.Names = append(r.Names, hf.Name)
		if t.fields.match(hf.Name) {
			r.Fields = append(r.Fields, Field{Name: hf.Name, Value: truncateValue(hf.Value)})
		}
		switch hf.Name {
		case "cookie":
			r.Cookies = append(r.Cookies, hf.Value)
		case "accept-language":
			if !r.HasAcceptLanguage {
				r.AcceptLanguage = hf.Value
				r.HasAcceptLanguage = true
			}
		}
	}

	if r.Method == "" {
		return
	}
	if len(t.pending) >= maxPendingRequests {
		t.pending[0] = nil
		t.pending = t.pending[1:]
	}
	t.pending = append(t.pending, r)
}

// setSettings records the SETTINGS of the client, in the order of the wire, which the Traefix fork of the QUIC library
// gives with the qlog event (traefix-quic-go). A frame without that order is sorted by identifier.
func (t *quicTrace) setSettings(f h3qlog.SettingsFrame) {
	settings := maps.Clone(f.Other)
	if settings == nil {
		settings = map[uint64]uint64{}
	}
	if f.MaxFieldSectionSize >= 0 {
		settings[0x6] = uint64(f.MaxFieldSectionSize)
	}
	if f.ExtendedConnect >= 0 {
		settings[0x8] = uint64(f.ExtendedConnect)
	}
	if f.Datagram >= 0 {
		settings[0x33] = uint64(f.Datagram)
	}

	ids := make([]uint64, 0, len(settings))
	for _, id := range f.Order {
		if _, ok := settings[id]; ok {
			ids = append(ids, id)
		}
	}
	if len(ids) != len(settings) {
		ids = ids[:0]
		for id := range settings {
			ids = append(ids, id)
		}
		slices.Sort(ids)
	}

	var b strings.Builder
	for i, id := range ids {
		if i > 0 {
			b.WriteByte(';')
		}
		b.WriteString(strconv.FormatUint(id, 10))
		b.WriteByte(':')
		b.WriteString(strconv.FormatUint(settings[id], 10))
	}

	t.mu.Lock()
	if t.settings == "" && t.settingsSeen != nil {
		close(t.settingsSeen)
	}
	t.settings = b.String()
	t.mu.Unlock()
}

// waitSettings waits for the SETTINGS frame of the client, at most d. The client sends it first on its control
// stream, usually in the packets of the first request, but the server reads the control stream and the request
// streams in their own goroutines: the first request can reach the handler before the SETTINGS is read.
func (t *quicTrace) waitSettings(d time.Duration) {
	if t.settingsSeen == nil || t.waited.Swap(true) {
		return
	}
	select {
	case <-t.settingsSeen:
		return
	default:
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-t.settingsSeen:
	case <-timer.C:
	}
}

func (t *quicTrace) take(req *http.Request) *Request {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, r := range t.pending {
		if r.matches(req) {
			t.pending = append(t.pending[:i], t.pending[i+1:]...)
			return r
		}
	}
	return nil
}
