//go:build linux || darwin || freebsd

package sysstat

import "golang.org/x/sys/unix"

// diskUsage 返回 dir 所在分区的容量。已用空间按“总量 - 空闲”计算，与 df 的口径一致。
func diskUsage(dir string) *Disk {
	var st unix.Statfs_t
	if err := unix.Statfs(dir, &st); err != nil {
		return nil
	}
	bs := uint64(st.Bsize)
	total := uint64(st.Blocks) * bs
	if total == 0 {
		return nil
	}
	free := uint64(st.Bavail) * bs
	used := total - uint64(st.Bfree)*bs
	return &Disk{Path: dir, Total: total, Used: used, Free: free}
}
