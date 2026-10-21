// Package fsperm 在创建文件和目录后显式设置权限。
//
// 部分文件系统会忽略创建时传入的权限：例如飞牛 fnOS 以 trimacl 挂载的 btrfs 卷，
// 新建的文件和目录一律按继承规则变成 0755，只有创建后再 chmod 才生效。
// 配置目录里有令牌、私钥、配置历史和访问日志，在其中创建文件都应当经过这里。
package fsperm

import (
	"io/fs"
	"os"
	"path/filepath"
)

// MkdirAll 与 os.MkdirAll 相同，随后把 dir 本身的权限设为 perm（不改动已存在的上级目录）。
func MkdirAll(dir string, perm os.FileMode) error {
	if err := os.MkdirAll(dir, perm); err != nil {
		return err
	}
	return os.Chmod(dir, perm)
}

// WriteFile 与 os.WriteFile 相同，随后把文件权限设为 perm。
func WriteFile(name string, data []byte, perm os.FileMode) error {
	if err := os.WriteFile(name, data, perm); err != nil {
		return err
	}
	return os.Chmod(name, perm)
}

// Tighten 收紧 root（文件或目录）及其下已有内容的权限：带有组或其他用户权限的目录改为 0700，
// 不是 0600 的普通文件改为 0600（这些目录只存数据，执行位一并去掉）。不跟随符号链接；root 不存在时什么也不做。
// 用于修复旧版本在上述文件系统上创建的文件，返回改动的数量。
//
// 文件不能只看组和其他用户位：飞牛的 trimacl 卷在收紧目录时会把子项一起改成 0700，保留了执行位。
func Tighten(root string) int {
	n := 0
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		mode := info.Mode().Perm()
		var perm os.FileMode
		switch {
		case d.IsDir() && mode&0o077 != 0:
			perm = 0o700
		case info.Mode().IsRegular() && mode != 0o600:
			perm = 0o600
		default:
			return nil
		}
		if os.Chmod(p, perm) == nil {
			n++
		}
		return nil
	})
	return n
}
