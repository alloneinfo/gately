package sysstat

import (
	"encoding/json"
	"runtime"
	"testing"
	"time"
)

// TestSnapshot 确认采集不会出错，且基本字段有值。
func TestSnapshot(t *testing.T) {
	c := New(t.TempDir())
	s := c.Snapshot()
	if s.CPU.Cores <= 0 || s.Process.PID <= 0 {
		t.Fatalf("基本字段为空: %+v", s)
	}
	// 紧接着再采会与上一次基准相隔仅几毫秒，/proc/stat 的计数可能还没变化，使用率为空；等够最小采样间隔
	time.Sleep(minInterval)
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

func TestIsVirtual(t *testing.T) {
	cases := map[string]bool{
		// OpenWrt 的局域网、广域网网桥要展示
		"br-lan": false, "br-wan": false, "br-guest": false, "eth0": false, "tailscale0": false,
		// Docker 自定义网络网桥与常见虚拟网卡隐藏
		"br-1a2b3c4d5e6f": true, "docker0": true, "veth12ab": true, "virbr0": true,
		// 长度或字符不符合 Docker 规则的 br- 网卡按真实网卡处理
		"br-1a2b3c4d5e6": false, "br-1A2B3C4D5E6F": false,
	}
	for name, want := range cases {
		if got := isVirtual(name); got != want {
			t.Errorf("isVirtual(%q) = %v，期望 %v", name, got, want)
		}
	}
}
