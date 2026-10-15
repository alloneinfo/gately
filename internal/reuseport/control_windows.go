//go:build windows

package reuseport

import (
	"syscall"
)

// control 在 Windows 上只能设置 SO_REUSEADDR（语义接近 Unix 的 SO_REUSEPORT）。
func control(_, _ string, c syscall.RawConn) error {
	var serr error
	err := c.Control(func(fd uintptr) {
		serr = syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	})
	if err != nil {
		return err
	}
	return serr
}
