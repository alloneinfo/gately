package webterminal

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wangwanjie/gately/internal/crud"
	"github.com/wangwanjie/gately/internal/luckyimport"
)

type luckyProxy struct {
	Enabled  bool   `json:"enabled"`
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (p luckyProxy) convert() Proxy {
	t := strings.ToLower(p.Type)
	switch {
	case strings.HasPrefix(t, "http"):
		t = ProxyHTTP
	default:
		t = ProxySOCKS5
	}
	return Proxy{Enabled: p.Enabled, Type: t, Host: p.Host, Port: p.Port, Username: p.Username, Password: p.Password}
}

type luckyConn struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Remark      string `json:"remark"`
	LocalConfig struct {
		Shell   string            `json:"shell"`
		WorkDir string            `json:"workDir"`
		Env     map[string]string `json:"env"`
	} `json:"localConfig"`
	SSHConfig struct {
		Host       string     `json:"host"`
		Port       int        `json:"port"`
		Username   string     `json:"username"`
		AuthType   string     `json:"authType"`
		Password   string     `json:"password"`
		PrivateKey string     `json:"privateKey"`
		KeyFile    string     `json:"keyFile"`
		Passphrase string     `json:"passphrase"`
		WorkDir    string     `json:"workDir"`
		Timeout    int        `json:"timeout"`
		KeepAlive  int        `json:"keepAlive"`
		Proxy      luckyProxy `json:"proxy"`
	} `json:"sshConfig"`
	TelnetConfig struct {
		Host     string     `json:"host"`
		Port     int        `json:"port"`
		Username string     `json:"username"`
		Password string     `json:"password"`
		Timeout  int        `json:"timeout"`
		Proxy    luckyProxy `json:"proxy"`
	} `json:"telnetConfig"`
	QuickAccessDirs []string `json:"quickAccessDirs"`
}

type luckySessionConfig struct {
	IdleTimeout       int `json:"idleTimeout"`
	BufferSize        int `json:"bufferSize"`
	HeartbeatInterval int `json:"heartbeatInterval"`
	MaxSessions       int `json:"maxSessions"`
	SessionKeepAlive  int `json:"sessionKeepAlive"`
}

type luckyShortcut struct {
	Name      string `json:"name"`
	Command   string `json:"command"`
	SendEnter bool   `json:"sendEnter"`
}

type luckySecurity struct {
	TwoFAEnable           bool   `json:"twoFAEnable"`
	TwoFAKey              string `json:"twoFAKey"`
	UserAgreementAccepted bool   `json:"userAgreementAccepted"`
}

// luckyAuthType 把 Lucky 的认证方式名称映射为 password / key。
func luckyAuthType(t string) string {
	if strings.Contains(strings.ToLower(t), "key") {
		return "key"
	}
	return t
}

// ImportLucky 实现 luckyimport.Importer。
func (m *Module) ImportLucky(doc luckyimport.Document, opts luckyimport.Options) (*luckyimport.Report, error) {
	rawConns, hasConns := doc["connections"]
	rawSession, hasSession := doc["sessionConfig"]
	rawShortcuts, hasShortcuts := doc["globalShortcuts"]
	rawSecurity, hasSecurity := doc["securityConfig"]
	if !hasConns && !hasSession && !hasShortcuts && !hasSecurity {
		return nil, nil
	}
	var src []luckyConn
	if hasConns {
		if err := json.Unmarshal(rawConns, &src); err != nil {
			return nil, err
		}
	}
	rep := &luckyimport.Report{Module: ID, Title: m.Title(), Found: len(src)}
	var out []Connection
	for _, lc := range src {
		c := Connection{Name: lc.Name, Type: lc.Type, Remark: lc.Remark, QuickAccessDirs: lc.QuickAccessDirs}
		s, t := lc.SSHConfig, lc.TelnetConfig
		switch lc.Type {
		case TypeTelnet:
			c.Telnet = TelnetConfig{Host: t.Host, Port: t.Port, Username: t.Username, Password: t.Password, Timeout: t.Timeout, Proxy: t.Proxy.convert()}
		case TypeLocal:
			c.Local = LocalConfig{Shell: lc.LocalConfig.Shell, WorkDir: lc.LocalConfig.WorkDir, Env: lc.LocalConfig.Env}
		default:
			c.SSH = SSHConfig{Host: s.Host, Port: s.Port, Username: s.Username, AuthType: luckyAuthType(s.AuthType), Password: s.Password,
				PrivateKey: s.PrivateKey, KeyFile: s.KeyFile, Passphrase: s.Passphrase, WorkDir: s.WorkDir, Timeout: s.Timeout,
				KeepAlive: s.KeepAlive, Proxy: s.Proxy.convert()}
		}
		if err := c.Validate(); err != nil {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s: 跳过，%v", lc.Name, err))
			continue
		}
		out = append(out, c)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	existing := map[string]bool{}
	for _, c := range m.cfg.Connections {
		existing[c.Name] = true
	}
	fresh := out[:0]
	for _, c := range out {
		if existing[c.Name] {
			rep.Warnings = append(rep.Warnings, luckyimport.Skipped(c.Name))
			continue
		}
		fresh = append(fresh, c)
	}
	out = fresh
	rep.Imported = len(out)
	if len(out) > 0 {
		rep.Warnings = append(rep.Warnings, "主机密钥将在首次连接时记录（首次信任），请在首次连接时核对指纹")
	}

	// 先在副本上合并其余配置，保存成功后再替换，失败时不影响现有配置
	next := m.cfg
	next.Connections = append([]Connection(nil), m.cfg.Connections...)
	for _, c := range out {
		c.Key = crud.NewKey()
		next.Connections = append(next.Connections, c)
	}
	if hasSession {
		var sc luckySessionConfig
		if err := json.Unmarshal(rawSession, &sc); err == nil {
			next.Settings.MaxSessions = sc.MaxSessions
			next.Settings.IdleTimeout = sc.IdleTimeout
			next.Settings.BufferSize = sc.BufferSize
			next.Settings.HeartbeatInterval = sc.HeartbeatInterval
			next.Settings.SessionKeepAlive = sc.SessionKeepAlive
			next.Settings.normalize()
			rep.Warnings = append(rep.Warnings, "已迁移会话设置（最大会话数、空闲超时、回放缓冲、心跳间隔、断线保留）")
		}
	}
	if hasShortcuts {
		var list []luckyShortcut
		if err := json.Unmarshal(rawShortcuts, &list); err == nil && len(list) > 0 {
			next.Shortcuts = append([]Shortcut(nil), m.cfg.Shortcuts...)
			added := 0
			for _, sc := range list {
				if sc.Command == "" {
					continue
				}
				dup := false
				for _, ex := range next.Shortcuts {
					if ex.Name == sc.Name && ex.Command == sc.Command {
						dup = true
					}
				}
				if dup {
					continue
				}
				name := sc.Name
				if name == "" {
					name = sc.Command
				}
				next.Shortcuts = append(next.Shortcuts, Shortcut{Name: name, Command: sc.Command, SendEnter: sc.SendEnter})
				added++
			}
			if added > 0 {
				rep.Warnings = append(rep.Warnings, fmt.Sprintf("已迁移 %d 条全局快捷命令", added))
			}
		}
	}
	if hasSecurity {
		var sec luckySecurity
		if err := json.Unmarshal(rawSecurity, &sec); err == nil {
			if sec.UserAgreementAccepted {
				next.Security.AgreementAccepted = true
			}
			// 已有二次验证配置时保留现状，避免覆盖
			if sec.TwoFAEnable && sec.TwoFAKey != "" && !m.cfg.Security.TwoFAEnable {
				next.Security.TwoFAEnable = true
				next.Security.TwoFAKey = sec.TwoFAKey
				rep.Warnings = append(rep.Warnings, "已迁移二次验证密钥，请使用原验证器应用中的动态码验证")
			}
		}
	}
	if opts.DryRun {
		return rep, nil
	}
	old := m.cfg
	m.cfg = next
	if err := m.saveLocked(); err != nil {
		m.cfg = old
		return nil, err
	}
	return rep, nil
}
