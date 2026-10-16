package cloudflared

import (
	"slices"
	"strings"
	"testing"
)

func TestTunnelArgs(t *testing.T) {
	tok := Tunnel{Name: "a", Mode: ModeToken, Token: "abc", Protocol: "http2", EdgeIPVersion: "4"}
	args, env := tok.Args()
	if !slices.Equal(args, []string{"tunnel", "--no-autoupdate", "--protocol", "http2", "--edge-ip-version", "4", "run"}) {
		t.Errorf("令牌隧道参数错误: %v", args)
	}
	if !slices.Equal(env, []string{"TUNNEL_TOKEN=abc"}) {
		t.Errorf("令牌应通过环境变量传递: %v", env)
	}
	for _, a := range args {
		if strings.Contains(a, "abc") {
			t.Error("令牌不应出现在命令行")
		}
	}

	quick := Tunnel{Name: "q", Mode: ModeQuick, Service: "http://localhost:8080", Protocol: "auto"}
	args, env = quick.Args()
	if !slices.Equal(args, []string{"tunnel", "--no-autoupdate", "--url", "http://localhost:8080"}) || env != nil {
		t.Errorf("临时隧道参数错误: %v %v", args, env)
	}
}

func TestTunnelValidate(t *testing.T) {
	bad := []Tunnel{
		{Mode: ModeToken, Token: "x"},
		{Name: "a", Mode: ModeToken},
		{Name: "a", Mode: ModeQuick, Service: "localhost:8080"},
		{Name: "a", Mode: ModeQuick, Service: "ftp://x"},
		{Name: "a", Mode: "other"},
		{Name: "a", Mode: ModeToken, Token: "x", Protocol: "udp"},
		{Name: "a", Mode: ModeToken, Token: "x", EdgeIPVersion: "5"},
	}
	for i, tn := range bad {
		if err := tn.Validate(); err == nil {
			t.Errorf("用例 %d 应被拒绝: %+v", i, tn)
		}
	}
	good := Tunnel{Name: " a ", Mode: ModeQuick, Service: "http://localhost:80", Token: "leftover"}
	if err := good.Validate(); err != nil || good.Name != "a" || good.Token != "" {
		t.Errorf("规范化失败: %v %+v", err, good)
	}
}

func TestRuntimeOnLine(t *testing.T) {
	rt := &runtime{}
	rt.onLine("INF |  https://random-words-here.trycloudflare.com  |")
	rt.onLine("INF Registered tunnel connection connIndex=0")
	rt.onLine("INF Registered tunnel connection connIndex=1")
	if rt.url != "https://random-words-here.trycloudflare.com" || rt.conns != 2 {
		t.Errorf("解析结果错误: %q %d", rt.url, rt.conns)
	}
	rt.reset()
	if rt.url != "" || rt.conns != 0 {
		t.Error("reset 未清空")
	}
}
