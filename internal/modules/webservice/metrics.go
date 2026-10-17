package webservice

import (
	"fmt"
	"net/http"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wangwanjie/gately/internal/geoip"
	"github.com/wangwanjie/gately/internal/httpx"
)

const (
	// metricMinutes 是保留的分钟桶数量（24 小时）
	metricMinutes = 1440
	// topKeysCap 限制排行榜统计的不同键数量，防止被大量来源撑大内存
	topKeysCap = 5000
)

type minuteBucket struct {
	minute                 int64
	req, bytes             int64
	s2xx, s3xx, s4xx, s5xx int64
}

// siteMetrics 记录站点最近 24 小时的分钟级统计与来源排行。
type siteMetrics struct {
	mu        sync.Mutex
	ring      [metricMinutes]minuteBucket
	ips       map[string]int64
	countries map[string]int64
	bytesOut  int64
	status    [6]int64
}

func newSiteMetrics() *siteMetrics {
	return &siteMetrics{ips: map[string]int64{}, countries: map[string]int64{}}
}

// record 登记一次请求。
func (m *siteMetrics) record(ip netip.Addr, status int, bytes int64, now time.Time) {
	min := now.Unix() / 60
	idx := int(min % metricMinutes)
	class := status / 100
	var country string
	if geoip.Default.Info().Loaded && !geoip.IsLocal(ip) {
		country = geoip.Default.Country(ip)
	}
	key := limitKeyAddr(ip)
	m.mu.Lock()
	defer m.mu.Unlock()
	b := &m.ring[idx]
	if b.minute != min {
		*b = minuteBucket{minute: min}
	}
	b.req++
	b.bytes += bytes
	m.bytesOut += bytes
	switch class {
	case 2:
		b.s2xx++
	case 3:
		b.s3xx++
	case 4:
		b.s4xx++
	case 5:
		b.s5xx++
	}
	if class >= 1 && class <= 5 {
		m.status[class]++
	}
	if _, ok := m.ips[key]; ok || len(m.ips) < topKeysCap {
		m.ips[key]++
	}
	if country != "" {
		m.countries[country]++
	}
}

// Point 是一个时间点的统计。
type Point struct {
	Time     int64 `json:"t"`
	Requests int64 `json:"requests"`
	Bytes    int64 `json:"bytes"`
	S4xx     int64 `json:"s4xx"`
	S5xx     int64 `json:"s5xx"`
}

// TopItem 是排行榜的一项。
type TopItem struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

// SiteStatsView 是站点统计接口的输出。
type SiteStatsView struct {
	Series       []Point          `json:"series"`
	Status       map[string]int64 `json:"status"`
	BytesOut     int64            `json:"bytesOut"`
	TopIPs       []TopItem        `json:"topIps"`
	TopCountries []TopItem        `json:"topCountries"`
	Upstreams    []UpstreamStatus `json:"upstreams,omitempty"`
}

func top(m map[string]int64, n int) []TopItem {
	out := make([]TopItem, 0, len(m))
	for k, v := range m {
		out = append(out, TopItem{k, v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Key < out[j].Key
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// view 返回最近 minutes 分钟的序列（每分钟一点，缺失的分钟为 0）与累计排行。
func (m *siteMetrics) view(minutes int, now time.Time) SiteStatsView {
	if minutes < 1 || minutes > metricMinutes {
		minutes = 60
	}
	cur := now.Unix() / 60
	m.mu.Lock()
	defer m.mu.Unlock()
	series := make([]Point, 0, minutes)
	for i := minutes - 1; i >= 0; i-- {
		min := cur - int64(i)
		b := m.ring[int(min%metricMinutes)]
		p := Point{Time: min * 60}
		if b.minute == min {
			p.Requests, p.Bytes, p.S4xx, p.S5xx = b.req, b.bytes, b.s4xx, b.s5xx
		}
		series = append(series, p)
	}
	return SiteStatsView{
		Series:       series,
		Status:       map[string]int64{"2xx": m.status[2], "3xx": m.status[3], "4xx": m.status[4], "5xx": m.status[5]},
		BytesOut:     m.bytesOut,
		TopIPs:       top(m.ips, 10),
		TopCountries: top(m.countries, 10),
	}
}

// totals 返回累计请求数、字节与各状态类别，供 Prometheus 导出。
func (m *siteMetrics) totals() (bytes int64, status [6]int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.bytesOut, m.status
}

// findSite 在运行中的监听器里查找站点处理器。
func (m *Module) findSiteLocked(listener, site string) *siteHandler {
	r, ok := m.runners[listener]
	if !ok {
		return nil
	}
	if site == "default" {
		return r.deflt
	}
	for _, s := range r.sites {
		if s.site.Key == site {
			return s
		}
	}
	return nil
}

func (m *Module) handleStats(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	minutes, _ := strconv.Atoi(q.Get("minutes"))
	m.mu.Lock()
	h := m.findSiteLocked(q.Get("listener"), q.Get("site"))
	m.mu.Unlock()
	if h == nil {
		httpx.Error(w, http.StatusNotFound, "站点不存在或所属监听未运行")
		return
	}
	v := h.metrics.view(minutes, time.Now())
	v.Upstreams = h.upstreamStatus()
	httpx.OK(w, v)
}

func escapeLabel(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(s)
}

// handleMetrics 以 Prometheus 文本格式导出各站点指标。
func (m *Module) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	var b strings.Builder
	b.WriteString("# HELP gately_site_requests_total 站点累计请求数\n# TYPE gately_site_requests_total counter\n")
	var rows []struct {
		labels string
		h      *siteHandler
	}
	m.mu.Lock()
	for lk, run := range m.runners {
		all := append([]*siteHandler{}, run.sites...)
		if run.deflt != nil {
			all = append(all, run.deflt)
		}
		for _, s := range all {
			rows = append(rows, struct {
				labels string
				h      *siteHandler
			}{fmt.Sprintf(`listener="%s",site="%s"`, escapeLabel(run.cfg.Name+"#"+lk), escapeLabel(siteLabel(s.site))), s})
		}
	}
	m.mu.Unlock()
	sort.Slice(rows, func(i, j int) bool { return rows[i].labels < rows[j].labels })
	for _, row := range rows {
		fmt.Fprintf(&b, "gately_site_requests_total{%s} %d\n", row.labels, row.h.stats.Requests.Load())
	}
	b.WriteString("# HELP gately_site_errors_total 站点后端错误数\n# TYPE gately_site_errors_total counter\n")
	for _, row := range rows {
		fmt.Fprintf(&b, "gately_site_errors_total{%s} %d\n", row.labels, row.h.stats.Errors.Load())
	}
	b.WriteString("# HELP gately_site_active_requests 站点进行中的请求数\n# TYPE gately_site_active_requests gauge\n")
	for _, row := range rows {
		fmt.Fprintf(&b, "gately_site_active_requests{%s} %d\n", row.labels, row.h.stats.Active.Load())
	}
	b.WriteString("# HELP gately_site_response_bytes_total 站点累计响应字节数\n# TYPE gately_site_response_bytes_total counter\n")
	for _, row := range rows {
		bytes, _ := row.h.metrics.totals()
		fmt.Fprintf(&b, "gately_site_response_bytes_total{%s} %d\n", row.labels, bytes)
	}
	b.WriteString("# HELP gately_site_responses_total 按状态类别统计的响应数\n# TYPE gately_site_responses_total counter\n")
	for _, row := range rows {
		_, st := row.h.metrics.totals()
		for c := 2; c <= 5; c++ {
			fmt.Fprintf(&b, "gately_site_responses_total{%s,class=\"%dxx\"} %d\n", row.labels, c, st[c])
		}
	}
	b.WriteString("# HELP gately_upstream_up 后端是否健康（1 健康，0 不可用）\n# TYPE gately_upstream_up gauge\n")
	for _, row := range rows {
		for _, u := range row.h.upstreamStatus() {
			v := 0
			if u.Healthy {
				v = 1
			}
			fmt.Fprintf(&b, "gately_upstream_up{%s,upstream=\"%s\"} %d\n", row.labels, escapeLabel(u.URL), v)
		}
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = w.Write([]byte(b.String()))
}
