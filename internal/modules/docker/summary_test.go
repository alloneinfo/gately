package docker

import (
	"net/http"
	"testing"
)

func TestSummaryCountsContainers(t *testing.T) {
	_, host := newCtrFake(t, map[string]http.HandlerFunc{
		"GET /containers/json": ctrJSON([]map[string]any{
			{"Id": "aaaaaaaaaaaa1", "Names": []string{"/a"}, "State": "running", "Status": "Up 2 days"},
			{"Id": "bbbbbbbbbbbb2", "Names": []string{"/b"}, "State": "running", "Status": "Up 1 hour (unhealthy)"},
			{"Id": "cccccccccccc3", "Names": []string{"/c"}, "State": "exited", "Status": "Exited (0) 3 days ago"},
			{"Id": "dddddddddddd4", "Names": []string{"/d"}, "State": "exited", "Status": "Exited (137) 1 hour ago"},
		}),
	})
	m := newModule(t, host)
	m.updates["aaaaaaaaaaaa1"] = UpdateInfo{HasUpdate: true}
	got := map[string]string{}
	for _, it := range m.Summary() {
		got[it.Label] = it.Value
	}
	if got["运行中"] != "1 / 4" || got["已停止"] != "1" || got["异常"] != "2" || got["有更新"] != "1" {
		t.Errorf("摘要错误: %v", got)
	}
}

func TestSummaryDisconnected(t *testing.T) {
	m := newModule(t, "tcp://127.0.0.1:1")
	items := m.Summary()
	if len(items) != 1 || items[0].Value != "未连接" || items[0].Level != "info" {
		t.Errorf("未连接摘要错误: %v", items)
	}
}
