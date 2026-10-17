package docker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wangwanjie/gately/internal/events"
)

func TestWatchEventsPublishesAbnormalExits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/events") || !strings.Contains(r.URL.Query().Get("filters"), "die") {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(200)
		for _, line := range []string{
			`{"Type":"container","Action":"die","Actor":{"ID":"abc","Attributes":{"name":"web","image":"nginx","exitCode":"0"}}}`,
			`{"Type":"container","Action":"die","Actor":{"ID":"abc","Attributes":{"name":"api","image":"app:1","exitCode":"137"}}}`,
			`{"Type":"container","Action":"oom","Actor":{"ID":"def0123456789","Attributes":{"image":"db"}}}`,
		} {
			_, _ = w.Write([]byte(line + "\n"))
		}
	}))
	defer srv.Close()
	got := make(chan events.Event, 4)
	defer events.Subscribe(func(e events.Event) {
		if e.Type == events.DockerExited {
			got <- e
		}
	})()
	c, err := NewClient("tcp://" + strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	// 服务端发完即关闭连接，watchEvents 返回"事件流已关闭"
	if err := c.watchEvents(context.Background(), publishContainerEvent); err == nil {
		t.Fatal("连接关闭应返回错误以触发重连")
	}
	var titles []string
	for i := 0; i < 2; i++ {
		select {
		case e := <-got:
			titles = append(titles, e.Title)
		case <-time.After(2 * time.Second):
			t.Fatalf("应收到 2 个事件，实际 %v", titles)
		}
	}
	joined := strings.Join(titles, "|")
	if !strings.Contains(joined, "容器异常退出：api") || !strings.Contains(joined, "内存不足") || !strings.Contains(joined, "def012345678") {
		t.Fatalf("事件内容不符: %v", titles)
	}
	select {
	case e := <-got:
		t.Fatalf("退出码为 0 的正常停止不应通知: %s", e.Title)
	case <-time.After(200 * time.Millisecond):
	}
}
