package webservice

import (
	"fmt"

	"github.com/wangwanjie/gately/internal/module"
)

// Summary 实现 module.Summarizer。
func (m *Module) Summary() []module.SummaryItem {
	m.mu.Lock()
	defer m.mu.Unlock()
	sites, requests := 0, int64(0)
	for _, r := range m.runners {
		sites += len(r.sites)
		for _, s := range r.sites {
			requests += s.stats.Requests.Load()
		}
	}
	level := "ok"
	if len(m.errors) > 0 {
		level = "error"
	}
	return []module.SummaryItem{
		{Label: "运行中的监听", Value: fmt.Sprintf("%d / %d", len(m.runners), len(m.cfg.Listeners)), Level: level},
		{Label: "站点", Value: fmt.Sprint(sites)},
		{Label: "请求数", Value: fmt.Sprint(requests)},
	}
}
