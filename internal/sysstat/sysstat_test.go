package sysstat

import (
	"encoding/json"
	"runtime"
	"testing"
)

// TestSnapshot 确认采集不会出错，且基本字段有值。
func TestSnapshot(t *testing.T) {
	c := New(t.TempDir())
	s := c.Snapshot()
	if s.CPU.Cores <= 0 || s.Process.PID <= 0 {
		t.Fatalf("基本字段为空: %+v", s)
	}
	s = c.Snapshot()
	// Linux 上 /proc 可读，内存、使用率与磁盘都应有值（CI 在 Linux 上运行）
	if runtime.GOOS == "linux" {
		if s.Memory == nil || s.CPU.Usage == nil || s.Process.CPU == nil || s.Disk == nil || s.Process.RSS == 0 {
			t.Fatalf("Linux 采集字段缺失: %+v", s)
		}
	}
	b, _ := json.MarshalIndent(s, "", "  ")
	t.Log(string(b))
}
