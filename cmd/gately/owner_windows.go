//go:build windows

package main

// matchOwner 在 Windows 上无需处理：文件权限继承自配置目录。
func matchOwner(path, ref string) {}
