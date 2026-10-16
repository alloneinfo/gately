package wol

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/luckyimport"
	"github.com/wangwanjie/gately/internal/module"
)

func TestMagicPacket(t *testing.T) {
	pkt, err := MagicPacket("aa:bb:cc:dd:ee:ff")
	if err != nil {
		t.Fatal(err)
	}
	if len(pkt) != 102 || !bytes.Equal(pkt[:6], bytes.Repeat([]byte{0xFF}, 6)) || !bytes.Equal(pkt[96:], []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}) {
		t.Fatalf("魔术包格式错误: %x", pkt)
	}
	if _, err := MagicPacket("zz"); err == nil {
		t.Fatal("非法 MAC 应报错")
	}
}

func TestWake(t *testing.T) {
	var mu sync.Mutex
	var sent []string
	old := sendFunc
	sendFunc = func(addr string, pkt []byte) error {
		mu.Lock()
		sent = append(sent, addr)
		mu.Unlock()
		return nil
	}
	defer func() { sendFunc = old }()
	d := Device{Name: "pc", MACs: []string{"AA-BB-CC-DD-EE-FF"}, BroadcastIPs: []string{"192.168.1.255"}, Repeat: 2}
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
	if d.MACs[0] != "aa:bb:cc:dd:ee:ff" || d.Port != 9 {
		t.Fatalf("规范化错误: %+v", d)
	}
	n, err := Wake(d)
	if err != nil || n != 2 || sent[0] != "192.168.1.255:9" {
		t.Fatalf("发送错误: n=%d %v %v", n, sent, err)
	}
}

func TestImportLucky(t *testing.T) {
	raw, _ := json.Marshal([]map[string]any{
		{"DeviceName": "NAS", "MacList": []string{"00:11:32:aa:bb:cc"}, "Port": 9, "Repeat": 5},
		{"DeviceName": "bad", "MacList": []string{"x"}},
	})
	st, _ := config.Open(t.TempDir())
	m := New()
	_ = m.Init(module.Env{Store: st, Logs: logbuf.New(10, log.New(io.Discard, "", 0))})
	rep, err := m.ImportLucky(luckyimport.Document{"WOLDeviceList": raw}, luckyimport.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Found != 2 || rep.Imported != 1 || len(m.cfg.Devices) != 1 || m.cfg.Devices[0].Repeat != 5 {
		t.Fatalf("导入结果错误: %+v %+v", rep, m.cfg.Devices)
	}
	if path := os.Getenv("GATELY_LUCKY_WOL_SAMPLE"); path != "" {
		data, _ := os.ReadFile(path)
		doc, err := luckyimport.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		rep, err := m.ImportLucky(doc, luckyimport.Options{DryRun: true})
		if err != nil || rep.Imported != rep.Found {
			t.Fatalf("真实样本导入失败: %+v %v", rep, err)
		}
		t.Logf("真实样本 %d/%d", rep.Imported, rep.Found)
	}
}
