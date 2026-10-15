//go:build unix

package reuseport

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// control 在 bind 之前设置 SO_REUSEADDR 与 SO_REUSEPORT。
func control(_, _ string, c syscall.RawConn) error {
	var serr error
	err := c.Control(func(fd uintptr) {
		if serr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); serr != nil {
			return
		}
		serr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
	})
	if err != nil {
		return err
	}
	return serr
}
