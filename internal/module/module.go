// Package module 定义功能模块的统一接口。
//
// 每个模块拥有独立的配置文件（config.Store 中以 ID 命名）、独立的日志缓冲，
// 以及挂载在 /api/<ID>/ 下的管理接口。
package module

import (
	"context"
	"net/http"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/logbuf"
)

// Env 是框架注入给模块的运行环境。
type Env struct {
	Store *config.Store
	Logs  *logbuf.Buffer
}

// Router 用于注册模块接口。pattern 采用 Go 1.22 ServeMux 语法，
// 但路径部分相对模块前缀，例如 "GET /rules" 实际挂载为 "GET /api/<ID>/rules"。
type Router interface {
	HandleFunc(pattern string, h http.HandlerFunc)
}

// Module 是所有功能模块需要实现的接口。
type Module interface {
	// ID 是模块唯一标识，同时作为配置文件名和 API 前缀
	ID() string
	// Title 是界面显示名称
	Title() string
	// Init 加载配置，不启动任何网络服务
	Init(env Env) error
	// Start 按配置启动服务，ctx 在程序退出时取消
	Start(ctx context.Context) error
	// Stop 停止全部服务并释放端口
	Stop() error
	// Routes 注册管理接口
	Routes(r Router)
}
