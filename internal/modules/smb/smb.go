// Package smb 实现 SMB 文件服务：多用户，每个用户映射到一个或多个目录，每个目录作为一个共享，可设为只读。
// 协议实现来自内置的 gosamba（internal/gosamba），支持 SMB 2.0.2 ~ 3.1.1、NTLMv2 认证、签名与加密。
package smb

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/crud"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/mountfs"
)

// ID 是模块标识。
const ID = "smb"

// DefaultPort 是 SMB 的标准端口。Windows 只会连接这个端口。
const DefaultPort = 445

// User 是一个 SMB 账号，结构与 FTP、WebDAV 的用户一致。
type User struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	// Root 为单目录模式下的目录（绝对路径），共享名为用户名；Mounts 非空时 Root 与 ReadOnly 不再使用
	Root     string `json:"root"`
	ReadOnly bool   `json:"readOnly"`
	// Mounts 为多个目录，每个目录以其名称作为一个共享，可单独设为只读
	Mounts []mountfs.Mount `json:"mounts,omitempty"`
}

// Settings 是模块配置。
type Settings struct {
	Enable   bool   `json:"enable"`
	ListenIP string `json:"listenIp"`
	Port     int    `json:"port"`
	// RequireEncryption 要求客户端使用 SMB 3 加密，不支持加密的客户端（SMB 2.x）将无法连接
	RequireEncryption bool `json:"requireEncryption"`
	// MDNS 通过 Bonjour 广播服务，macOS 访达的「网络」中可以直接看到
	MDNS  bool   `json:"mdns"`
	Users []User `json:"users"`
}

// Share 是用户可见的一个共享。
type Share struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	ReadOnly bool   `json:"readOnly"`
}

// Shares 返回用户的共享列表：单目录模式下只有一个以用户名命名的共享，多目录模式下每个目录一个共享。
func (u User) Shares() []Share {
	if len(u.Mounts) == 0 {
		return []Share{{Name: u.Name, Path: u.Root, ReadOnly: u.ReadOnly}}
	}
	out := make([]Share, 0, len(u.Mounts))
	for _, mt := range u.Mounts {
		out = append(out, Share{Name: mt.Name, Path: mt.Path, ReadOnly: mt.ReadOnly})
	}
	return out
}

// invalidNameChars 为 Windows 共享名与账号名都不允许的字符，另外不允许控制字符。
const invalidNameChars = `"/\[]:;|=,+*?<>`

// validName 检查用户名或共享名能否被 Windows、macOS 与 Linux 客户端使用。
func validName(name string) bool {
	if name == "" || len([]rune(name)) > 80 || strings.ContainsAny(name, invalidNameChars) {
		return false
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	// IPC$ 是协议保留的管道共享，以 . 开头或结尾的名称在 Windows 上无法访问
	return !strings.EqualFold(name, "IPC$") && !strings.HasPrefix(name, ".") && !strings.HasSuffix(name, ".")
}

// Validate 校验并规范化配置。
func (s *Settings) Validate() error {
	if s.Port == 0 {
		s.Port = DefaultPort
	}
	if s.Port < 1 || s.Port > 65535 {
		return errors.New("端口无效")
	}
	s.ListenIP = strings.Trim(strings.TrimSpace(s.ListenIP), "[]")
	if s.ListenIP != "" && net.ParseIP(s.ListenIP) == nil {
		return errors.New("监听地址无效")
	}
	// SMB 客户端按不区分大小写匹配用户名与共享名，重复检查也不区分大小写
	seen := map[string]bool{}
	for i := range s.Users {
		u := &s.Users[i]
		u.Name = strings.TrimSpace(u.Name)
		if u.Name == "" || u.Password == "" {
			return errors.New("用户名和密码不能为空")
		}
		if !validName(u.Name) {
			return fmt.Errorf("用户名 %q 无效，不能包含 %s 等字符", u.Name, invalidNameChars)
		}
		key := strings.ToLower(u.Name)
		if seen[key] {
			return fmt.Errorf("用户 %s 重复", u.Name)
		}
		seen[key] = true
		// NTLM 认证需要由明文密码计算 NT 哈希，bcrypt 哈希无法使用
		if strings.HasPrefix(u.Password, "$2") && len(u.Password) == 60 {
			return fmt.Errorf("用户 %s 的密码是 bcrypt 哈希，SMB 需要明文密码，请重新设置", u.Name)
		}
		if len(u.Mounts) > 0 {
			// 多目录模式下单目录字段清空，避免两套配置并存
			u.Root, u.ReadOnly = "", false
			if err := mountfs.Validate(u.Mounts); err != nil {
				return fmt.Errorf("用户 %s: %w", u.Name, err)
			}
			names := map[string]bool{}
			for _, mt := range u.Mounts {
				if !validName(mt.Name) {
					return fmt.Errorf("用户 %s 的共享名 %q 无效，不能包含 %s 等字符", u.Name, mt.Name, invalidNameChars)
				}
				if names[strings.ToLower(mt.Name)] {
					return fmt.Errorf("用户 %s 的共享名 %s 重复（不区分大小写）", u.Name, mt.Name)
				}
				names[strings.ToLower(mt.Name)] = true
			}
			continue
		}
		u.Root = strings.TrimSpace(u.Root)
		if !filepath.IsAbs(u.Root) {
			return fmt.Errorf("用户 %s 的目录需为绝对路径", u.Name)
		}
	}
	if s.Enable && len(s.Users) == 0 {
		return errors.New("启用 SMB 时至少添加一个用户")
	}
	return nil
}

// Module 是 SMB 模块。
type Module struct {
	mu       sync.Mutex
	store    *config.Store
	logs     *logbuf.Buffer
	settings Settings
	srv      *server
	err      string
}

// New 创建模块。
func New() *Module { return &Module{} }

// ID 实现 module.Module。
func (m *Module) ID() string { return ID }

// Title 实现 module.Module。
func (m *Module) Title() string { return "SMB 服务" }

// Init 实现 module.Module。
func (m *Module) Init(env module.Env) error {
	m.store, m.logs = env.Store, env.Logs
	m.settings.Port = DefaultPort
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
		m.srv.Close()
		m.srv = nil
	}
}

func (m *Module) restartLocked() {
	m.stopLocked()
	m.err = ""
	if !m.settings.Enable {
		return
	}
	srv, err := startServer(m.settings, m.logs)
	if err != nil {
		m.err = err.Error()
		m.logs.Add(logbuf.LevelError, "", "监听失败: "+err.Error())
		return
	}
	m.srv = srv
	m.logs.Add(logbuf.LevelInfo, "", fmt.Sprintf("SMB 监听 %s（%d 个用户）", srv.Addr(), len(m.settings.Users)))
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
		httpx.OK(w, map[string]any{"settings": s, "running": m.srv != nil, "error": m.err, "supported": supported})
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
