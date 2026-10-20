package app

import (
	"net/http"
	"runtime"
	"time"

	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/version"
)

// handleMetrics 以 Prometheus 文本格式导出全局指标：进程状态加上所有实现了 module.Metricer 的模块。
// Prometheus 抓取时在 bearer_token 中填写具有 metrics:read 权限的 API 令牌。
func (a *App) handleMetrics(w http.ResponseWriter, r *http.Request) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	samples := []module.Sample{
		{Name: "gately_build_info", Help: "Build information; the value is always 1", Type: module.MetricGauge,
			Labels: map[string]string{"version": version.Version, "commit": version.Commit, "goversion": runtime.Version()}, Value: 1},
		{Name: "gately_uptime_seconds", Help: "Seconds since the process started", Type: module.MetricGauge, Value: time.Since(a.startTime).Seconds()},
		{Name: "gately_goroutines", Help: "Number of goroutines", Type: module.MetricGauge, Value: float64(runtime.NumGoroutine())},
		{Name: "gately_memory_alloc_bytes", Help: "Bytes of allocated heap objects", Type: module.MetricGauge, Value: float64(mem.Alloc)},
	}
	for _, e := range a.modules {
		if mm, ok := e.mod.(module.Metricer); ok && e.enabled {
			samples = append(samples, mm.Metrics()...)
		}
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = module.WriteSamples(w, samples)
}
