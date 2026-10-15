package cron

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/webhook"
)

func newModule(t *testing.T) *Module {
	st, _ := config.Open(t.TempDir())
	m := New()
	if err := m.Init(module.Env{Store: st, Logs: logbuf.New(100, log.New(io.Discard, "", 0))}); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestValidate(t *testing.T) {
	good := []string{"0 3 * * *", "*/5 * * * * *", "@every 10m", "@daily"}
	for _, s := range good {
		task := Task{Name: "a", Spec: s, Command: "true"}
		if err := task.Validate(); err != nil {
			t.Errorf("%q 应合法: %v", s, err)
		}
	}
	bad := []Task{{Name: "a", Spec: "bad", Command: "x"}, {Name: "", Spec: "@daily", Command: "x"}, {Name: "a", Spec: "@daily"}, {Name: "a", Spec: "@daily", Type: "http"}}
	for i, task := range bad {
		if err := task.Validate(); err == nil {
			t.Errorf("第 %d 个非法任务未报错", i)
		}
	}
}

func TestScheduledCommandAndHTTP(t *testing.T) {
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("pong"))
	}))
	defer srv.Close()
	m := newModule(t)
	m.cfg.Tasks = []Task{
		{Key: "c", Name: "cmd", Enable: true, Spec: "@every 1s", Type: TypeCommand, Command: `echo "run $taskName"`},
		{Key: "h", Name: "hook", Enable: true, Spec: "@every 1s", Type: TypeHTTP, HTTP: webhook.Config{Enable: true, URL: srv.URL, Method: "GET"}},
		{Key: "off", Name: "off", Enable: false, Spec: "@every 1s", Command: "echo never"},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := m.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer m.Stop()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		done := len(m.history["c"]) > 0 && len(m.history["h"]) > 0
		m.mu.Unlock()
		if done {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.history["c"]) == 0 || !strings.Contains(m.history["c"][0].Output, "run cmd") || !m.history["c"][0].OK {
		t.Fatalf("命令任务未正确执行: %+v", m.history["c"])
	}
	if hits.Load() == 0 || !strings.Contains(m.history["h"][0].Output, "pong") {
		t.Fatalf("HTTP 任务未执行: %+v", m.history["h"])
	}
	if len(m.history["off"]) != 0 {
		t.Fatal("停用的任务不应执行")
	}
}

func TestManualRunFailure(t *testing.T) {
	m := newModule(t)
	m.cfg.Tasks = []Task{{Key: "f", Name: "fail", Spec: "@daily", Command: "echo boom; exit 3"}}
	m.execute("f", true)
	h := m.history["f"]
	if len(h) != 1 || h[0].OK || !h[0].Manual || !strings.Contains(h[0].Output, "boom") {
		t.Fatalf("失败记录错误: %+v", h)
	}
}
