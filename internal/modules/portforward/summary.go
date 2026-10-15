package portforward

import (
	"fmt"

	"github.com/wangwanjie/gately/internal/module"
)

// Summary 实现 module.Summarizer。
func (m *Module) Summary() []module.SummaryItem {
	m.mu.Lock()
	defer m.mu.Unlock()
	enabled, conns := 0, int64(0)
	for _, r := range m.cfg.Rules {
		if r.Enable {
			enabled++
		}
	}
	for _, r := range m.runners {
		conns += r.stats.ActiveConns.Load()
	}
	level := "ok"
	if len(m.errors) > 0 {
		level = "error"
	}
	return []module.SummaryItem{
		{Label: "运行中", Value: fmt.Sprintf("%d / %d", len(m.runners), enabled), Level: level},
		{Label: "活动连接", Value: fmt.Sprint(conns)},
	}
}
