package logbuf

import (
	"strings"
	"testing"
	"time"
)

func TestServerErrorLog(t *testing.T) {
	b := New(100, nil)
	lg := b.Logger("site").ServerErrorLog()
	for i := 0; i < 5; i++ {
		lg.Printf(`http: TLS handshake error from [240e::1]:5268%d: 没有匹配 "nas.example.com" 的证书`, i)
	}
	lg.Printf("http: TLS handshake error from 1.2.3.4:5: EOF")
	lg.Printf("http: panic serving 1.2.3.4:5: boom")
	es := b.Since(0)
	if len(es) != 3 {
		t.Fatalf("应记录 3 条（同类握手错误节流），实际 %d: %+v", len(es), es)
	}
	if es[0].Level != LevelWarn || es[0].Source != "site" || !strings.Contains(es[0].Message, "nas.example.com") {
		t.Errorf("握手错误记录不正确: %+v", es[0])
	}
	if es[1].Level != LevelDebug {
		t.Errorf("EOF 应为调试级别: %+v", es[1])
	}
	if es[2].Level != LevelWarn {
		t.Errorf("其他错误应为警告: %+v", es[2])
	}

	// 节流期结束后输出被合并的次数
	serverLogThrottle = 0
	defer func() { serverLogThrottle = time.Minute }()
	lg.Printf(`http: TLS handshake error from [240e::1]:1: 没有匹配 "nas.example.com" 的证书`)
	last := b.Since(0)[3]
	if !strings.Contains(last.Message, "另有 4 次") {
		t.Errorf("应提示被合并的次数: %s", last.Message)
	}
}
