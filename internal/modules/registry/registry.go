// Package registry 集中创建全部功能模块，供主程序、文档导出与测试共用。
package registry

import (
	"github.com/wangwanjie/gately/internal/module"
	backupmod "github.com/wangwanjie/gately/internal/modules/backup"
	"github.com/wangwanjie/gately/internal/modules/cloudflared"
	"github.com/wangwanjie/gately/internal/modules/cron"
	"github.com/wangwanjie/gately/internal/modules/ddns"
	"github.com/wangwanjie/gately/internal/modules/docker"
	"github.com/wangwanjie/gately/internal/modules/frp"
	"github.com/wangwanjie/gately/internal/modules/ftp"
	"github.com/wangwanjie/gately/internal/modules/nettools"
	"github.com/wangwanjie/gately/internal/modules/notify"
	"github.com/wangwanjie/gately/internal/modules/portforward"
	"github.com/wangwanjie/gately/internal/modules/ssl"
	"github.com/wangwanjie/gately/internal/modules/storage"
	"github.com/wangwanjie/gately/internal/modules/stun"
	"github.com/wangwanjie/gately/internal/modules/tailscale"
	"github.com/wangwanjie/gately/internal/modules/waf"
	"github.com/wangwanjie/gately/internal/modules/webdav"
	"github.com/wangwanjie/gately/internal/modules/webservice"
	"github.com/wangwanjie/gately/internal/modules/webterminal"
	"github.com/wangwanjie/gately/internal/modules/wol"
)

// New 创建全部模块。Web 服务使用证书模块按 SNI 提供 HTTPS 证书，并按站点引用 WAF 实例，
// 因此一并返回证书模块供管理界面 HTTPS 复用。
func New() (mods []module.Module, certs *ssl.Module) {
	certs = ssl.New()
	firewall := waf.New()
	files := storage.New()
	mods = []module.Module{
		portforward.New(),
		ddns.New(),
		stun.New(),
		tailscale.New(),
		frp.New(),
		cloudflared.New(),
		certs,
		firewall,
		webservice.New(certs, firewall),
		wol.New(),
		webterminal.New(),
		cron.New(),
		docker.New(),
		webdav.New(certs),
		ftp.New(certs),
		files,
		notify.New(),
		nettools.New(),
		backupmod.New(files),
	}
	return mods, certs
}
