package events

import (
	"testing"
	"time"
)

func TestPublishFillsDefaultsAndUnsubscribe(t *testing.T) {
	b := NewBus()
	got := make(chan Event, 2)
	cancel := b.Subscribe(func(e Event) {
		got <- e
	})
	b.Publish(Event{Type: DDNSChanged})
	select {
	case e := <-got:
		if e.Level != LevelInfo || e.Time.IsZero() {
			t.Fatalf("默认值未补全: %+v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("未收到事件")
	}
	cancel()
	b.Publish(Event{Type: DDNSChanged})
	select {
	case <-got:
		t.Fatal("取消订阅后仍收到事件")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestLevelRank(t *testing.T) {
	if !(LevelRank(LevelError) > LevelRank(LevelWarn) && LevelRank(LevelWarn) > LevelRank(LevelInfo)) {
		t.Fatal("级别顺序错误")
	}
}
