package app

import "github.com/wangwanjie/gately/internal/fileusers"

// fileUsersExtra 在 /api/fileusers 的结果中附加各文件服务对共享用户的使用情况，已关闭的模块不列出。
func (a *App) fileUsersExtra(out map[string]any) {
	out["services"] = fileusers.Default.Services(a.moduleEnabled)
}

// moduleEnabled 判断模块 id 是否开启，未知模块视为关闭。
func (a *App) moduleEnabled(id string) bool {
	for _, e := range a.modules {
		if e.mod.ID() == id {
			return e.enabled
		}
	}
	return false
}
