package stun

import (
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

// TestPublicServers 向真实公共 STUN 服务器查询，仅在设置 GATELY_NET_TEST=1 时运行。
func TestPublicServers(t *testing.T) {
	if os.Getenv("GATELY_NET_TEST") != "1" {
		t.Skip("未设置 GATELY_NET_TEST=1")
	}
	servers := strings.Split(os.Getenv("GATELY_STUN_SERVERS"), ",")
	if servers[0] == "" {
		servers = []string{"stun.nextcloud.com:3478", "stun.hot-chilli.net:3478", "stun.sipnet.com:3478"}
	}
	for _, s := range servers {
		c, err := net.DialTimeout("tcp4", s, 5*time.Second)
		if err != nil {
			t.Logf("TCP %s: 连接失败 %v", s, err)
			continue
		}
		_ = c.SetDeadline(time.Now().Add(5 * time.Second))
		ap, err := BindingTCP(c)
		c.Close()
		t.Logf("TCP %s: %v %v", s, ap, err)

		pc, err := net.ListenPacket("udp4", ":0")
		if err != nil {
			t.Fatal(err)
		}
		ua, err := net.ResolveUDPAddr("udp4", s)
		if err != nil {
			pc.Close()
			continue
		}
		req, id := NewBindingRequest()
		_, _ = pc.WriteTo(req, ua)
		_ = pc.SetReadDeadline(time.Now().Add(3 * time.Second))
		buf := make([]byte, 1500)
		n, _, err := pc.ReadFrom(buf)
		if err == nil {
			ap, err = ParseBindingResponse(buf[:n], id)
		}
		pc.Close()
		t.Logf("UDP %s: %v %v", s, ap, err)
	}
}
