package webdav

import (
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
)

func TestWebDAVUsers(t *testing.T) {
	rootA, rootB := t.TempDir(), t.TempDir()
	_ = os.WriteFile(filepath.Join(rootB, "readme.txt"), []byte("hello"), 0o644)
	ln, _ := net.Listen("tcp4", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	st, _ := config.Open(t.TempDir())
	m := New(nil)
	if err := m.Init(module.Env{Store: st, Logs: logbuf.New(20, log.New(io.Discard, "", 0))}); err != nil {
		t.Fatal(err)
	}
	m.settings = Settings{Enable: true, ListenIP: "127.0.0.1", Port: port, Users: []User{
		{Name: "rw", Password: "p1", Root: rootA},
		{Name: "ro", Password: "p2", Root: rootB, ReadOnly: true},
	}}
	if err := m.settings.Validate(); err != nil {
		t.Fatal(err)
	}
	m.restartLocked()
	defer m.Stop()
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	do := func(method, path, user, pass, body string) int {
		req, _ := http.NewRequest(method, base+path, strings.NewReader(body))
		if user != "" {
			req.SetBasicAuth(user, pass)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if c := do("PUT", "/a.txt", "rw", "bad", "x"); c != 401 {
		t.Fatalf("错误密码应 401: %d", c)
	}
	if c := do("PUT", "/a.txt", "rw", "p1", "data"); c != 201 {
		t.Fatalf("读写用户上传失败: %d", c)
	}
	if b, _ := os.ReadFile(filepath.Join(rootA, "a.txt")); string(b) != "data" {
		t.Fatal("文件应写入用户目录")
	}
	if c := do("GET", "/readme.txt", "ro", "p2", ""); c != 200 {
		t.Fatalf("只读用户应能读取: %d", c)
	}
	if c := do("PUT", "/new.txt", "ro", "p2", "x"); c == 201 {
		t.Fatal("只读用户不应能写入")
	}
	if c := do("GET", "/a.txt", "ro", "p2", ""); c != 404 {
		t.Fatalf("用户之间目录应隔离: %d", c)
	}
}
