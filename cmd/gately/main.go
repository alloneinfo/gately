// Gately 是开源的家庭网络网关工具集：端口转发、DDNS、STUN 穿透、反向代理与证书管理。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/wangwanjie/gately/internal/app"
	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/modules/cron"
	"github.com/wangwanjie/gately/internal/modules/ddns"
	"github.com/wangwanjie/gately/internal/modules/docker"
	"github.com/wangwanjie/gately/internal/modules/portforward"
	"github.com/wangwanjie/gately/internal/modules/ssl"
	"github.com/wangwanjie/gately/internal/modules/stun"
	"github.com/wangwanjie/gately/internal/modules/waf"
	"github.com/wangwanjie/gately/internal/modules/webservice"
	"github.com/wangwanjie/gately/internal/modules/webterminal"
	"github.com/wangwanjie/gately/internal/modules/wol"
	"github.com/wangwanjie/gately/internal/version"
	"github.com/wangwanjie/gately/web"
)

func main() {
	configDir := flag.String("c", defaultConfigDir(), "配置目录")
	listen := flag.String("listen", "", "管理界面监听地址，覆盖配置文件，例如 :16680")
	showVersion := flag.Bool("version", false, "输出版本信息")
	flag.Parse()

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
	certs := ssl.New()
	firewall := waf.New()
	a, err := app.New(app.Options{
		ConfigDir: *configDir,
		Listen:    *listen,
		Web:       web.FS(),
		Modules: []module.Module{
			portforward.New(),
			ddns.New(),
			stun.New(),
			certs,
			firewall,
			webservice.New(certs, firewall),
			wol.New(),
			webterminal.New(),
			cron.New(),
			docker.New(),
		},
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
