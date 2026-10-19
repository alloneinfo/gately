package ddns

import (
	"github.com/wangwanjie/gately/internal/module"
)

// Metrics 实现 module.Metricer：每条启用记录最近一次同步是否成功及同步时间。
func (m *Module) Metrics() []module.Sample {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []module.Sample
	var syncAt []module.Sample
	for _, t := range m.cfg.Tasks {
		r, ok := m.runners[t.Key]
		if !ok {
			continue
		}
		st := r.snapshot()
		for _, rec := range t.Records {
			if !rec.Enable {
				continue
			}
			rs := st.Records[rec.Key]
			labels := map[string]string{"task": t.Name, "record": rec.Sub + "." + rec.Zone, "type": rec.Type}
			up := 0.0
			if rs.LastError == "" && !rs.LastCheck.IsZero() {
				up = 1
			}
			out = append(out, module.Sample{Name: "gately_ddns_record_up", Help: "Whether the last DDNS sync of the record succeeded (1) or failed / never ran (0)", Type: module.MetricGauge, Labels: labels, Value: up})
			if !rs.LastSync.IsZero() {
				syncAt = append(syncAt, module.Sample{Name: "gately_ddns_last_update_timestamp_seconds", Help: "Unix time when the record was last changed at the DNS provider", Type: module.MetricGauge, Labels: labels, Value: float64(rs.LastSync.Unix())})
			}
		}
	}
	return append(out, syncAt...)
}
