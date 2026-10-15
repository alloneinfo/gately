package webterminal

import (
	"encoding/json"
	"fmt"

	"github.com/wangwanjie/gately/internal/crud"
	"github.com/wangwanjie/gately/internal/luckyimport"
)

type luckyConn struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Remark      string `json:"remark"`
	LocalConfig struct {
		Shell   string `json:"shell"`
		WorkDir string `json:"workDir"`
	} `json:"localConfig"`
	SSHConfig struct {
		Host       string `json:"host"`
		Port       int    `json:"port"`
		Username   string `json:"username"`
		AuthType   string `json:"authType"`
		Password   string `json:"password"`
		PrivateKey string `json:"privateKey"`
		Passphrase string `json:"passphrase"`
		KeepAlive  int    `json:"keepAlive"`
		Proxy      struct {
			Enabled bool `json:"enabled"`
		} `json:"proxy"`
	} `json:"sshConfig"`
}

// ImportLucky 实现 luckyimport.Importer。
func (m *Module) ImportLucky(doc luckyimport.Document, dryRun bool) (*luckyimport.Report, error) {
	raw, ok := doc["connections"]
	if !ok {
		return nil, nil
	}
	var src []luckyConn
	if err := json.Unmarshal(raw, &src); err != nil {
		return nil, err
	}
	rep := &luckyimport.Report{Module: ID, Title: m.Title(), Found: len(src)}
	var out []Connection
	for _, lc := range src {
		c := Connection{Name: lc.Name, Type: lc.Type, Remark: lc.Remark, Local: LocalConfig{Shell: lc.LocalConfig.Shell, WorkDir: lc.LocalConfig.WorkDir}}
		s := lc.SSHConfig
		c.SSH = SSHConfig{Host: s.Host, Port: s.Port, Username: s.Username, AuthType: s.AuthType, Password: s.Password,
			PrivateKey: s.PrivateKey, Passphrase: s.Passphrase, KeepAlive: s.KeepAlive}
		if lc.Type == "telnet" {
			rep.Warnings = append(rep.Warnings, lc.Name+": Telnet 暂不支持，已跳过")
			continue
		}
		if s.Proxy.Enabled {
			rep.Warnings = append(rep.Warnings, lc.Name+": SSH 代理暂不支持，将直接连接")
		}
		if err := c.Validate(); err != nil {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s: 跳过，%v", lc.Name, err))
			continue
		}
		out = append(out, c)
	}
	rep.Imported = len(out)
	if len(out) > 0 {
		rep.Warnings = append(rep.Warnings, "主机密钥将在首次连接时记录（首次信任），请在首次连接时核对指纹")
	}
	if dryRun {
		return rep, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	old := m.cfg.Connections
	for _, c := range out {
		c.Key = crud.NewKey()
		m.cfg.Connections = append(m.cfg.Connections, c)
	}
	if err := m.saveLocked(); err != nil {
		m.cfg.Connections = old
		return nil, err
	}
	return rep, nil
}
