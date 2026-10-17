package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wangwanjie/gately/internal/events"
)

func testMsg() Message {
	return NewMessage(events.Event{Type: events.DDNSChanged, Title: "解析已更新", Message: "1.2.3.4", Level: events.LevelInfo})
}

func TestTelegramSend(t *testing.T) {
	var path string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &body)
		w.WriteHeader(200)
	}))
	defer srv.Close()
	ch := Channel{Kind: KindTelegram, URL: srv.URL, Secret: "TOKEN", Target: "42"}
	if err := send(context.Background(), ch, testMsg()); err != nil {
		t.Fatal(err)
	}
	if path != "/botTOKEN/sendMessage" || body["chat_id"] != "42" {
		t.Fatalf("请求不符: %s %v", path, body)
	}
	if !strings.Contains(body["text"].(string), "解析已更新") {
		t.Fatalf("正文缺少标题: %v", body["text"])
	}
}

func TestWeComBusinessError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":93000,"errmsg":"invalid webhook url"}`))
	}))
	defer srv.Close()
	err := send(context.Background(), Channel{Kind: KindWeCom, URL: srv.URL}, testMsg())
	if err == nil {
		t.Fatal("HTTP 200 但 errcode 非 0 应视为失败")
	}
}

func TestDingTalkSignAppended(t *testing.T) {
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"errcode":0}`))
	}))
	defer srv.Close()
	if err := send(context.Background(), Channel{Kind: KindDingTalk, URL: srv.URL + "?access_token=x", Secret: "s"}, testMsg()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "timestamp=") || !strings.Contains(query, "sign=") || !strings.Contains(query, "access_token=x") {
		t.Fatalf("加签参数缺失: %s", query)
	}
}

func TestWebhookChannelUsesVars(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		got = string(data)
	}))
	defer srv.Close()
	ch := Channel{Kind: KindWebhook}
	ch.Webhook.URL = srv.URL
	ch.Webhook.Method = "POST"
	ch.Webhook.Body = `{"t":"{type}","m":"{message}"}`
	if err := send(context.Background(), ch, testMsg()); err != nil {
		t.Fatal(err)
	}
	if got != `{"t":"ddns.changed","m":"1.2.3.4"}` {
		t.Fatalf("模板渲染结果不符: %s", got)
	}
}
