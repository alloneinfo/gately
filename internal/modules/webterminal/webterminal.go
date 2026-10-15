// Package webterminal 实现 Web 终端：通过 WebSocket 在浏览器中使用 SSH 或本机 Shell。
//
// SSH 主机密钥采用"首次信任"：第一次连接时记录指纹，之后指纹变化会拒绝连接，防止中间人攻击。
// 本机 Shell 等同于服务器的完全控制权，默认关闭，需要在模块设置中显式开启。
package webterminal

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"golang.org/x/crypto/ssh"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/crud"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
)

// ID 是模块标识。
const ID = "webterminal"

// 连接类型
const (
	TypeSSH   = "ssh"
	TypeLocal = "local"
)

// SSHConfig 是 SSH 连接参数。
type SSHConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	// AuthType 为 password 或 key
	AuthType   string `json:"authType"`
	Password   string `json:"password"`
	PrivateKey string `json:"privateKey"`
	Passphrase string `json:"passphrase"`
	// HostKey 为首次连接时记录的主机密钥指纹（SHA256），为空表示下次连接时重新信任
	HostKey string `json:"hostKey"`
	// KeepAlive 为保活间隔（秒）
	KeepAlive int `json:"keepAlive"`
}

// LocalConfig 是本机 Shell 参数。
type LocalConfig struct {
	Shell   string `json:"shell"`
	WorkDir string `json:"workDir"`
}

// Connection 是一个终端连接配置。
type Connection struct {
	Key    string      `json:"key"`
	Name   string      `json:"name"`
	Type   string      `json:"type"`
	SSH    SSHConfig   `json:"ssh"`
	Local  LocalConfig `json:"local"`
	Remark string      `json:"remark"`
	// Enable 为复用通用接口保留，终端连接始终可用
	Enable bool `json:"enable"`
}

// Settings 是模块级配置。
type Settings struct {
	// AllowLocalShell 允许打开本机 Shell
	AllowLocalShell bool `json:"allowLocalShell"`
	MaxSessions     int  `json:"maxSessions"`
}

type fileConfig struct {
	Settings    Settings     `json:"settings"`
	Connections []Connection `json:"connections"`
}

// Validate 校验连接配置。
func (c *Connection) Validate() error {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		return errors.New("名称不能为空")
	}
	switch c.Type {
	case "", TypeSSH:
		c.Type = TypeSSH
		s := &c.SSH
		s.Host = strings.TrimSpace(s.Host)
		if s.Host == "" || s.Username == "" {
			return errors.New("请填写主机与用户名")
		}
		if s.Port == 0 {
			s.Port = 22
		}
		if s.Port < 1 || s.Port > 65535 {
			return errors.New("端口无效")
		}
		switch s.AuthType {
		case "", "password":
			s.AuthType = "password"
		case "key":
			if s.PrivateKey == "" {
				return errors.New("请填写私钥")
			}
			if s.PrivateKey != crud.SecretMask {
				if _, err := parseKey(s.PrivateKey, s.Passphrase); err != nil {
					return fmt.Errorf("私钥无效: %w", err)
				}
			}
		default:
			return fmt.Errorf("未知的认证方式 %q", s.AuthType)
		}
	case TypeLocal:
	default:
		return fmt.Errorf("未知的连接类型 %q", c.Type)
	}
	c.Enable = true
	return nil
}

func parseKey(key, passphrase string) (ssh.Signer, error) {
	if passphrase != "" {
		return ssh.ParsePrivateKeyWithPassphrase([]byte(key), []byte(passphrase))
	}
	return ssh.ParsePrivateKey([]byte(key))
}

// Fingerprint 返回主机密钥的 SHA256 指纹（与 ssh-keygen -lf 输出格式一致）。
func Fingerprint(k ssh.PublicKey) string {
	h := sha256.Sum256(k.Marshal())
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(h[:])
}

// Module 是 Web 终端模块。
type Module struct {
	mu       sync.Mutex
	store    *config.Store
	logs     *logbuf.Buffer
	cfg      fileConfig
	sessions atomic.Int64
}

// New 创建模块。
func New() *Module { return &Module{} }

// ID 实现 module.Module。
func (m *Module) ID() string { return ID }

// Title 实现 module.Module。
func (m *Module) Title() string { return "Web 终端" }

// Init 实现 module.Module。
func (m *Module) Init(env module.Env) error {
	m.store, m.logs = env.Store, env.Logs
	m.cfg.Settings.MaxSessions = 20
	_, err := m.store.Load(ID, &m.cfg)
	if m.cfg.Settings.MaxSessions <= 0 {
		m.cfg.Settings.MaxSessions = 20
	}
	return err
}

// Start 实现 module.Module。
func (m *Module) Start(context.Context) error { return nil }

// Stop 实现 module.Module。
func (m *Module) Stop() error { return nil }

func (m *Module) saveLocked() error { return m.store.Save(ID, m.cfg) }

func (m *Module) viewLocked(c *Connection) any {
	v := *c
	v.SSH.Password = crud.Mask(c.SSH.Password)
	v.SSH.PrivateKey = crud.Mask(c.SSH.PrivateKey)
	v.SSH.Passphrase = crud.Mask(c.SSH.Passphrase)
	return v
}

// Routes 实现 module.Module。
func (m *Module) Routes(r module.Router) {
	conns := &crud.Collection[Connection]{
		Mu:       &m.mu,
		Items:    &m.cfg.Connections,
		Save:     m.saveLocked,
		Key:      func(c *Connection) *string { return &c.Key },
		Enabled:  func(c *Connection) *bool { return &c.Enable },
		Validate: func(c *Connection) error { return c.Validate() },
		Merge: func(old, updated *Connection) {
			crud.KeepSecret(old.SSH.Password, &updated.SSH.Password)
			crud.KeepSecret(old.SSH.PrivateKey, &updated.SSH.PrivateKey)
			crud.KeepSecret(old.SSH.Passphrase, &updated.SSH.Passphrase)
			// 主机或端口变化时需要重新信任主机密钥
			if old.SSH.Host != updated.SSH.Host || old.SSH.Port != updated.SSH.Port {
				updated.SSH.HostKey = ""
			} else if updated.SSH.HostKey == "" {
				updated.SSH.HostKey = old.SSH.HostKey
			}
		},
		View: m.viewLocked,
	}
	conns.Register(r, "/connections")
	r.HandleFunc("DELETE /connections/{key}/hostkey", m.handleResetHostKey)
	r.HandleFunc("GET /connections/{key}/ws", m.handleWS)
	r.HandleFunc("GET /settings", func(w http.ResponseWriter, req *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		httpx.OK(w, map[string]any{"settings": m.cfg.Settings, "activeSessions": m.sessions.Load()})
	})
	r.HandleFunc("PUT /settings", func(w http.ResponseWriter, req *http.Request) {
		var s Settings
		if err := httpx.ReadJSON(req, &s); err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
		if s.MaxSessions <= 0 {
			s.MaxSessions = 20
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		old := m.cfg.Settings
		m.cfg.Settings = s
		if err := m.saveLocked(); err != nil {
			m.cfg.Settings = old
			httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
			return
		}
		m.logs.Add(logbuf.LevelInfo, "", fmt.Sprintf("设置已修改：本机 Shell=%v", s.AllowLocalShell))
		httpx.OK(w, s)
	})
}

func (m *Module) handleResetHostKey(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.cfg.Connections {
		if m.cfg.Connections[i].Key == r.PathValue("key") {
			m.cfg.Connections[i].SSH.HostKey = ""
			if err := m.saveLocked(); err != nil {
				httpx.Error(w, http.StatusInternalServerError, "%v", err)
				return
			}
			httpx.OK(w, map[string]bool{"ok": true})
			return
		}
	}
	httpx.Error(w, http.StatusNotFound, "连接不存在")
}

// clientMsg 是浏览器发来的消息：t=i 输入，t=r 调整窗口大小。
type clientMsg struct {
	T string `json:"t"`
	D string `json:"d"`
	C int    `json:"c"`
	R int    `json:"r"`
}

// session 是一个已建立的终端会话的抽象。
type session interface {
	io.ReadWriter
	Resize(cols, rows int) error
	Close() error
}

func (m *Module) handleWS(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	m.mu.Lock()
	var conn *Connection
	for i := range m.cfg.Connections {
		if m.cfg.Connections[i].Key == key {
			c := m.cfg.Connections[i]
			conn = &c
		}
	}
	settings := m.cfg.Settings
	m.mu.Unlock()
	if conn == nil {
		httpx.Error(w, http.StatusNotFound, "连接不存在")
		return
	}
	if m.sessions.Load() >= int64(settings.MaxSessions) {
		httpx.Error(w, http.StatusTooManyRequests, "会话数已达上限 %d", settings.MaxSessions)
		return
	}
	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	ws.SetReadLimit(1 << 20)
	m.sessions.Add(1)
	defer m.sessions.Add(-1)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	cols, _ := strconv.Atoi(r.URL.Query().Get("cols"))
	rows, _ := strconv.Atoi(r.URL.Query().Get("rows"))
	if cols <= 0 || rows <= 0 {
		cols, rows = 80, 24
	}
	notice := func(msg string) {
		b, _ := json.Marshal(map[string]string{"t": "n", "d": msg})
		_ = ws.Write(ctx, websocket.MessageText, b)
	}
	log := m.logs.Logger(conn.Name)

	var sess session
	switch conn.Type {
	case TypeLocal:
		if !settings.AllowLocalShell {
			notice("本机 Shell 未开启，请在 Web 终端设置中允许后再使用")
			_ = ws.Close(websocket.StatusPolicyViolation, "local shell disabled")
			return
		}
		sess, err = startLocal(conn.Local, cols, rows)
	default:
		sess, err = m.startSSH(ctx, conn, cols, rows, notice)
	}
	if err != nil {
		log.Warnf("连接失败: %v", err)
		notice("连接失败: " + err.Error())
		_ = ws.Close(websocket.StatusNormalClosure, "")
		return
	}
	defer sess.Close()
	log.Infof("会话开始（来自 %s）", r.RemoteAddr)
	started := time.Now()

	// 终端输出 -> 浏览器
	go func() {
		defer cancel()
		buf := make([]byte, 32*1024)
		for {
			n, err := sess.Read(buf)
			if n > 0 {
				if werr := ws.Write(ctx, websocket.MessageBinary, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	// 浏览器输入 -> 终端
	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			break
		}
		var msg clientMsg
		if json.Unmarshal(data, &msg) != nil {
			continue
		}
		switch msg.T {
		case "i":
			if _, err := sess.Write([]byte(msg.D)); err != nil {
				cancel()
			}
		case "r":
			if msg.C > 0 && msg.R > 0 {
				_ = sess.Resize(msg.C, msg.R)
			}
		}
		if ctx.Err() != nil {
			break
		}
	}
	log.Infof("会话结束，持续 %s", time.Since(started).Round(time.Second))
	_ = ws.Close(websocket.StatusNormalClosure, "")
}

// sshSession 包装 SSH 客户端与会话。
type sshSession struct {
	client *ssh.Client
	sess   *ssh.Session
	stdin  io.WriteCloser
	out    io.Reader
	stop   chan struct{}
}

func (s *sshSession) Read(p []byte) (int, error)  { return s.out.Read(p) }
func (s *sshSession) Write(p []byte) (int, error) { return s.stdin.Write(p) }
func (s *sshSession) Resize(c, r int) error       { return s.sess.WindowChange(r, c) }
func (s *sshSession) Close() error {
	close(s.stop)
	s.sess.Close()
	return s.client.Close()
}

// dialSSH 允许测试替换网络拨号。
var dialSSH = func(ctx context.Context, addr string, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{Timeout: timeout}
	return d.DialContext(ctx, "tcp", addr)
}

func (m *Module) startSSH(ctx context.Context, c *Connection, cols, rows int, notice func(string)) (session, error) {
	s := c.SSH
	var auth []ssh.AuthMethod
	if s.AuthType == "key" {
		signer, err := parseKey(s.PrivateKey, s.Passphrase)
		if err != nil {
			return nil, err
		}
		auth = append(auth, ssh.PublicKeys(signer))
	} else {
		auth = append(auth, ssh.Password(s.Password), ssh.KeyboardInteractive(func(_, _ string, qs []string, _ []bool) ([]string, error) {
			ans := make([]string, len(qs))
			for i := range ans {
				ans[i] = s.Password
			}
			return ans, nil
		}))
	}
	var newFingerprint string
	cfg := &ssh.ClientConfig{
		User:    s.Username,
		Auth:    auth,
		Timeout: 15 * time.Second,
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			fp := Fingerprint(key)
			if s.HostKey == "" {
				newFingerprint = fp
				return nil
			}
			if fp != s.HostKey {
				return fmt.Errorf("主机密钥已变化（记录 %s，当前 %s），可能存在中间人攻击；确认无误后在连接设置中重置主机密钥", s.HostKey, fp)
			}
			return nil
		},
	}
	addr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	nc, err := dialSSH(ctx, addr, cfg.Timeout)
	if err != nil {
		return nil, err
	}
	cc, chans, reqs, err := ssh.NewClientConn(nc, addr, cfg)
	if err != nil {
		nc.Close()
		return nil, err
	}
	client := ssh.NewClient(cc, chans, reqs)
	if newFingerprint != "" {
		m.recordHostKey(c.Key, newFingerprint)
		notice("首次连接，已信任主机密钥 " + newFingerprint)
	}
	sess, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, err
	}
	stdin, _ := sess.StdinPipe()
	pr, pw := io.Pipe()
	sess.Stdout, sess.Stderr = pw, pw
	modes := ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}
	if err := sess.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		client.Close()
		return nil, err
	}
	if err := sess.Shell(); err != nil {
		client.Close()
		return nil, err
	}
	ss := &sshSession{client: client, sess: sess, stdin: stdin, out: pr, stop: make(chan struct{})}
	go func() {
		_ = sess.Wait()
		pw.Close()
	}()
	// 定期发送保活请求，避免空闲连接被中间设备断开
	if ka := s.KeepAlive; ka > 0 {
		go func() {
			t := time.NewTicker(time.Duration(ka) * time.Second)
			defer t.Stop()
			for {
				select {
				case <-ss.stop:
					return
				case <-t.C:
					if _, _, err := client.SendRequest("keepalive@openssh.com", true, nil); err != nil {
						return
					}
				}
			}
		}()
	}
	return ss, nil
}

func (m *Module) recordHostKey(key, fp string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.cfg.Connections {
		if m.cfg.Connections[i].Key == key && m.cfg.Connections[i].SSH.HostKey == "" {
			m.cfg.Connections[i].SSH.HostKey = fp
			_ = m.saveLocked()
		}
	}
}
