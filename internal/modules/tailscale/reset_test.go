package tailscale

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
)

// TestReset 验证恢复默认配置会恢复默认设置并删除默认状态目录，自定义状态目录保留。
func TestReset(t *testing.T) {
	dir := t.TempDir()
	store, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	m := New()
	if err := m.Init(module.Env{Store: store, Logs: logbuf.New(100, log.New(io.Discard, "", 0))}); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(dir, "tailscale")
	if err := os.MkdirAll(state, 0o700); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(state, "tailscaled.state"), []byte("{}"), 0o600)
	m.settings.Hostname, m.settings.AuthKey, m.settings.InstallProxy = "nas", "tskey-x", "http://127.0.0.1:7890"

	m.mu.Lock()
	err = m.resetLocked()
	m.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	def := DefaultSettings()
	if m.settings.Hostname != "" || m.settings.AuthKey != "" || m.settings.InstallProxy != "" || m.settings.ManageDaemon != def.ManageDaemon || m.settings.Userspace != def.Userspace {
		t.Errorf("设置未恢复默认: %+v", m.settings)
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Errorf("默认状态目录应被删除: %v", err)
	}

	// 自定义状态目录不删除
	custom := filepath.Join(t.TempDir(), "ts")
	_ = os.MkdirAll(custom, 0o700)
	m.settings.StateDir = custom
	m.mu.Lock()
	err = m.resetLocked()
	m.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(custom); err != nil {
		t.Errorf("自定义状态目录应保留: %v", err)
	}
}
