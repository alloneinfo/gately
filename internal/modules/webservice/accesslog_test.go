package webservice

import (
	"io"
	"log"
	"testing"

	"github.com/wangwanjie/gately/internal/logbuf"
)

// TestAccessLogFields 验证访问日志与访问控制拒绝日志带上结构化字段，日志页可按站点、域名、状态码与 IP 过滤。
func TestAccessLogFields(t *testing.T) {
	logs := logbuf.New(50, log.New(io.Discard, "", 0))
	l := Listener{
		Sites: []Site{
			{Enable: true, Name: "博客", Domains: []string{"blog.test"}, Type: TypeText, Text: "gone", StatusCode: 404, AccessLog: true},
			{Enable: true, Name: "内网", Domains: []string{"lan.test"}, Type: TypeText, Text: "x", AccessLog: true, IPMode: IPModeWhitelist, IPList: []string{"10.0.0.0/8"}},
		},
	}
	addr, _ := startRunnerLogs(t, l, nil, nil, logs)
	c := noRedirect()
	get(t, c, addr, "blog.test:8080", "/post?id=1", nil)
	get(t, c, addr, "lan.test", "/", nil)

	f := &logbuf.Filter{Status: []logbuf.StatusRange{{Min: 404, Max: 404}}}
	got := logs.Query(0, 0, 0, f)
	if len(got) != 1 {
		t.Fatalf("按状态码过滤应得到 1 条访问日志，得到 %+v", got)
	}
	want := logbuf.Fields{"ip": "127.0.0.1", "host": "blog.test", "site": "博客", "method": "GET", "path": "/post", "status": "404"}
	for k, v := range want {
		if got[0].Fields[k] != v {
			t.Errorf("字段 %s = %q，期望 %q（%+v）", k, got[0].Fields[k], v, got[0].Fields)
		}
	}
	denied := logs.Query(0, 0, 0, &logbuf.Filter{Host: "内网", MinLevel: logbuf.LevelWarn})
	if len(denied) != 1 || denied[0].Fields["ip"] != "127.0.0.1" {
		t.Fatalf("访问控制拒绝日志应带站点与 IP 字段: %+v", denied)
	}
}
