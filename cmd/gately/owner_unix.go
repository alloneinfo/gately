//go:build !windows

package main

import (
	"os"
	"syscall"
)

// matchOwner 把 path 的属主改为与 ref 相同；失败时忽略（例如非 root 运行且属主本就一致）。
func matchOwner(path, ref string) {
	fi, err := os.Stat(ref)
	if err != nil {
		return
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}
	_ = os.Chown(path, int(st.Uid), int(st.Gid))
}
