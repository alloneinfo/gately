package app

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/wangwanjie/gately/internal/apitoken"
	"github.com/wangwanjie/gately/internal/auth"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/logbuf"
)

// roleGuard 限制只读账号：只允许需要读权限的接口，写操作与账号、令牌、备份等高危接口一律拒绝。
// 管理员与 API 令牌（自带权限范围，上下文中没有角色）不受影响。
func (a *App) roleGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth.RoleFromContext(r.Context()) == auth.RoleReadonly {
			need := apitoken.Required(r.Method, r.URL.Path)
			if need != apitoken.Any && (need == apitoken.Denied || !strings.HasSuffix(need, ":"+apitoken.LevelRead)) {
				httpx.Error(w, http.StatusForbidden, "只读账号不能执行此操作")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

type userRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *App) handleUserList(w http.ResponseWriter, r *http.Request) {
	httpx.OK(w, a.auth.Users())
}

func (a *App) handleUserCreate(w http.ResponseWriter, r *http.Request) {
	var req userRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	if err := a.auth.AddUser(req.Username, req.Password); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	a.sysLogs.Add(logbuf.LevelInfo, "auth", fmt.Sprintf("已创建只读账号 %s", req.Username))
	httpx.OK(w, a.auth.Users())
}

func (a *App) handleUserPassword(w http.ResponseWriter, r *http.Request) {
	var req userRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	name := r.PathValue("name")
	if err := a.auth.SetUserPassword(name, req.Password); err != nil {
		a.userError(w, err)
		return
	}
	a.sysLogs.Add(logbuf.LevelInfo, "auth", fmt.Sprintf("已重置账号 %s 的密码", name))
	httpx.OK(w, map[string]bool{"ok": true})
}

func (a *App) handleUserDelete(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := a.auth.DeleteUser(name); err != nil {
		a.userError(w, err)
		return
	}
	a.sysLogs.Add(logbuf.LevelWarn, "auth", fmt.Sprintf("已删除账号 %s", name))
	httpx.OK(w, a.auth.Users())
}

func (a *App) userError(w http.ResponseWriter, err error) {
	if errors.Is(err, auth.ErrUserNotFound) {
		httpx.Error(w, http.StatusNotFound, "%v", err)
		return
	}
	httpx.Error(w, http.StatusBadRequest, "%v", err)
}

// handleOnboarding 返回是否需要展示首次启动引导（全新安装且尚未完成）。
func (a *App) handleOnboarding(w http.ResponseWriter, r *http.Request) {
	httpx.OK(w, map[string]bool{"pending": a.auth.OnboardingPending()})
}

// handleOnboardingDismiss 标记引导已完成。
func (a *App) handleOnboardingDismiss(w http.ResponseWriter, r *http.Request) {
	if err := a.auth.DismissOnboarding(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	httpx.OK(w, map[string]bool{"pending": false})
}
