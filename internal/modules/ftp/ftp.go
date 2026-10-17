// Package ftp 实现 FTP 文件服务：多用户、每个用户映射到一个或多个目录，可设为只读，支持 FTPS 与被动端口范围。
package ftp

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	ftpserver "github.com/fclairamb/ftpserverlib"
	"github.com/spf13/afero"
	"golang.org/x/crypto/bcrypt"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/crud"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/mountfs"
)

// ID 是模块标识。
const ID = "ftp"

// TLS 模式，与 Lucky 的 TLSRequired 取值一致。
const (
	TLSOptional  = 0 // 明文与 FTPS 均可
	TLSMandatory = 1 // 必须先 AUTH TLS（显式 FTPS）
	TLSImplicit  = 2 // 隐式 FTPS，连接即加密
)

// User 是一个 FTP 账号。
type User struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	// Root 为该用户可访问的目录（绝对路径），ReadOnly 作用于该目录；Mounts 非空时二者不再使用
	Root     string `json:"root"`
	ReadOnly bool   `json:"readOnly"`
	// Mounts 为多个目录：登录后根目录按名称列出各目录，每个目录可单独设为只读
	Mounts []mountfs.Mount `json:"mounts,omitempty"`
}

// Settings 是模块配置。
type Settings struct {
	Enable   bool   `json:"enable"`
	ListenIP string `json:"listenIp"`
	Port     int    `json:"port"`
	// PassiveStart/PassiveEnd 为被动模式数据端口范围，均为 0 时由系统随机分配
	PassiveStart int `json:"passiveStart"`
	PassiveEnd   int `json:"passiveEnd"`
	// PublicHost 为被动模式对外通告的 IP（位于 NAT 之后时填写公网或转发后的 IP），留空自动取连接地址
	PublicHost string `json:"publicHost"`
	// TLSMode 取 TLSOptional / TLSMandatory / TLSImplicit，非 0 时证书由证书模块按 SNI 提供
	TLSMode int `json:"tlsMode"`
	// IdleTimeout 控制连接空闲多少秒后断开，ConnectionTimeout 为建立数据连接的超时秒数
	IdleTimeout       int    `json:"idleTimeout"`
	ConnectionTimeout int    `json:"connectionTimeout"`
	DisableActiveMode bool   `json:"disableActiveMode"`
	Users             []User `json:"users"`
}

// CertSource 提供 FTPS 证书。
type CertSource interface {
	GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)
}

// Validate 校验并规范化配置。
func (s *Settings) Validate() error {
	if s.Port == 0 {
		s.Port = 2121
	}
	if s.Port < 1 || s.Port > 65535 {
		return errors.New("端口无效")
	}
	s.ListenIP = strings.Trim(strings.TrimSpace(s.ListenIP), "[]")
	if s.ListenIP != "" && net.ParseIP(s.ListenIP) == nil {
		return errors.New("监听地址无效")
	}
	s.PublicHost = strings.TrimSpace(s.PublicHost)
	if s.PublicHost != "" && net.ParseIP(s.PublicHost) == nil {
		return errors.New("被动模式对外 IP 无效，仅支持 IP 地址")
	}
	if s.PassiveStart != 0 || s.PassiveEnd != 0 {
		if s.PassiveStart < 1 || s.PassiveEnd > 65535 || s.PassiveStart > s.PassiveEnd {
			return errors.New("被动端口范围无效")
		}
	}
	if s.TLSMode < TLSOptional || s.TLSMode > TLSImplicit {
		return errors.New("TLS 模式无效")
	}
	if s.IdleTimeout < 0 || s.ConnectionTimeout < 0 {
		return errors.New("超时时间不能为负数")
	}
	if s.IdleTimeout == 0 {
		s.IdleTimeout = 900
	}
	if s.ConnectionTimeout == 0 {
		s.ConnectionTimeout = 30
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
		if len(u.Mounts) > 0 {
			// 多目录模式下根目录为虚拟目录，单目录字段清空，避免两套配置并存
			u.Root, u.ReadOnly = "", false
			if err := mountfs.Validate(u.Mounts); err != nil {
				return fmt.Errorf("用户 %s: %w", u.Name, err)
			}
			continue
		}
		if !filepath.IsAbs(u.Root) {
			return fmt.Errorf("用户 %s 的目录需为绝对路径", u.Name)
		}
	}
	if s.Enable && len(s.Users) == 0 {
		return errors.New("启用 FTP 时至少添加一个用户")
	}
	return nil
}

func checkPassword(stored, given string) bool {
	if strings.HasPrefix(stored, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(given)) == nil
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(given)) == 1
}

// driver 实现 ftpserver.MainDriver，每次启动持有一份配置快照。
type driver struct {
	s     Settings
	users map[string]User
	certs CertSource
	logs  *logbuf.Buffer
}

func (d *driver) GetSettings() (*ftpserver.Settings, error) {
	st := &ftpserver.Settings{
		ListenAddr:        net.JoinHostPort(d.s.ListenIP, strconv.Itoa(d.s.Port)),
		PublicHost:        d.s.PublicHost,
		Banner:            "Gately FTP",
		IdleTimeout:       d.s.IdleTimeout,
		ConnectionTimeout: d.s.ConnectionTimeout,
		DisableActiveMode: d.s.DisableActiveMode,
		TLSRequired:       ftpserver.TLSRequirement(d.s.TLSMode),
	}
	if d.s.PassiveStart != 0 {
		st.PassiveTransferPortRange = &ftpserver.PortRange{Start: d.s.PassiveStart, End: d.s.PassiveEnd}
	}
	return st, nil
}

func (d *driver) ClientConnected(cc ftpserver.ClientContext) (string, error) {
	return "Gately FTP", nil
}

func (d *driver) ClientDisconnected(cc ftpserver.ClientContext) {}

func (d *driver) AuthUser(cc ftpserver.ClientContext, user, pass string) (ftpserver.ClientDriver, error) {
	u, ok := d.users[user]
	if !ok || !checkPassword(u.Password, pass) {
		d.logs.Add(logbuf.LevelWarn, "", fmt.Sprintf("%s 认证失败（用户 %q）", cc.RemoteAddr(), user))
		return nil, errors.New("用户名或密码错误")
	}
	var fs afero.Fs
	if len(u.Mounts) > 0 {
		fs = mountfs.Afero(u.Mounts)
	} else {
		fs = afero.NewBasePathFs(afero.NewOsFs(), u.Root)
		if u.ReadOnly {
			fs = afero.NewReadOnlyFs(fs)
		}
	}
	d.logs.Add(logbuf.LevelInfo, "", fmt.Sprintf("%s 登录（用户 %q）", cc.RemoteAddr(), user))
	return fs, nil
}

func (d *driver) GetTLSConfig() (*tls.Config, error) {
	if d.certs == nil {
		return nil, errors.New("证书模块不可用")
	}
	return &tls.Config{GetCertificate: d.certs.GetCertificate, MinVersion: tls.VersionTLS12}, nil
}

// Module 是 FTP 模块。
type Module struct {
	mu       sync.Mutex
	store    *config.Store
	logs     *logbuf.Buffer
	certs    CertSource
	settings Settings
	srv      *ftpserver.FtpServer
	err      string
}

// New 创建模块。
func New(certs CertSource) *Module { return &Module{certs: certs} }

// ID 实现 module.Module。
func (m *Module) ID() string { return ID }

// Title 实现 module.Module。
func (m *Module) Title() string { return "FTP 服务" }

// Init 实现 module.Module。
func (m *Module) Init(env module.Env) error {
	m.store, m.logs = env.Store, env.Logs
	m.settings.Port = 2121
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
		_ = m.srv.Stop()
		m.srv = nil
	}
}

func (m *Module) restartLocked() {
	m.stopLocked()
	m.err = ""
	if !m.settings.Enable {
		return
	}
	if m.settings.TLSMode != TLSOptional && m.certs == nil {
		m.err = "证书模块不可用"
		return
	}
	d := &driver{s: m.settings, users: map[string]User{}, certs: m.certs, logs: m.logs}
	for _, u := range m.settings.Users {
		d.users[u.Name] = u
	}
	srv := ftpserver.NewFtpServer(d)
	if err := srv.Listen(); err != nil {
		m.err = err.Error()
		m.logs.Add(logbuf.LevelError, "", "监听失败: "+err.Error())
		return
	}
	m.srv = srv
	go func() {
		// Serve 在 Stop 后返回，其他错误写入运行日志
		if err := srv.Serve(); err != nil {
			m.logs.Add(logbuf.LevelWarn, "", "FTP 服务退出: "+err.Error())
		}
	}()
	m.logs.Add(logbuf.LevelInfo, "", fmt.Sprintf("FTP 监听 %s（%d 个用户）", srv.Addr(), len(d.users)))
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
