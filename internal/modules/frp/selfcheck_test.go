package frp

import (
	"context"
	"io"
	"log"
	"testing"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
)

func TestSelfCheckConflicts(t *testing.T) {
	st, _ := config.Open(t.TempDir())
	m := New()
	logs := logbuf.New(100, log.New(io.Discard, "", 0))
	if err := m.Init(module.Env{Store: st, Logs: logs}); err != nil {
		t.Fatal(err)
	}
	if got := m.SelfCheck(context.Background(), module.CheckEnv{}); len(got) != 0 {
		t.Fatalf("未启用 frp 时不应报告: %+v", got)
	}
	m.cfg.Client.Enable = true
	m.cfg.Proxies = []Proxy{{Name: "nas", Enable: true}, {Name: "ai", Enable: true}}
	got := m.SelfCheck(context.Background(), module.CheckEnv{})
	if len(got) != 1 || got[0].ID != "frp.ok" || got[0].Params["count"] != "2" {
		t.Fatalf("无冲突时应报告正常: %+v", got)
	}

	m.cfg.Proxies = append(m.cfg.Proxies, Proxy{Name: "nas", Enable: true}, Proxy{Name: "off", Enable: false}, Proxy{Name: "off", Enable: false})
	frpc := logs.Logger("frpc")
	frpc.Warnf("2026-10-19 10:00:00.000 [W] [client/proxy/proxy_manager.go:173] [5f1c] [ai] start error: proxy [ai] already exists")
	frpc.Warnf("2026-10-19 10:00:10.000 [W] [client/proxy/proxy_manager.go:173] [5f1c] [ai] start error: proxy [ai] already exists")
	frpc.Warnf("[5f1c] [web] start error: router config conflict")
	frpc.Warnf("[5f1c] [ssh] start error: port already used")
	frpc.Warnf("[5f1c] [other] start error: connection reset")
	logs.Logger("frps").Warnf("[2b7e] new proxy [gitea] type [http] error: proxy [gitea] already exists")
	// 同一个冲突在 frps 日志中也会出现，不应重复报告
	logs.Logger("frps").Warnf("[W] [server/control.go:756] [5f1c] new proxy [ai] type [tcp] error: proxy [ai] already exists")
	logs.Logger("frps").Warnf("[W] [server/control.go:756] [9a0d] new proxy [remote] type [tcp] error: port already used")
	logs.Logger("frps").Warnf("[W] [server/control.go:756] [9a0d] new proxy [other] type [tcp] error: authorization failed")
	logs.Logger("ddns").Warnf("proxy [x] already exists")
	got = m.SelfCheck(context.Background(), module.CheckEnv{})
	ids := map[string]int{}
	proxies := map[string]bool{}
	for _, r := range got {
		ids[r.ID]++
		proxies[r.Params["proxy"]] = true
	}
	if ids["frp.duplicate"] != 1 || ids["frp.conflict"] != 5 || ids["frp.ok"] != 0 {
		t.Fatalf("冲突统计错误: %v %+v", ids, got)
	}
	for _, p := range []string{"nas", "ai", "web", "ssh", "gitea", "remote"} {
		if !proxies[p] {
			t.Errorf("应报告代理 %s", p)
		}
	}
	if proxies["other"] || proxies["x"] {
		t.Fatal("不应报告非冲突错误或其他模块的日志")
	}
}
