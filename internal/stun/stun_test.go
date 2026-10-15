package stun

import (
	"encoding/hex"
	"net"
	"net/netip"
	"strings"
	"testing"
	"time"
)

// RFC 5769 第 2.2 节 IPv4 响应测试向量（含 SOFTWARE、XOR-MAPPED-ADDRESS、MESSAGE-INTEGRITY、FINGERPRINT）
const rfc5769IPv4Response = "" +
	"0101003c2112a442b7e7a701bc34d686fa87dfae" +
	"8022000b7465737420766563746f7220" +
	"002000080001a147e112a643" +
	"000800142b91f599fd9e90c38c7489f92af9ba53f06be7d7" +
	"80280004c07d4c96"

func TestRFC5769Vector(t *testing.T) {
	b, err := hex.DecodeString(strings.ReplaceAll(rfc5769IPv4Response, " ", ""))
	if err != nil {
		t.Fatal(err)
	}
	var id TxID
	copy(id[:], b[8:20])
	ap, err := ParseBindingResponse(b, id)
	if err != nil {
		t.Fatal(err)
	}
	if ap.String() != "192.0.2.1:32853" {
		t.Fatalf("解析结果 %s，期望 192.0.2.1:32853", ap)
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	for _, s := range []string{"203.0.113.7:4500", "[2001:db8::42]:65000"} {
		addr := netip.MustParseAddrPort(s)
		_, id := NewBindingRequest()
		got, err := ParseBindingResponse(EncodeBindingResponse(id, addr), id)
		if err != nil || got != addr {
			t.Fatalf("%s: got %s %v", s, got, err)
		}
	}
	_, id := NewBindingRequest()
	var other TxID
	if _, err := ParseBindingResponse(EncodeBindingResponse(id, netip.MustParseAddrPort("1.2.3.4:5")), other); err == nil {
		t.Fatal("事务 ID 不匹配时应报错")
	}
}

func TestServerTCPAndUDP(t *testing.T) {
	srv, err := ListenServer("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	c, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))
	got, err := BindingTCP(c)
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != c.LocalAddr().String() {
		t.Fatalf("TCP 映射地址 %s，期望 %s", got, c.LocalAddr())
	}

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	req, id := NewBindingRequest()
	ua, _ := net.ResolveUDPAddr("udp", srv.Addr())
	if _, err := pc.WriteTo(req, ua); err != nil {
		t.Fatal(err)
	}
	_ = pc.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 1500)
	n, _, err := pc.ReadFrom(buf)
	if err != nil {
		t.Fatal(err)
	}
	got, err = ParseBindingResponse(buf[:n], id)
	if err != nil || got.String() != pc.LocalAddr().String() {
		t.Fatalf("UDP 映射地址 %s %v", got, err)
	}
}
