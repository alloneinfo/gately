package app

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/wangwanjie/gately/internal/audit"
	"github.com/wangwanjie/gately/internal/events"
	"github.com/wangwanjie/gately/internal/httpx"
)

// handleAudit 按条件查询操作审计日志（时间倒序）。
func (a *App) handleAudit(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()
	limit, _ := strconv.Atoi(qs.Get("limit"))
	offset, _ := strconv.Atoi(qs.Get("offset"))
	httpx.OK(w, a.audit.Search(audit.Query{
		Category: qs.Get("category"),
		Module:   qs.Get("module"),
		Actor:    qs.Get("actor"),
		Text:     qs.Get("q"),
		Limit:    limit,
		Offset:   offset,
	}))
}

// publishAudit 把变更与敏感访问转成通知事件，是否推送由通知中心的规则决定。
func publishAudit(e audit.Entry) {
	var typ, level, title string
	switch e.Category {
	case audit.CategoryChange:
		typ, level, title = events.AuditChange, events.LevelInfo, "管理后台配置变更"
	case audit.CategoryAccess:
		typ, level, title = events.AuditAccess, events.LevelWarn, "管理后台访问敏感内容"
	default:
		return
	}
	events.Publish(events.Event{
		Type: typ, Source: "管理后台", Level: level, Title: title,
		Message: fmt.Sprintf("%s 从 %s 执行 %s %s（状态 %d）", e.Actor, e.IP, e.Method, e.Path, e.Status),
		Fields:  map[string]string{"actor": e.Actor, "ip": e.IP, "method": e.Method, "path": e.Path},
	})
}
