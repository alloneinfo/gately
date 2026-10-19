package portforward

import (
	"sort"

	"github.com/wangwanjie/gately/internal/module"
)

// Metrics 实现 module.Metricer：每条运行中规则的连接数与流量。
func (m *Module) Metrics() []module.Sample {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := make([]string, 0, len(m.runners))
	for k := range m.runners {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []module.Sample
	add := func(name, help, typ string, labels map[string]string, v int64) {
		out = append(out, module.Sample{Name: name, Help: help, Type: typ, Labels: labels, Value: float64(v)})
	}
	for _, k := range keys {
		r := m.runners[k]
		l := map[string]string{"rule": r.rule.Name}
		add("gately_portforward_active_connections", "Connections currently open on the port forwarding rule", module.MetricGauge, l, r.stats.ActiveConns.Load())
	}
	for _, k := range keys {
		r := m.runners[k]
		add("gately_portforward_connections_total", "Connections accepted by the port forwarding rule", module.MetricCounter, map[string]string{"rule": r.rule.Name}, r.stats.TotalConns.Load())
	}
	for _, k := range keys {
		r := m.runners[k]
		add("gately_portforward_bytes_total", "Bytes relayed by the port forwarding rule", module.MetricCounter, map[string]string{"rule": r.rule.Name, "direction": "in"}, r.stats.BytesIn.Load())
		add("gately_portforward_bytes_total", "Bytes relayed by the port forwarding rule", module.MetricCounter, map[string]string{"rule": r.rule.Name, "direction": "out"}, r.stats.BytesOut.Load())
	}
	return out
}
