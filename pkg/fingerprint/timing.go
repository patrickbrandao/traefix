package fingerprint

import (
	"time"
)

// connTiming holds the milestones of the start of a connection, protected by connInfo.mu.
//
// Over TCP, the arrival times (first byte, end of the ClientHello, first client flight after the server flight)
// come from the kernel when the tcp option is set on Linux, and from the Traefix clock otherwise. The other
// milestones (server flight written, TLS handshake done) always come from the Traefix clock. Both clocks are
// CLOCK_REALTIME.
type connTiming struct {
	kernel bool

	// firstByte is the arrival of the first byte from the client: the start of the ClientHello over TLS,
	// the first Initial datagram over QUIC.
	firstByte time.Time
	// helloDone is the arrival of the end of the ClientHello.
	helloDone time.Time
	// serverFlight is the end of the last write of the server before the client answers it: the ServerHello
	// flight, up to the server Finished in TLS 1.3.
	serverFlight time.Time
	// clientFlight is the arrival of the first client bytes after the server flight: the client Finished in
	// TLS 1.3, the ClientKeyExchange flight in TLS 1.2.
	clientFlight time.Time
	// handshakeDone is the end of the TLS handshake, or of the QUIC handshake.
	handshakeDone time.Time

	// done is set once the TLS handshake is timed, or after the first byte of a connection without TLS.
	done bool
}

// settleQUICHandshake gives the end of the QUIC handshake when the goroutine watching it has not run yet by the time
// the first request arrives: the client sends the request in the flight that completes the handshake, so the request
// is the first moment the handshake is known to be over.
func (info *connInfo) settleQUICHandshake(at time.Time) {
	if info.quic == nil {
		return
	}
	info.mu.Lock()
	defer info.mu.Unlock()
	if !info.timing.handshakeDone.IsZero() {
		return
	}
	select {
	case <-info.quic.HandshakeComplete():
		info.timing.handshakeDone = at
	default:
	}
}

func micros(from, to time.Time) (int64, bool) {
	if from.IsZero() || to.IsZero() {
		return 0, false
	}
	return to.Sub(from).Microseconds(), true
}

// timingHeader returns the value of the timing header for a request starting at start.
func timingHeader(info *connInfo, start time.Time) string {
	info.mu.Lock()
	t := info.timing
	info.mu.Unlock()

	w := transportWriter{b: make([]byte, 0, 256)}
	if t.kernel {
		w.str("src", "kernel")
	} else {
		w.str("src", "user")
	}

	if info.quic != nil {
		w.str("proto", "udp")
		if v, ok := micros(t.firstByte, t.handshakeDone); ok {
			w.int("initial_to_hs_us", v)
		}
		// The accept of a QUIC connection comes after the server processed the first Initial: the request times
		// (req_us) count from the accept, the handshake from the Initial.
		if v, ok := micros(t.firstByte, info.accepted); ok {
			w.int("initial_to_acc_us", v)
		}
		stats := info.quic.ConnectionStats()
		w.int("quic_rtt_us", stats.SmoothedRTT.Microseconds())
		w.int("quic_min_rtt_us", stats.MinRTT.Microseconds())
		if v, ok := micros(info.accepted, start); ok {
			w.int("req_us", v)
		}
		return string(w.b)
	}

	w.str("proto", "tcp")
	tcpRTT := info.rtt.Microseconds()
	if tcpRTT > 0 {
		w.int("tcp_rtt_us", tcpRTT)
	}
	if v, ok := micros(info.accepted, t.firstByte); ok {
		w.int("est_to_first_us", v)
	}
	if v, ok := micros(t.firstByte, t.helloDone); ok {
		w.int("ch_us", v)
	}
	if v, ok := micros(t.helloDone, t.serverFlight); ok {
		w.int("srv_us", v)
	}
	tlsRTT, hasTLSRTT := micros(t.serverFlight, t.clientFlight)
	if hasTLSRTT {
		w.int("tls_rtt_us", tlsRTT)
	}
	if v, ok := micros(t.firstByte, t.handshakeDone); ok {
		w.int("tls_us", v)
	}
	if hasTLSRTT && tcpRTT > 0 {
		w.int("rtt_gap_us", tlsRTT-tcpRTT)
	}
	if v, ok := micros(info.accepted, start); ok {
		w.int("req_us", v)
	}

	return string(w.b)
}

// timingRecord is the timing of the connection record, in microseconds from the accept.
type timingRecord struct {
	Source          string     `json:"source"`
	FirstByte       *time.Time `json:"firstByte,omitempty"`
	TCPRTTMicros    *int64     `json:"tcpRttMicros,omitempty"`
	EstToFirst      *int64     `json:"estToFirstMicros,omitempty"`
	ClientHello     *int64     `json:"clientHelloMicros,omitempty"`
	Server          *int64     `json:"serverMicros,omitempty"`
	TLSRTT          *int64     `json:"tlsRttMicros,omitempty"`
	Handshake       *int64     `json:"handshakeMicros,omitempty"`
	RTTGap          *int64     `json:"rttGapMicros,omitempty"`
	InitialToHS     *int64     `json:"initialToHandshakeMicros,omitempty"`
	InitialToAccept *int64     `json:"initialToAcceptMicros,omitempty"`
	QUICRTTMicros   *int64     `json:"quicRttMicros,omitempty"`
	QUICMinRTTMicro *int64     `json:"quicMinRttMicros,omitempty"`
}

func newTimingRecord(info *connInfo) *timingRecord {
	info.mu.Lock()
	t := info.timing
	info.mu.Unlock()

	rec := &timingRecord{Source: "user"}
	if t.kernel {
		rec.Source = "kernel"
	}
	if !t.firstByte.IsZero() {
		first := t.firstByte.UTC()
		rec.FirstByte = &first
	}
	ptr := func(v int64, ok bool) *int64 {
		if !ok {
			return nil
		}
		return &v
	}

	if info.quic != nil {
		rec.InitialToHS = ptr(micros(t.firstByte, t.handshakeDone))
		rec.InitialToAccept = ptr(micros(t.firstByte, info.accepted))
		stats := info.quic.ConnectionStats()
		rec.QUICRTTMicros = ptr(stats.SmoothedRTT.Microseconds(), true)
		rec.QUICMinRTTMicro = ptr(stats.MinRTT.Microseconds(), true)
		return rec
	}

	tcpRTT := info.rtt.Microseconds()
	rec.TCPRTTMicros = ptr(tcpRTT, tcpRTT > 0)
	rec.EstToFirst = ptr(micros(info.accepted, t.firstByte))
	rec.ClientHello = ptr(micros(t.firstByte, t.helloDone))
	rec.Server = ptr(micros(t.helloDone, t.serverFlight))
	tlsRTT, ok := micros(t.serverFlight, t.clientFlight)
	rec.TLSRTT = ptr(tlsRTT, ok)
	rec.Handshake = ptr(micros(t.firstByte, t.handshakeDone))
	rec.RTTGap = ptr(tlsRTT-tcpRTT, ok && tcpRTT > 0)
	return rec
}
