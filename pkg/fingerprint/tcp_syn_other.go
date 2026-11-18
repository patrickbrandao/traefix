//go:build !linux || 386

package fingerprint

import (
	"errors"
	"net"
	"time"
)

const savedSYNSupported = false

var errSavedSYNUnsupported = errors.New("the saved SYN is only available on Linux")

func saveSYN(*net.TCPListener) error {
	return errSavedSYNUnsupported
}

func readSYN(*net.TCPConn) ([]byte, time.Duration, error) {
	return nil, 0, errSavedSYNUnsupported
}
