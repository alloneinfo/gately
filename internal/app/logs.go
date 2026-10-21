package app

import (
	"path/filepath"

	"github.com/wangwanjie/gately/internal/logbuf"
)

// logsDir 是配置目录下保存运行日志的子目录：logs/<模块 ID>.log，系统日志为 logs/gately.log。
// 备份只收集配置目录顶层的 *.json，不会包含日志。
const logsDir = "logs"

// persistLogs 按设置开启或关闭一个日志缓冲的落盘；source 为模块 ID，系统日志使用 settingsName。
func (a *App) persistLogs(buf *logbuf.Buffer, source string, s Settings) {
	if s.DisableLogPersist {
		buf.ClosePersist()
		return
	}
	path := filepath.Join(a.opts.ConfigDir, logsDir, source+".log")
	if err := buf.Persist(path, int64(s.LogFileMB)<<20); err != nil {
		// 落盘失败不影响运行，日志仍保留在内存
		a.logger.Printf("开启日志落盘失败（%s）: %v", path, err)
	}
}

// applyLogPersist 把日志落盘设置应用到系统日志与全部模块日志。
func (a *App) applyLogPersist(s Settings) {
	a.persistLogs(a.sysLogs, settingsName, s)
	for _, e := range a.modules {
		a.persistLogs(e.logs, e.mod.ID(), s)
	}
}

// closeLogs 把尚未写盘的日志写入文件并停止落盘，在退出或重启前调用。
func (a *App) closeLogs() {
	a.sysLogs.ClosePersist()
	for _, e := range a.modules {
		e.logs.ClosePersist()
	}
}
