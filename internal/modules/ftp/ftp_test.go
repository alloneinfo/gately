package ftp

import (
	"io"
	"log"
	"net"
	"net/textproto"
	"strconv"
	"testing"
	"time"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
)

func TestSettingsValidate(t *testing.T) {
	s := Settings{Enable: true}
	if err := s.Validate(); err == nil {
		t.Fatal("启用但无用户应报错")
	}
	s = Settings{Users: []User{{Name: "a", Password: "p", Root: "relative"}}}
	if err := s.Validate(); err == nil {
		t.Fatal("相对路径应报错")
	}
	s = Settings{PassiveStart: 3000, PassiveEnd: 2000}
	if err := s.Validate(); err == nil {
		t.Fatal("被动端口范围反向应报错")
	}
	s = Settings{}
	if err := s.Validate(); err != nil || s.Port != 2121 || s.IdleTimeout != 900 {
		t.Fatalf("默认值不正确: %v %+v", err, s)
	}
}

func TestFTPLogin(t *testing.T) {
	ln, _ := net.Listen("tcp4", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	st, _ := config.Open(t.TempDir())
	m := New(nil)
	if err := m.Init(module.Env{Store: st, Logs: logbuf.New(20, log.New(io.Discard, "", 0))}); err != nil {
		t.Fatal(err)
	}
	m.settings = Settings{Enable: true, ListenIP: "127.0.0.1", Port: port, Users: []User{
		{Name: "rw", Password: "p1", Root: t.TempDir()},
	}}
	if err := m.settings.Validate(); err != nil {
		t.Fatal(err)
	}
	m.restartLocked()
	if m.err != "" {
		t.Fatal(m.err)
	}
	defer m.Stop()

	login := func(user, pass string) int {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(port), 3*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		tp := textproto.NewConn(conn)
		if _, _, err := tp.ReadResponse(220); err != nil {
			t.Fatal(err)
		}
		_ = tp.PrintfLine("USER %s", user)
		if _, _, err := tp.ReadResponse(331); err != nil {
			t.Fatal(err)
		}
		_ = tp.PrintfLine("PASS %s", pass)
		code, _, _ := tp.ReadResponse(0)
		return code
	}
	if c := login("rw", "p1"); c != 230 {
		t.Fatalf("正确密码应返回 230，实际 %d", c)
	}
	if c := login("rw", "bad"); c != 530 {
		t.Fatalf("错误密码应返回 530，实际 %d", c)
	}
}
