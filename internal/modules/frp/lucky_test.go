package frp

import (
	"encoding/json"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/luckyimport"
	"github.com/wangwanjie/gately/internal/module"
)

// luckyFRPSample 按 Lucky 2.27.2 前端提交的实例结构构造。
const luckyFRPSample = `{"List":[
  {"Key":"c1","Remark":"home","Enable":true,"Type":"client",
   "Params":{"ServerAddr":"frp.example.com","ServerPort":7001,"User":"me","AuthMethod":"token","Token":"t0k","Protocol":"tcp","TLSEnable":true,"AdminPort":0},
   "proxies":[
     {"name":"ssh","type":"tcp","disabled":false,"annotations":{},"metadatas":{},"localIP":"127.0.0.1","localPort":22,"remotePort":6022,"useEncryption":false,"natTraversal":{"disableAssistedAddrs":false}},
     {"name":"web","type":"http","localIP":"192.168.1.2","localPort":80,"customDomains":["a.example.com"],"useCompression":true},
     {"name":"off","type":"tcp","disabled":true,"localPort":3389,"remotePort":6389},
     {"name":"vnet","type":"tcp","plugin":"virtual_net","remotePort":7000}
   ],
   "visitors":[{"name":"v","type":"stcp"}]},
  {"Key":"c2","Remark":"second","Enable":true,"Type":"client","Params":{"ServerAddr":"b.example.com"}},
  {"Key":"s1","Remark":"srv","Enable":false,"Type":"server",
   "Params":{"BindAddr":"0.0.0.0","BindPort":7100,"Token":"st","VhostHTTPPort":8080,"VhostHTTPSPort":0,"SubDomainHost":"example.com","AllowPorts":"6000-6100, 7000","DashboardPort":7500,"DashboardUser":"admin"}}
]}`

func TestImportLuckyFRP(t *testing.T) {
	st, _ := config.Open(t.TempDir())
	m := New()
	if err := m.Init(module.Env{Store: st, Logs: logbuf.New(20, log.New(io.Discard, "", 0))}); err != nil {
		t.Fatal(err)
	}
	var doc luckyimport.Document
	if err := json.Unmarshal([]byte(luckyFRPSample), &doc); err != nil {
		t.Fatal(err)
	}
	rep, err := m.ImportLucky(doc, luckyimport.Options{})
	if err != nil || rep == nil {
		t.Fatalf("导入失败: %v %v", rep, err)
	}
	if rep.Found != 3 || rep.Imported != 2 {
		t.Fatalf("统计错误: %+v", rep)
	}
	c := m.cfg.Client
	if c.Enable || c.ServerAddr != "frp.example.com" || c.ServerPort != 7001 || c.Token != "t0k" || c.User != "me" || c.Protocol != "" {
		t.Fatalf("客户端映射错误: %+v", c)
	}
	if len(m.cfg.Proxies) != 3 {
		t.Fatalf("应导入 3 个代理（插件代理跳过），实际 %+v", m.cfg.Proxies)
	}
	if p := m.cfg.Proxies[1]; p.Type != TypeHTTP || p.CustomDomains[0] != "a.example.com" || p.LocalIP != "192.168.1.2" {
		t.Fatalf("http 代理映射错误: %+v", p)
	}
	if m.cfg.Proxies[2].Enable {
		t.Fatal("disabled 代理应保持停用")
	}
	s := m.cfg.Server
	if s.Enable || s.BindPort != 7100 || s.Token != "st" || s.VhostHTTPPort != 8080 || s.SubdomainHost != "example.com" || s.AllowPorts != "6000-6100,7000" || s.DashboardPort != 0 {
		t.Fatalf("服务端映射错误: %+v", s)
	}
	joined := strings.Join(rep.Warnings, "\n")
	for _, want := range []string{"second: 跳过", "virtual_net", "useCompression（1 个代理）", "访问者", "控制台未迁移"} {
		if !strings.Contains(joined, want) {
			t.Errorf("缺少提示 %q:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "annotations") || strings.Contains(joined, "natTraversal") || strings.Contains(joined, "useEncryption") {
		t.Errorf("空值字段不应提示:\n%s", joined)
	}

	// 再次导入：同名代理跳过，不产生重复
	if _, err := m.ImportLucky(doc, luckyimport.Options{}); err != nil {
		t.Fatal(err)
	}
	if len(m.cfg.Proxies) != 3 {
		t.Fatalf("重复导入不应产生重复代理: %d", len(m.cfg.Proxies))
	}
	// 其他模块的 List 配置不应被认领
	other := luckyimport.Document{"List": json.RawMessage(`[{"Type":"tunnel"}]`)}
	if rep, _ := m.ImportLucky(other, luckyimport.Options{}); rep != nil {
		t.Fatal("cloudflared 配置不应被 frp 认领")
	}
}
