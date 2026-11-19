package fingerprint

import (
	"strconv"
	"time"
)

// tcpLive is what the kernel knows of a TCP connection when a request arrives (TCP_INFO and TCP_CONGESTION).
type tcpLive struct {
	sndMSS, rcvMSS, advMSS, pmtu uint32
	// rtt, rttVar and minRTT are in microseconds.
	rtt, rttVar, minRTT  uint32
	cwnd, retrans, lost  uint32
	sndWScale, rcvWScale int
	timestamps, sack     bool
	ecn                  bool
	cc                   string
}

// transportWriter writes the key=value pairs of the transport header.
type transportWriter struct {
	b []byte
}

func (w *transportWriter) key(k string) {
	if len(w.b) > 0 {
		w.b = append(w.b, ',')
	}
	w.b = append(w.b, k...)
	w.b = append(w.b, '=')
}

func (w *transportWriter) str(k, v string) {
	if v == "" {
		return
	}
	w.key(k)
	w.b = append(w.b, v...)
}

func (w *transportWriter) int(k string, v int64) {
	w.key(k)
	w.b = strconv.AppendInt(w.b, v, 10)
}

func (w *transportWriter) bool(k string, v bool) {
	w.key(k)
	if v {
		w.b = append(w.b, '1')
	} else {
		w.b = append(w.b, '0')
	}
}

// initialTTL returns the usual initial TTL closest above ttl, as p0f guesses it.
func initialTTL(ttl int) int {
	for _, initial := range []int{32, 64, 128, 255} {
		if ttl <= initial {
			return initial
		}
	}
	return 255
}

// transportHeader returns the value of the transport header for a request of the connection: the IP and TCP
// headers of the saved SYN, what the kernel knows of the TCP connection now, or what the QUIC handshake and the
// QUIC library tell, then the age of the connection and the rank of the request.
func transportHeader(info *connInfo, now time.Time, seq int64) string {
	w := transportWriter{b: make([]byte, 0, 640)}

	switch {
	case info.quic != nil:
		writeQUICTransport(&w, info)
	case info.synView != nil || info.tcpConn != nil:
		writeTCPTransport(&w, info)
	default:
		return ""
	}

	w.int("conn_age_us", now.Sub(info.accepted).Microseconds())
	w.int("req_seq", seq)

	return string(w.b)
}

func writeTCPTransport(w *transportWriter, info *connInfo) {
	w.str("proto", "tcp")

	if syn := info.synView; syn != nil {
		w.int("ipv", int64(syn.IPVersion))
		w.int("ttl", int64(syn.TTL))
		initial := initialTTL(syn.TTL)
		w.int("ttl_init", int64(initial))
		w.int("hops", int64(initial-syn.TTL))
		w.int("dscp", int64(syn.TrafficClass>>2))
		w.int("ecn", int64(syn.TrafficClass&0x3))
		if syn.IPVersion == 4 {
			w.bool("df", syn.DF)
			if syn.IPIDZero {
				w.str("ipid", "0")
			} else {
				w.str("ipid", "nz")
			}
			w.int("ipopt", int64(syn.IPOptionsLen))
		} else {
			w.bool("flow", syn.FlowLabel != 0)
		}
		w.int("syn_len", int64(syn.Length))
		w.int("syn_win", int64(syn.Window))
		if syn.MSS >= 0 {
			w.int("syn_mss", int64(syn.MSS))
			overhead := 40
			if syn.IPVersion == 6 {
				overhead = 60
			}
			w.int("mtu_est", int64(syn.MSS+overhead))
		}
		if syn.WScale >= 0 {
			w.int("syn_wscale", int64(syn.WScale))
		}
		w.bool("syn_sack", syn.SACK)
		w.bool("syn_ts", syn.Timestamps)
		w.bool("syn_ecn", syn.ECN)
		w.bool("syn_tfo", syn.FastOpen)
		w.str("syn_opts", syn.Options)
	}

	if info.tcpConn != nil {
		if live, err := readTCPLive(info.tcpConn); err == nil {
			w.int("mss_snd", int64(live.sndMSS))
			w.int("mss_rcv", int64(live.rcvMSS))
			w.int("mss_adv", int64(live.advMSS))
			w.int("pmtu", int64(live.pmtu))
			if live.sndWScale >= 0 {
				w.int("wscale_snd", int64(live.sndWScale))
				w.int("wscale_rcv", int64(live.rcvWScale))
			}
			w.bool("opt_ts", live.timestamps)
			w.bool("opt_sack", live.sack)
			w.bool("opt_ecn", live.ecn)
			if info.rtt > 0 {
				w.int("rtt_syn_us", info.rtt.Microseconds())
			}
			w.int("rtt_us", int64(live.rtt))
			w.int("rttvar_us", int64(live.rttVar))
			w.int("rtt_min_us", int64(live.minRTT))
			w.int("cwnd", int64(live.cwnd))
			w.int("retrans", int64(live.retrans))
			w.int("lost", int64(live.lost))
			w.str("cc", live.cc)
		}
	}
}

func writeQUICTransport(w *transportWriter, info *connInfo) {
	w.str("proto", "udp")
	if a := info.src.Addr(); a.Is4() {
		w.int("ipv", 4)
	} else {
		w.int("ipv", 6)
	}

	in := info.quicInitial
	if in.version != 0 {
		switch in.version {
		case quicVersion1:
			w.str("quic_ver", "1")
		case quicVersion2:
			w.str("quic_ver", "2")
		}
		w.int("initial_size", int64(in.size))
		w.int("dcid_len", int64(in.dcidLen))
		w.int("scid_len", int64(in.scidLen))
		w.int("initial_pkts", int64(in.packets))
	}

	stats := info.quic.ConnectionStats()
	w.int("rtt_us", stats.SmoothedRTT.Microseconds())
	w.int("rttvar_us", stats.MeanDeviation.Microseconds())
	w.int("rtt_min_us", stats.MinRTT.Microseconds())
	w.int("cwnd", int64(stats.CongestionWindow))
	w.int("lost", int64(stats.PacketsLost))
}
