//go:build windows

package webterminal

import "io/fs"

// localOwner 在 Windows 上不提供属主信息。
func localOwner(fi fs.FileInfo) string {
	return ""
}
