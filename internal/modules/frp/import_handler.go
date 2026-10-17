package frp

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/wangwanjie/gately/internal/crud"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/module"
)

// 代理导入方式。
const (
	// ProxyAppend 只追加名称不存在的代理，同名代理保持不变
	ProxyAppend = "append"
	// ProxyReplace 用导入的代理替换现有全部代理
	ProxyReplace = "replace"
)

// ImportRequest 是导入请求。Apply 为 false 时只返回解析预览，不保存。
type ImportRequest struct {
	// Role 为 client（frpc）或 server（frps）
	Role    string `json:"role"`
	Content string `json:"content"`
	Apply   bool   `json:"apply"`
	// ProxyMode 为 append 或 replace，仅客户端有效，留空等同 append
	ProxyMode string `json:"proxyMode"`
}

// ImportResponse 是导入响应，密钥均已脱敏。
type ImportResponse struct {
	Applied  bool            `json:"applied"`
	Role     string          `json:"role"`
	Client   *ClientSettings `json:"client,omitempty"`
	Server   *ServerSettings `json:"server,omitempty"`
	Proxies  []Proxy         `json:"proxies"`
	Warnings []string        `json:"warnings"`
	// Added 为实际新增的代理数量
	Added int `json:"added"`
}

func (m *Module) registerImport(r module.Router) {
	module.DocOf(r, "POST /import", module.Doc{
		Summary: "导入已有的 frpc / frps 配置（支持旧版 ini 与新版 toml）；apply 为 false 时仅预览",
		Req:     &ImportRequest{},
		Resp:    &ImportResponse{},
	})
	r.HandleFunc("POST /import", func(w http.ResponseWriter, req *http.Request) {
		var in ImportRequest
		if err := httpx.ReadJSON(req, &in); err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
		if in.ProxyMode != "" && in.ProxyMode != ProxyAppend && in.ProxyMode != ProxyReplace {
			httpx.Error(w, http.StatusBadRequest, "不支持的代理导入方式 %q", in.ProxyMode)
			return
		}
		res, err := ParseImport(in.Role, in.Content)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
		out := &ImportResponse{Role: res.Role, Proxies: res.Proxies, Warnings: res.Warnings}
		m.mu.Lock()
		defer m.mu.Unlock()
		if in.Apply {
			if err := m.applyImportLocked(res, in.ProxyMode, out); err != nil {
				httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
				return
			}
			out.Applied = true
		}
		if res.Client != nil {
			c := *res.Client
			c.Token = crud.Mask(c.Token)
			out.Client = &c
		}
		if res.Server != nil {
			s := *res.Server
			s.Token, s.DashboardPwd = crud.Mask(s.Token), crud.Mask(s.DashboardPwd)
			out.Server = &s
		}
		masked := make([]Proxy, len(out.Proxies))
		for i, p := range out.Proxies {
			p.SecretKey = crud.Mask(p.SecretKey)
			masked[i] = p
		}
		out.Proxies = masked
		httpx.OK(w, out)
	})
}

// applyImportLocked 保存导入结果并重启对应进程；调用方持有 mu。
// 启用状态与二进制路径沿用现有设置，导入不会自动启动进程，避免与正在运行的同名连接冲突。
func (m *Module) applyImportLocked(res *ImportResult, mode string, out *ImportResponse) error {
	oldCfg := m.cfg
	switch res.Role {
	case RoleClient:
		c := *res.Client
		c.Enable, c.BinPath = m.cfg.Client.Enable, m.cfg.Client.BinPath
		m.cfg.Client = c
		if mode == ProxyReplace {
			m.cfg.Proxies = nil
		}
		exists := map[string]bool{}
		for _, p := range m.cfg.Proxies {
			exists[p.Name] = true
		}
		var skipped []string
		// 复制一份再追加，保存失败时 oldCfg 中的切片不受影响
		items := append([]Proxy(nil), m.cfg.Proxies...)
		for _, p := range res.Proxies {
			if exists[p.Name] {
				skipped = append(skipped, p.Name)
				continue
			}
			p.Key = crud.NewKey()
			items = append(items, p)
			out.Added++
		}
		m.cfg.Proxies = items
		if len(skipped) > 0 {
			out.Warnings = append(out.Warnings, fmt.Sprintf("已存在同名代理，保持原样：%s", strings.Join(skipped, "、")))
		}
	case RoleServer:
		s := *res.Server
		s.Enable, s.BinPath = m.cfg.Server.Enable, m.cfg.Server.BinPath
		m.cfg.Server = s
	}
	if err := m.saveLocked(); err != nil {
		m.cfg = oldCfg
		return err
	}
	if res.Role == RoleClient {
		m.applyClientLocked()
	} else {
		m.applyServerLocked()
	}
	return nil
}
