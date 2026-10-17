package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/wangwanjie/gately/internal/events"
)

// dockerEvent 是 Engine 事件流中的一条消息（只取用到的字段）。
type dockerEvent struct {
	Type   string `json:"Type"`
	Action string `json:"Action"`
	Actor  struct {
		ID         string            `json:"ID"`
		Attributes map[string]string `json:"Attributes"`
	} `json:"Actor"`
}

// watchEvents 订阅容器的 die / oom 事件，连接中断时返回错误。
func (c *Client) watchEvents(ctx context.Context, handle func(dockerEvent)) error {
	filters, _ := json.Marshal(map[string][]string{"type": {"container"}, "event": {"die", "oom"}})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/events?filters="+url.QueryEscape(string(filters)), nil)
	if err != nil {
		return err
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &APIError{Status: resp.StatusCode, Message: "无法订阅事件"}
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		var ev dockerEvent
		if json.Unmarshal(sc.Bytes(), &ev) == nil {
			handle(ev)
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return fmt.Errorf("事件流已关闭")
}

// publishContainerEvent 把容器异常退出转换为通知事件。退出码为 0 的正常停止不通知。
func publishContainerEvent(ev dockerEvent) {
	name := ev.Actor.Attributes["name"]
	if name == "" {
		name = ev.Actor.ID
		if len(name) > 12 {
			name = name[:12]
		}
	}
	image := ev.Actor.Attributes["image"]
	switch ev.Action {
	case "oom":
		events.Publish(events.Event{Type: events.DockerExited, Source: name, Level: events.LevelError,
			Title: "容器内存不足被终止：" + name, Message: "容器 " + name + "（" + image + "）触发 OOM。",
			Fields: map[string]string{"container": name, "image": image}})
	case "die":
		code := ev.Actor.Attributes["exitCode"]
		if code == "" || code == "0" {
			return
		}
		events.Publish(events.Event{Type: events.DockerExited, Source: name, Level: events.LevelWarn,
			Title: "容器异常退出：" + name, Message: "容器 " + name + "（" + image + "）退出，退出码 " + code + "。",
			Fields: map[string]string{"container": name, "image": image, "exitCode": code}})
	}
}

// runEventWatch 持续订阅容器事件。Docker 不可用时静默重试，仅在状态变化时记录一次日志。
func (m *Module) runEventWatch(ctx context.Context) {
	log := m.logs.Logger("Docker")
	wasUp := true
	for ctx.Err() == nil {
		c, err := m.client()
		if err == nil {
			err = c.watchEvents(ctx, publishContainerEvent)
		}
		if ctx.Err() != nil {
			return
		}
		if wasUp {
			log.Debugf("容器事件订阅中断，将自动重试: %v", err)
			wasUp = false
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(30 * time.Second):
		}
	}
}
