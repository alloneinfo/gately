package ssl

import (
	"fmt"
	"time"

	"github.com/wangwanjie/gately/internal/module"
)

// Summary 实现 module.Summarizer：证书数量与最近到期时间。
func (m *Module) Summary() []module.SummaryItem {
	m.mu.Lock()
	defer m.mu.Unlock()
	enabled := 0
	var soonest time.Time
	var soonestName string
	for i := range m.cfg.Certs {
		c := &m.cfg.Certs[i]
		if !c.Enable || c.CertPEM == "" {
			continue
		}
		enabled++
		if info, err := certInfo(c.CertPEM); err == nil && (soonest.IsZero() || info.NotAfter.Before(soonest)) {
			soonest, soonestName = info.NotAfter, label(c)
		}
	}
	items := []module.SummaryItem{{Label: "启用的证书", Value: fmt.Sprint(enabled)}}
	if !soonest.IsZero() {
		days := int(time.Until(soonest).Hours() / 24)
		level := "ok"
		if days < 7 {
			level = "error"
		} else if days < 20 {
			level = "warn"
		}
		items = append(items, module.SummaryItem{Label: "最近到期", Value: fmt.Sprintf("%s 剩余 %d 天", soonestName, days), Level: level})
	}
	return items
}
