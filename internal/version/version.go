// Package version 保存构建时注入的版本信息。
package version

// 以下变量通过 -ldflags "-X github.com/wangwanjie/gately/internal/version.Version=..." 注入。
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)
