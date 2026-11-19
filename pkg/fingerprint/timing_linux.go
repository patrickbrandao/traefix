//go:build linux && !386

package fingerprint

import (
	"io"
	"net"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const kernelTimestampsSupported = true

// enableKernelTimestamps asks the kernel to stamp the received segments of the connection (SO_TIMESTAMPNS),
// whose times readTimed reads with the data.
func enableKernelTimestamps(c *net.TCPConn) bool {
	rc, err := c.SyscallConn()
	if err != nil {
		return false
	}
	var setErr error
	if err := rc.Control(func(fd uintptr) {
		setErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_TIMESTAMPNS, 1)
	}); err != nil || setErr != nil {
		return false
	}
	return true
}

// disableKernelTimestamps stops the stamping once the handshake is timed.
func disableKernelTimestamps(c *net.TCPConn) {
	if rc, err := c.SyscallConn(); err == nil {
		_ = rc.Control(func(fd uintptr) {
			_ = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_TIMESTAMPNS, 0)
		})
	}
}

// readTimed reads like net.Conn.Read, with recvmsg, and returns the time the kernel received the last segment
// of the bytes read. It waits through the network poller, so that the deadlines of the connection apply.
func readTimed(c *net.TCPConn, p []byte) (int, time.Time, error) {
	rc, err := c.SyscallConn()
	if err != nil {
		return 0, time.Time{}, err
	}

	var oob [64]byte
	var n, oobn int
	var readErr error
	err = rc.Read(func(fd uintptr) bool {
		n, oobn, _, _, readErr = unix.Recvmsg(int(fd), p, oob[:], 0)
		return readErr != unix.EAGAIN
	})
	if err != nil {
		return 0, time.Time{}, err
	}
	if readErr != nil {
		return 0, time.Time{}, &net.OpError{Op: "read", Net: "tcp", Source: c.LocalAddr(), Addr: c.RemoteAddr(), Err: readErr}
	}
	if n == 0 && len(p) > 0 {
		return 0, time.Time{}, io.EOF
	}

	var at time.Time
	if msgs, err := unix.ParseSocketControlMessage(oob[:oobn]); err == nil {
		for _, m := range msgs {
			if m.Header.Level == unix.SOL_SOCKET && m.Header.Type == unix.SCM_TIMESTAMPNS && len(m.Data) >= int(unsafe.Sizeof(unix.Timespec{})) {
				ts := (*unix.Timespec)(unsafe.Pointer(&m.Data[0]))
				at = time.Unix(ts.Unix())
			}
		}
	}

	return n, at, nil
}
