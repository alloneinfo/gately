package stun

import (
	"fmt"

	"github.com/wangwanjie/gately/internal/module"
)

// Summary 实现 module.Summarizer。
func (m *Module) Summary() []module.SummaryItem {
	m.mu.Lock()
	defer m.mu.Unlock()
	ok, bad := 0, 0
	for _, r := range m.runners {
		switch r.snapshot().Status {
		case StatusOK:
			ok++
		case StatusFailed, StatusRetrying:
			bad++
		}
	}
	level := "ok"
	if bad > 0 {
		level = "warn"
	}
	return []module.SummaryItem{
		{Label: "已穿透", Value: fmt.Sprintf("%d / %d", ok, len(m.runners)), Level: level},
		{Label: "异常", Value: fmt.Sprint(bad), Level: level},
	}
}
