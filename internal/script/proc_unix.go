//go:build unix

package script

import (
	"os/exec"
	"syscall"
)

// Prepare 让命令在独立进程组中运行，超时或取消时结束整个进程组，避免后台子进程残留并卡住输出读取。
func Prepare(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = waitDelay
}
