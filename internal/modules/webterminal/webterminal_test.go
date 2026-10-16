package webterminal

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"golang.org/x/crypto/ssh"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/luckyimport"
	"github.com/wangwanjie/gately/internal/module"
)

// startSSHServer 启动一个只接受指定公钥的 SSH 服务端：申请 pty 后把输入加上前缀回显。
func startSSHServer(t *testing.T, clientPub ssh.PublicKey) (string, ssh.PublicKey) {
	t.Helper()
	_, hostPriv, _ := ed25519.GenerateKey(rand.Reader)
	hostSigner, _ := ssh.NewSignerFromKey(hostPriv)
	cfg := &ssh.ServerConfig{PublicKeyCallback: func(_ ssh.ConnMetadata, k ssh.PublicKey) (*ssh.Permissions, error) {
		if string(k.Marshal()) == string(clientPub.Marshal()) {
			return nil, nil
		}
		return nil, io.EOF
	}}
	cfg.AddHostKey(hostSigner)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			nc, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				_, chans, reqs, err := ssh.NewServerConn(nc, cfg)
				if err != nil {
					return
				}
				go ssh.DiscardRequests(reqs)
				for nch := range chans {
					ch, creqs, _ := nch.Accept()
					go func() {
						for req := range creqs {
							_ = req.Reply(req.Type == "pty-req" || req.Type == "shell" || req.Type == "window-change", nil)
						}
					}()
					go func() {
						buf := make([]byte, 1024)
						for {
							n, err := ch.Read(buf)
							if err != nil {
								return
							}
							_, _ = ch.Write(append([]byte("echo:"), buf[:n]...))
						}
					}()
				}
			}()
		}
	}()
	return ln.Addr().String(), hostSigner.PublicKey()
}

func newKey(t *testing.T) (string, ssh.PublicKey) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	signer, _ := ssh.NewSignerFromKey(priv)
	return string(pem.EncodeToMemory(block)), signer.PublicKey()
}

type router struct{ mux *http.ServeMux }

func (r router) HandleFunc(p string, h http.HandlerFunc) {
	method, path, _ := strings.Cut(p, " ")
	r.mux.HandleFunc(method+" /api/webterminal"+path, h)
}

func newModule(t *testing.T) (*Module, *httptest.Server) {
	st, _ := config.Open(t.TempDir())
	m := New()
	if err := m.Init(module.Env{Store: st, Logs: logbuf.New(100, log.New(io.Discard, "", 0))}); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	m.Routes(router{mux})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return m, srv
}

// openWS 连接终端，返回读取函数与写函数。
func openWS(t *testing.T, srv *httptest.Server, key string) (*websocket.Conn, context.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	ws, _, err := websocket.Dial(ctx, strings.Replace(srv.URL, "http", "ws", 1)+"/api/webterminal/connections/"+key+"/ws?cols=100&rows=30", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ws.CloseNow() })
	return ws, ctx
}

// readUntil 读取消息直到包含 want。
func readUntil(t *testing.T, ws *websocket.Conn, ctx context.Context, want string) string {
	t.Helper()
	var all strings.Builder
	for !strings.Contains(all.String(), want) {
		_, b, err := ws.Read(ctx)
		if err != nil {
			t.Fatalf("等待 %q 失败: %v（已收到 %q）", want, err, all.String())
		}
		all.Write(b)
	}
	return all.String()
}

func TestSSHSessionAndHostKeyTOFU(t *testing.T) {
	keyPEM, pub := newKey(t)
	addr, hostPub := startSSHServer(t, pub)
	host, port, _ := net.SplitHostPort(addr)
	m, srv := newModule(t)
	c := Connection{Key: "c1", Name: "nas", SSH: SSHConfig{Host: host, Username: "root", AuthType: "key", PrivateKey: keyPEM}}
	c.SSH.Port, _ = strconv.Atoi(port)
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	m.cfg.Connections = []Connection{c}

	ws, ctx := openWS(t, srv, "c1")
	out := readUntil(t, ws, ctx, "首次连接")
	if !strings.Contains(out, Fingerprint(hostPub)) {
		t.Fatalf("首次连接应提示主机指纹: %s", out)
	}
	msg, _ := json.Marshal(clientMsg{T: "i", D: "ls\n"})
	_ = ws.Write(ctx, websocket.MessageText, msg)
	readUntil(t, ws, ctx, "echo:ls")
	resize, _ := json.Marshal(clientMsg{T: "r", C: 120, R: 40})
	_ = ws.Write(ctx, websocket.MessageText, resize)
	ws.Close(websocket.StatusNormalClosure, "")

	m.mu.Lock()
	recorded := m.cfg.Connections[0].SSH.HostKey
	m.mu.Unlock()
	if recorded != Fingerprint(hostPub) {
		t.Fatalf("应记录主机指纹: %q", recorded)
	}

	// 模拟主机密钥被替换（中间人）
	m.mu.Lock()
	m.cfg.Connections[0].SSH.HostKey = "SHA256:forged"
	m.mu.Unlock()
	ws2, ctx2 := openWS(t, srv, "c1")
	readUntil(t, ws2, ctx2, "主机密钥已变化")
}

func TestLocalShellGate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("本机 Shell 仅支持 Unix")
	}
	m, srv := newModule(t)
	m.cfg.Connections = []Connection{{Key: "l", Name: "local", Type: TypeLocal, Local: LocalConfig{Shell: "/bin/sh"}}}
	ws, ctx := openWS(t, srv, "l")
	readUntil(t, ws, ctx, "本机 Shell 未开启")

	m.cfg.Settings.AllowLocalShell = true
	ws2, ctx2 := openWS(t, srv, "l")
	msg, _ := json.Marshal(clientMsg{T: "i", D: "echo gately-$((40+2))\n"})
	_ = ws2.Write(ctx2, websocket.MessageText, msg)
	readUntil(t, ws2, ctx2, "gately-42")
}

func TestValidateAndMasking(t *testing.T) {
	keyPEM, _ := newKey(t)
	bad := []Connection{
		{Name: ""},
		{Name: "a", SSH: SSHConfig{Host: "h"}},
		{Name: "a", SSH: SSHConfig{Host: "h", Username: "u", AuthType: "key", PrivateKey: "garbage"}},
		{Name: "a", Type: "telnet"},
	}
	for i, c := range bad {
		if err := c.Validate(); err == nil {
			t.Errorf("第 %d 个非法配置未报错", i)
		}
	}
	m, _ := newModule(t)
	c := Connection{Name: "a", SSH: SSHConfig{Host: "h", Username: "u", AuthType: "key", PrivateKey: keyPEM}}
	_ = c.Validate()
	v := m.viewLocked(&c).(Connection)
	if v.SSH.PrivateKey != "********" {
		t.Fatal("私钥应脱敏")
	}
}

func TestImportLucky(t *testing.T) {
	keyPEM, _ := newKey(t)
	raw, _ := json.Marshal([]map[string]any{
		{"name": "pve", "type": "ssh", "sshConfig": map[string]any{"host": "192.168.1.2", "port": 22, "username": "root", "authType": "key", "privateKey": keyPEM, "keepAlive": 60, "proxy": map[string]any{"enabled": true}}},
		{"name": "old", "type": "telnet"},
	})
	m, _ := newModule(t)
	rep, err := m.ImportLucky(luckyimport.Document{"connections": raw}, luckyimport.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Found != 2 || rep.Imported != 1 || m.cfg.Connections[0].SSH.KeepAlive != 60 {
		t.Fatalf("导入结果错误: %+v", rep)
	}
	if path := os.Getenv("GATELY_LUCKY_TERM_SAMPLE"); path != "" {
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
