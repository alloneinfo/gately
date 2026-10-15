package waf

import (
	"fmt"

	"github.com/wangwanjie/gately/internal/module"
)

// Summary 实现 module.Summarizer。
func (m *Module) Summary() []module.SummaryItem {
	m.mu.Lock()
	n := len(m.cfg.Instances)
	m.mu.Unlock()
	var blocked int64
	m.state.Range(func(_, v any) bool {
		blocked += v.(*compiled).blocked.Load()
		return true
	})
	level := "info"
	if blocked > 0 {
		level = "warn"
	}
	return []module.SummaryItem{{Label: "实例", Value: fmt.Sprint(n)}, {Label: "已拦截", Value: fmt.Sprint(blocked), Level: level}}
}
