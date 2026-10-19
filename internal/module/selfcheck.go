package module

import "context"

// 自检结果状态
const (
	CheckOK   = "ok"
	CheckInfo = "info"
	CheckWarn = "warn"
	CheckFail = "fail"
)

// CheckResult 是一项自检结果。ID 与 Params 由前端翻译成可读说明，后端不拼接面向人的文案。
type CheckResult struct {
	ID     string            `json:"id"`
	Status string            `json:"status"`
	Params map[string]string `json:"params,omitempty"`
}

// CheckEnv 是自检时由框架提供的公共信息。
type CheckEnv struct {
	// PublicIPv4 / PublicIPv6 为查询到的出口公网地址，未知时为空
	PublicIPv4 string
	PublicIPv6 string
}

// SelfChecker 由能够检查自身配置与外部环境是否一致的模块实现。
type SelfChecker interface {
	SelfCheck(ctx context.Context, env CheckEnv) []CheckResult
}
