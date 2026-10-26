package cloudflared

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
)

type testRouter struct{ mux *http.ServeMux }

func (tr testRouter) HandleFunc(p string, h http.HandlerFunc) {
	method, path, _ := strings.Cut(p, " ")
	tr.mux.HandleFunc(method+" /api/cloudflared"+path, h)
}

// TestReset 验证恢复默认配置会停止并删除全部隧道、清空设置，保留安装的程序。
func TestReset(t *testing.T) {
	if goruntime.GOOS == "windows" {
		t.Skip("依赖 sh")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin", "cloudflared")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexec sleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	store, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	m := New()
	if err := m.Init(module.Env{Store: store, Logs: logbuf.New(100, log.New(io.Discard, "", 0))}); err != nil {
		t.Fatal(err)
	}
	m.cfg.Settings.BinPath = bin
	m.cfg.Install.Proxy = "http://127.0.0.1:7890"
	m.cfg.Tunnels = []Tunnel{{Key: "k", Name: "t", Enable: true, Mode: ModeQuick, Service: "http://localhost:8080"}}
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer m.Stop()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && !m.runs["k"].sup.State().Running {
		time.Sleep(20 * time.Millisecond)
	}
	mux := http.NewServeMux()
	m.Routes(testRouter{mux})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/cloudflared/reset", "application/json", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("恢复默认配置失败: %v %v", err, resp)
	}
	resp.Body.Close()
	if len(m.runs) != 0 || len(m.cfg.Tunnels) != 0 || m.cfg.Settings.BinPath != "" || m.cfg.Install.Proxy != "" {
		t.Errorf("配置未恢复默认: %+v %d", m.cfg, len(m.runs))
	}
	if _, err := os.Stat(bin); err != nil {
		t.Errorf("程序应保留: %v", err)
	}
}
