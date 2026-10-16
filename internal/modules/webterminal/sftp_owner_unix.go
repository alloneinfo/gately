//go:build !windows

package webterminal

import (
	"io/fs"
	"os/user"
	"strconv"
	"sync"
	"syscall"
)

// 用户名缓存，避免列目录时对同一 uid 反复查询。
var ownerCache sync.Map

// localOwner 返回本机文件的属主用户名，查询失败时返回数字 uid。
func localOwner(fi fs.FileInfo) string {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return ""
	}
	uid := strconv.FormatUint(uint64(st.Uid), 10)
	if v, ok := ownerCache.Load(uid); ok {
		return v.(string)
	}
	name := uid
	if u, err := user.LookupId(uid); err == nil {
		name = u.Username
	}
	ownerCache.Store(uid, name)
	return name
}
