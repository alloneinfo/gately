//go:build !unix

package script

import "os/exec"

// Prepare 设置取消后等待输出管道关闭的上限，防止子进程持有管道导致永久阻塞。
func Prepare(cmd *exec.Cmd) {
	cmd.WaitDelay = waitDelay
}
