// Gately 是开源的网络网关工具集，适用于家庭网络、NAS 与各类服务器：端口转发、DDNS、STUN 穿透、反向代理与证书管理。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/wangwanjie/gately/internal/app"
	"github.com/wangwanjie/gately/internal/ctl"
	"github.com/wangwanjie/gately/internal/modules/registry"
	"github.com/wangwanjie/gately/internal/version"
	"github.com/wangwanjie/gately/web"
)

func main() {
	// 子命令：ctl 调用运行中的实例，mcp 提供 MCP 服务；其余参数走服务启动流程
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "ctl":
			os.Exit(ctl.RunCtl(os.Args[2:], os.Stdout, os.Stderr))
		case "mcp":
			os.Exit(ctl.RunMCP(os.Args[2:], os.Stdin, os.Stdout, os.Stderr))
		}
	}
	configDir := flag.String("c", defaultConfigDir(), "configuration directory / 配置目录")
	listen := flag.String("listen", "", "admin UI listen address, overrides the config file, e.g. :16680 / 管理界面监听地址，覆盖配置文件")
	showVersion := flag.Bool("version", false, "print version info and exit / 输出版本信息")
	apiDocs := flag.String("api-docs", "", "export API docs (openapi.json and API.md) to the given directory and exit / 导出 API 文档到指定目录后退出")
	flag.Parse()

	if *apiDocs != "" {
		if err := exportAPIDocs(*apiDocs); err != nil {
			log.Fatalf("导出 API 文档失败: %v", err)
		}
		return
	}

	if *showVersion {
		info, _ := json.Marshal(map[string]string{
			"version":   version.Version,
			"commit":    version.Commit,
			"buildDate": version.BuildDate,
			"go":        runtime.Version(),
		})
		fmt.Println(string(info))
		return
	}

	// Web 服务使用证书模块按 SNI 提供 HTTPS 证书，并按站点引用 WAF 实例
	mods, certs := registry.New()
	a, err := app.New(app.Options{
		ConfigDir: *configDir,
		Listen:    *listen,
		Web:       web.FS(),
		Certs:     certs,
		Modules:   mods,
	})
	if err != nil {
		log.Fatalf("启动失败: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := a.Run(ctx); err != nil {
		log.Fatalf("运行失败: %v", err)
	}
}

// defaultConfigDir 优先使用环境变量 GATELY_CONFIG_DIR，否则为当前目录下的 data。
func defaultConfigDir() string {
	if d := os.Getenv("GATELY_CONFIG_DIR"); d != "" {
		return d
	}
	return "data"
}

// exportAPIDocs 在临时配置目录中构建应用（不监听端口），把 API 文档写到 dir。
func exportAPIDocs(dir string) error {
	tmp, err := os.MkdirTemp("", "gately-apidocs-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	mods, certs := registry.New()
	a, err := app.New(app.Options{ConfigDir: tmp, Certs: certs, Modules: mods})
	if err != nil {
		return err
	}
	spec, md, err := a.APIDocs()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "openapi.json"), spec, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "API.md"), md, 0o644)
}
