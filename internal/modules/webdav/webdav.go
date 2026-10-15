// Package webdav 实现 WebDAV 文件服务：多用户、每个用户映射到独立目录，可设为只读。
package webdav

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	xwebdav "golang.org/x/net/webdav"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/crud"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
)

// ID 是模块标识。
const ID = "webdav"

// User 是一个 WebDAV 账号。
type User struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	// Root 为该用户可访问的目录（绝对路径）
	Root     string `json:"root"`
	ReadOnly bool   `json:"readOnly"`
}

// Settings 是模块配置。
type Settings struct {
	Enable   bool   `json:"enable"`
	ListenIP string `json:"listenIp"`
	Port     int    `json:"port"`
	// TLS 时证书由证书模块按 SNI 提供
	TLS   bool   `json:"tls"`
	Users []User `json:"users"`
}

// CertSource 提供 HTTPS 证书。
type CertSource interface {
	GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)
}

// Validate 校验配置。
func (s *Settings) Validate() error {
	if s.Port == 0 {
		s.Port = 18800
	}
	if s.Port < 1 || s.Port > 65535 {
		return errors.New("端口无效")
	}
	s.ListenIP = strings.Trim(strings.TrimSpace(s.ListenIP), "[]")
	if s.ListenIP != "" && net.ParseIP(s.ListenIP) == nil {
		return errors.New("监听地址无效")
	}
	seen := map[string]bool{}
	for i := range s.Users {
		u := &s.Users[i]
		u.Name = strings.TrimSpace(u.Name)
		if u.Name == "" || u.Password == "" {
			return errors.New("用户名和密码不能为空")
		}
		if seen[u.Name] {
			return fmt.Errorf("用户 %s 重复", u.Name)
		}
		seen[u.Name] = true
		if !filepath.IsAbs(u.Root) {
			return fmt.Errorf("用户 %s 的目录需为绝对路径", u.Name)
		}
	}
	if s.Enable && len(s.Users) == 0 {
		return errors.New("启用 WebDAV 时至少添加一个用户")
	}
	return nil
}

// readOnlyFS 拒绝所有写操作。
type readOnlyFS struct{ xwebdav.FileSystem }

func (readOnlyFS) Mkdir(context.Context, string, os.FileMode) error { return os.ErrPermission }
func (readOnlyFS) RemoveAll(context.Context, string) error          { return os.ErrPermission }
func (readOnlyFS) Rename(context.Context, string, string) error     { return os.ErrPermission }
func (r readOnlyFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (xwebdav.File, error) {
	if flag&(os.O_WRONLY|os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_APPEND) != 0 {
		return nil, os.ErrPermission
	}
	return r.FileSystem.OpenFile(ctx, name, flag, perm)
}

type userHandler struct {
	user    User
	handler *xwebdav.Handler
}

// Module 是 WebDAV 模块。
type Module struct {
	mu       sync.Mutex
	store    *config.Store
	logs     *logbuf.Buffer
	certs    CertSource
	settings Settings
	srv      *http.Server
	err      string
	handlers map[string]*userHandler
}

// New 创建模块。
func New(certs CertSource) *Module { return &Module{certs: certs} }

// ID 实现 module.Module。
func (m *Module) ID() string { return ID }

// Title 实现 module.Module。
func (m *Module) Title() string { return "WebDAV" }

// Init 实现 module.Module。
func (m *Module) Init(env module.Env) error {
	m.store, m.logs = env.Store, env.Logs
	m.settings.Port = 18800
	_, err := m.store.Load(ID, &m.settings)
	return err
}

// Start 实现 module.Module。
func (m *Module) Start(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.restartLocked()
	return nil
}

// Stop 实现 module.Module。
func (m *Module) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopLocked()
	return nil
}

func (m *Module) stopLocked() {
	if m.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = m.srv.Shutdown(ctx)
		m.srv = nil
	}
}

func (m *Module) restartLocked() {
	m.stopLocked()
	m.err = ""
	if !m.settings.Enable {
		return
	}
	m.handlers = map[string]*userHandler{}
	for _, u := range m.settings.Users {
		var fs xwebdav.FileSystem = xwebdav.Dir(u.Root)
		if u.ReadOnly {
			fs = readOnlyFS{fs}
		}
		m.handlers[u.Name] = &userHandler{user: u, handler: &xwebdav.Handler{FileSystem: fs, LockSystem: xwebdav.NewMemLS()}}
	}
	addr := net.JoinHostPort(m.settings.ListenIP, strconv.Itoa(m.settings.Port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		m.err = err.Error()
		m.logs.Add(logbuf.LevelError, "", "监听失败: "+err.Error())
		return
	}
	m.srv = &http.Server{Handler: http.HandlerFunc(m.serve), ReadHeaderTimeout: 30 * time.Second}
	if m.settings.TLS {
		if m.certs == nil {
			ln.Close()
			m.err = "证书模块不可用"
			return
		}
		m.srv.TLSConfig = &tls.Config{GetCertificate: m.certs.GetCertificate, MinVersion: tls.VersionTLS12}
		ln = tls.NewListener(ln, m.srv.TLSConfig)
	}
	srv := m.srv
	go func() { _ = srv.Serve(ln) }()
	m.logs.Add(logbuf.LevelInfo, "", fmt.Sprintf("WebDAV 监听 %s（%d 个用户）", addr, len(m.handlers)))
}

func checkPassword(stored, given string) bool {
	if strings.HasPrefix(stored, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(given)) == nil
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(given)) == 1
}

func (m *Module) serve(w http.ResponseWriter, r *http.Request) {
	name, pass, ok := r.BasicAuth()
	m.mu.Lock()
	uh := m.handlers[name]
	m.mu.Unlock()
	if !ok || uh == nil || !checkPassword(uh.user.Password, pass) {
		if ok {
			m.logs.Add(logbuf.LevelWarn, "", fmt.Sprintf("%s 认证失败（用户 %q）", r.RemoteAddr, name))
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="Gately WebDAV"`)
		http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
		return
	}
	uh.handler.ServeHTTP(w, r)
}

// Routes 实现 module.Module。
func (m *Module) Routes(r module.Router) {
	r.HandleFunc("GET /settings", func(w http.ResponseWriter, req *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		s := m.settings
		s.Users = append([]User{}, s.Users...)
		for i := range s.Users {
			s.Users[i].Password = crud.Mask(s.Users[i].Password)
		}
		httpx.OK(w, map[string]any{"settings": s, "running": m.srv != nil, "error": m.err})
	})
	r.HandleFunc("PUT /settings", func(w http.ResponseWriter, req *http.Request) {
		var s Settings
		if err := httpx.ReadJSON(req, &s); err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		old := map[string]string{}
		for _, u := range m.settings.Users {
			old[u.Name] = u.Password
		}
		for i := range s.Users {
			crud.KeepSecret(old[s.Users[i].Name], &s.Users[i].Password)
		}
		if err := s.Validate(); err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
		if err := m.store.Save(ID, s); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "%v", err)
			return
		}
		m.settings = s
		m.restartLocked()
		httpx.OK(w, map[string]any{"running": m.srv != nil, "error": m.err})
	})
}
