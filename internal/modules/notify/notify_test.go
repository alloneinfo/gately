package notify

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/wangwanjie/gately/internal/events"
)

func newTestModule(cfg fileConfig) (*Module, *[]string) {
	m := New()
	m.cfg = cfg
	var mu sync.Mutex
	var sent []string
	m.sender = func(_ context.Context, ch Channel, msg Message) error {
		mu.Lock()
		sent = append(sent, ch.Key+":"+msg.Event.Type)
		mu.Unlock()
		return nil
	}
	return m, &sent
}

func TestRuleMatching(t *testing.T) {
	r := Rule{Enable: true, Events: []string{"ddns.*"}, MinLevel: events.LevelWarn}
	if r.matches(events.Event{Type: events.DDNSChanged, Level: events.LevelInfo}) {
		t.Fatal("级别不足不应匹配")
	}
	if !r.matches(events.Event{Type: events.DDNSFailed, Level: events.LevelError}) {
		t.Fatal("通配符应匹配 ddns.failed")
	}
	if r.matches(events.Event{Type: events.WAFBlocked, Level: events.LevelError}) {
		t.Fatal("不同前缀不应匹配")
	}
	all := Rule{Enable: true, Events: []string{"*"}}
	if !all.matches(events.Event{Type: "anything", Level: events.LevelInfo}) {
		t.Fatal("* 应匹配全部")
	}
}

func TestPlanSilenceAndDedupChannels(t *testing.T) {
	m, _ := newTestModule(fileConfig{
		Channels: []Channel{
			{Key: "a", Name: "A", Enable: true},
			{Key: "b", Name: "B", Enable: false},
		},
		Rules: []Rule{
			{Key: "r1", Name: "r1", Enable: true, Events: []string{"*"}, Channels: []string{"a", "b"}, Silence: 60},
			{Key: "r2", Name: "r2", Enable: true, Events: []string{"*"}, Channels: []string{"a"}},
		},
	})
	base := time.Now()
	m.now = func() time.Time { return base }
	e := events.Event{Type: events.DDNSChanged, Source: "x", Level: events.LevelInfo}
	got := m.plan(e)
	if len(got) != 1 || got[0].ch.Key != "a" {
		t.Fatalf("同一渠道应只发一次且跳过停用渠道: %+v", got)
	}
	// 静默期内 r1 被丢弃，r2 无静默仍发送
	m.now = func() time.Time { return base.Add(10 * time.Second) }
	got = m.plan(e)
	if len(got) != 1 || got[0].rule != "r2" {
		t.Fatalf("静默期内只应由 r2 投递: %+v", got)
	}
	m.now = func() time.Time { return base.Add(61 * time.Second) }
	if got = m.plan(e); len(got) != 1 || got[0].rule != "r1" {
		t.Fatalf("静默期过后 r1 应恢复: %+v", got)
	}
}

func TestHandleDelivers(t *testing.T) {
	m, sent := newTestModule(fileConfig{
		Channels: []Channel{{Key: "a", Name: "A", Enable: true}},
		Rules:    []Rule{{Key: "r", Name: "r", Enable: true, Events: []string{"waf.*"}, Channels: []string{"a"}}},
	})
	m.log = nil
	m.handleSync(events.Event{Type: events.WAFBlocked, Level: events.LevelWarn, Time: time.Now()})
	if len(*sent) != 1 || (*sent)[0] != "a:waf.blocked" {
		t.Fatalf("投递结果不符: %v", *sent)
	}
}

func TestChannelValidate(t *testing.T) {
	cases := []struct {
		name string
		ch   Channel
		ok   bool
	}{
		{"telegram 缺少 chat", Channel{Name: "t", Kind: KindTelegram, Secret: "x"}, false},
		{"wecom 地址无效", Channel{Name: "w", Kind: KindWeCom, URL: "ftp://x"}, false},
		{"wecom 正常", Channel{Name: "w", Kind: KindWeCom, URL: "https://qyapi.weixin.qq.com/x"}, true},
		{"smtp 默认端口", Channel{Name: "s", Kind: KindSMTP, Host: "smtp.x.com", Target: "a@b.c", User: "u"}, true},
		{"未知类型", Channel{Name: "u", Kind: "sms"}, false},
	}
	for _, c := range cases {
		err := c.ch.Validate()
		if (err == nil) != c.ok {
			t.Errorf("%s: err=%v", c.name, err)
		}
	}
	s := Channel{Name: "s", Kind: KindSMTP, Host: "h", Target: "a@b.c", User: "u"}
	_ = s.Validate()
	if s.Port != 587 || s.Security != SecurityStartTLS {
		t.Fatalf("SMTP 默认值不符: %+v", s)
	}
}
