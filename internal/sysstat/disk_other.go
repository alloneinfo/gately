//go:build !linux && !darwin && !freebsd

package sysstat

func diskUsage(dir string) *Disk {
	return nil
}
