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
	"github.com/wangwanjie/gately/internal/modules/portforward"
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

	a, err := app.New(app.Options{
		ConfigDir: *configDir,
		Listen:    *listen,
		Web:       web.FS(),
		Modules: []module.Module{
			portforward.New(),
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
