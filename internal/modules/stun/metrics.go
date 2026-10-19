package stun

import (
	"sort"

	"github.com/wangwanjie/gately/internal/module"
)

// Metrics 实现 module.Metricer：每条规则是否已穿透，以及当前映射保持的时长。
func (m *Module) Metrics() []module.Sample {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := make([]string, 0, len(m.runners))
	for k := range m.runners {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []module.Sample
	for _, k := range keys {
		r := m.runners[k]
		st := r.snapshot()
		up := 0.0
		if st.Status == StatusOK {
			up = 1
		}
		out = append(out, module.Sample{Name: "gately_stun_up", Help: "Whether the STUN rule currently has a working public mapping (1) or not (0)", Type: module.MetricGauge,
			Labels: map[string]string{"rule": r.rule.Name}, Value: up})
	}
	for _, k := range keys {
		r := m.runners[k]
		st := r.snapshot()
		out = append(out, module.Sample{Name: "gately_stun_connections_total", Help: "Connections handled by the STUN rule's built-in forwarder", Type: module.MetricCounter,
			Labels: map[string]string{"rule": r.rule.Name}, Value: float64(st.Stats.TotalConns)})
	}
	return out
}
