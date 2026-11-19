//go:build !linux || 386

package fingerprint

import (
	"net"
	"time"
)

const kernelTimestampsSupported = false

func enableKernelTimestamps(*net.TCPConn) bool { return false }

func disableKernelTimestamps(*net.TCPConn) {}

func readTimed(c *net.TCPConn, p []byte) (int, time.Time, error) {
	n, err := c.Read(p)
	return n, time.Time{}, err
}
