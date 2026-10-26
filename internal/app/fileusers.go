package app

import "github.com/wangwanjie/gately/internal/fileusers"

// fileUsersExtra 在 /api/fileusers 的结果中附加各文件服务对共享用户的使用情况，已关闭的模块不列出。
func (a *App) fileUsersExtra(out map[string]any) {
	enabled := map[string]bool{}
	for _, e := range a.modules {
		enabled[e.mod.ID()] = e.enabled
	}
	out["services"] = fileusers.Default.Services(func(id string) bool { return enabled[id] })
}
