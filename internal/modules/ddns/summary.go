package ddns

import (
	"fmt"

	"github.com/wangwanjie/gately/internal/module"
)

// Summary 实现 module.Summarizer。
func (m *Module) Summary() []module.SummaryItem {
	m.mu.Lock()
	defer m.mu.Unlock()
	records, failed := 0, 0
	var ip4, ip6 string
	for _, t := range m.cfg.Tasks {
		r, ok := m.runners[t.Key]
		if !ok {
			continue
		}
		st := r.snapshot()
		if ip4 == "" {
			ip4 = st.IPv4
		}
		if ip6 == "" {
			ip6 = st.IPv6
		}
		for _, rec := range t.Records {
			if rec.Enable {
				records++
				if st.Records[rec.Key].LastError != "" {
					failed++
				}
			}
		}
	}
	level := "ok"
	if failed > 0 {
		level = "error"
	}
	items := []module.SummaryItem{{Label: "运行中的记录", Value: fmt.Sprint(records)}, {Label: "同步失败", Value: fmt.Sprint(failed), Level: level}}
	if ip4 != "" {
		items = append(items, module.SummaryItem{Label: "IPv4", Value: ip4})
	}
	if ip6 != "" {
		items = append(items, module.SummaryItem{Label: "IPv6", Value: ip6})
	}
	return items
}
