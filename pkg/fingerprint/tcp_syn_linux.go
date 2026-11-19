//go:build linux && !386

package fingerprint

import (
	"fmt"
	"net"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// maxSavedSYN bounds the saved SYN: an IPv6 header and a TCP header with 40 bytes of options take 100 bytes,
// IPv4 options and IPv6 extension headers can add more.
const maxSavedSYN = 512

const savedSYNSupported = true

// saveSYN asks the kernel to keep the SYN of the connections the listener accepts (TCP_SAVE_SYN, Linux 4.2).
func saveSYN(l *net.TCPListener) error {
	rc, err := l.SyscallConn()
	if err != nil {
		return err
	}

	var setErr error
	if err := rc.Control(func(fd uintptr) {
		setErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_SAVE_SYN, 1)
	}); err != nil {
		return err
	}
	if setErr != nil {
		return fmt.Errorf("setting TCP_SAVE_SYN: %w", setErr)
	}

	return nil
}

// readSYN returns the SYN the kernel saved for the connection, from its IP header, which the kernel then frees,
// and the round-trip time it measured during the handshake.
func readSYN(c *net.TCPConn) ([]byte, time.Duration, error) {
	rc, err := c.SyscallConn()
	if err != nil {
		return nil, 0, err
	}

	var syn []byte
	var rtt time.Duration
	var readErr error
	err = rc.Control(func(fd uintptr) {
		buf := make([]byte, maxSavedSYN)
		n := uint32(len(buf))
		// unix.GetsockoptString would cut the bytes at the first zero byte.
		_, _, errno := unix.Syscall6(unix.SYS_GETSOCKOPT, fd, unix.IPPROTO_TCP, unix.TCP_SAVED_SYN,
			uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)), 0)
		if errno != 0 {
			readErr = errno
			return
		}
		syn = buf[:n:n]

		if info, err := unix.GetsockoptTCPInfo(int(fd), unix.IPPROTO_TCP, unix.TCP_INFO); err == nil {
			rtt = time.Duration(info.Rtt) * time.Microsecond
		}
	})
	if err != nil {
		return nil, 0, err
	}

	return syn, rtt, readErr
}

// The bits of tcpi_options (include/uapi/linux/tcp.h).
const (
	tcpiOptTimestamps = 1
	tcpiOptSACK       = 2
	tcpiOptWScale     = 4
	tcpiOptECN        = 8
)

// readTCPLive reads what the kernel knows of the connection now (TCP_INFO and TCP_CONGESTION).
func readTCPLive(c *net.TCPConn) (*tcpLive, error) {
	rc, err := c.SyscallConn()
	if err != nil {
		return nil, err
	}

	var live *tcpLive
	var readErr error
	err = rc.Control(func(fd uintptr) {
		info, err := unix.GetsockoptTCPInfo(int(fd), unix.IPPROTO_TCP, unix.TCP_INFO)
		if err != nil {
			readErr = err
			return
		}
		// tcpi_snd_wscale and tcpi_rcv_wscale are the 4-bit fields of the byte after tcpi_options,
		// which the unix.TCPInfo struct leaves as padding.
		wscale := (*[8]byte)(unsafe.Pointer(info))[6]
		live = &tcpLive{
			sndMSS:     info.Snd_mss,
			rcvMSS:     info.Rcv_mss,
			advMSS:     info.Advmss,
			pmtu:       info.Pmtu,
			rtt:        info.Rtt,
			rttVar:     info.Rttvar,
			minRTT:     info.Min_rtt,
			cwnd:       info.Snd_cwnd,
			retrans:    info.Total_retrans,
			lost:       info.Lost,
			timestamps: info.Options&tcpiOptTimestamps != 0,
			sack:       info.Options&tcpiOptSACK != 0,
			ecn:        info.Options&tcpiOptECN != 0,
		}
		if info.Options&tcpiOptWScale != 0 {
			live.sndWScale, live.rcvWScale = int(wscale&0xf), int(wscale>>4)
		} else {
			live.sndWScale, live.rcvWScale = -1, -1
		}
		if cc, err := unix.GetsockoptString(int(fd), unix.IPPROTO_TCP, unix.TCP_CONGESTION); err == nil {
			live.cc = cc
		}
	})
	if err != nil {
		return nil, err
	}

	return live, readErr
}
