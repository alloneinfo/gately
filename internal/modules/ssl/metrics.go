package ssl

import (
	"time"

	"github.com/wangwanjie/gately/internal/module"
)

// Metrics 实现 module.Metricer：证书到期时间与剩余天数。
func (m *Module) Metrics() []module.Sample {
	m.mu.Lock()
	defer m.mu.Unlock()
	var expiry, days []module.Sample
	now := time.Now()
	for i := range m.cfg.Certs {
		c := &m.cfg.Certs[i]
		if !c.Enable || c.CertPEM == "" {
			continue
		}
		info, err := certInfo(c.CertPEM)
		if err != nil {
			continue
		}
		l := map[string]string{"cert": label(c)}
		expiry = append(expiry, module.Sample{Name: "gately_cert_expiry_timestamp_seconds", Help: "Unix time when the certificate expires", Type: module.MetricGauge, Labels: l, Value: float64(info.NotAfter.Unix())})
		days = append(days, module.Sample{Name: "gately_cert_days_remaining", Help: "Days until the certificate expires (negative once expired)", Type: module.MetricGauge, Labels: l, Value: info.NotAfter.Sub(now).Hours() / 24})
	}
	return append(expiry, days...)
}
