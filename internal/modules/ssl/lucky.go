package ssl

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/wangwanjie/gately/internal/crud"
	"github.com/wangwanjie/gately/internal/dnsprovider"
	"github.com/wangwanjie/gately/internal/luckyimport"
)

// luckyCert 对应 lucky_ssl 配置中的证书（仅迁移需要的字段）。
type luckyCert struct {
	Enable              bool
	AddFrom             string
	Remark              string
	CertBase64          string
	KeyBase64           string
	IssuerCertificate   string
	MappingToPath       bool
	MappingPath         string
	MappingChangeScript string
	ExtParams           map[string]any
}

// decodePEM 解码 Lucky 存储的证书内容（base64 编码的 PEM，部分字段可能直接是 PEM）。
func decodePEM(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, "-----BEGIN") {
		return s
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b)) + "\n"
}

func extStr(m map[string]any, k string) string {
	v, _ := m[k].(string)
	return strings.TrimSpace(v)
}

func extBool(m map[string]any, k string) bool {
	v, _ := m[k].(bool)
	return v
}

// caFromURL 把目录地址还原成预置 CA 名称。
func caFromURL(u string) string {
	for name, dir := range caDirectories {
		if strings.EqualFold(u, dir) {
			return name
		}
	}
	if u == "" {
		return "letsencrypt"
	}
	return "custom"
}

func convertLuckyCerts(raw json.RawMessage, settings Settings) ([]Cert, []string, int, error) {
	var src []luckyCert
	if err := json.Unmarshal(raw, &src); err != nil {
		return nil, nil, 0, err
	}
	var (
		out   []Cert
		warns []string
		// 启用了映射的目录，汇总提示用户确认挂载
		mapped []string
	)
	for i, lc := range src {
		name := lc.Remark
		if name == "" {
			name = fmt.Sprintf("第 %d 张证书", i+1)
		}
		warn := func(msg string) { warns = append(warns, name+": "+msg) }
		c := Cert{
			Remark:    lc.Remark,
			Enable:    lc.Enable,
			CertPEM:   decodePEM(lc.CertBase64),
			KeyPEM:    decodePEM(lc.KeyBase64),
			IssuerPEM: decodePEM(lc.IssuerCertificate),
			Mapping:   Mapping{Enable: lc.MappingToPath, Dir: lc.MappingPath, Script: lc.MappingChangeScript},
			UpdatedAt: time.Now(),
		}
		switch lc.AddFrom {
		case "acme":
			e := lc.ExtParams
			dir := extStr(e, "acmeCADirURL")
			c.Source = SourceACME
			c.ACME = ACMEConfig{
				Email:                     extStr(e, "acmeEmail"),
				CA:                        caFromURL(dir),
				EABKid:                    extStr(e, "acmeKid"),
				EABHMAC:                   extStr(e, "acmeHMAC"),
				KeyType:                   extStr(e, "acmeKeyType"),
				Provider:                  extStr(e, "acmeDNSServer"),
				Credentials:               dnsprovider.Credentials{ID: extStr(e, "acmeDNSID"), Secret: extStr(e, "acmeDNSSecret")},
				SkipPropagationCheck:      extBool(e, "acmeIgnorePropagationChecErrors"),
				DisableAuthoritativeCheck: extBool(e, "acmeDisableAuthoritativeNssPropagationRequirement"),
			}
			if c.ACME.CA == "custom" {
				c.ACME.DirectoryURL = dir
			}
			if v, ok := e["acmePropagationTimeout"].(float64); ok {
				c.ACME.PropagationTimeout = int(v)
			}
			if ds, ok := e["acmeDomains"].([]any); ok {
				for _, d := range ds {
					if s, ok := d.(string); ok {
						c.ACME.Domains = append(c.ACME.Domains, s)
					}
				}
			}
			if ns := extStr(e, "acmeRecursiveNameservers"); ns != "" {
				c.ACME.Resolvers = strings.FieldsFunc(ns, func(r rune) bool { return r == ',' || r == '\n' || r == ' ' })
			}
		case "upload", "file", "":
			c.Source = SourceUpload
		default:
			// 路径映射等来源按上传处理：保留证书内容，但不再自动刷新
			c.Source = SourceUpload
			warn(fmt.Sprintf("来源 %q 按上传证书导入，后续不会自动更新", lc.AddFrom))
		}
		if c.Mapping.Enable {
			mapped = append(mapped, c.Mapping.Dir)
		}
		if err := c.Validate(settings); err != nil {
			warn("跳过，" + err.Error())
			continue
		}
		out = append(out, c)
	}
	if len(mapped) > 0 {
		warns = append(warns, fmt.Sprintf("%d 张证书映射到目录（%s），这些路径来自 Lucky 的运行环境，Docker 部署时请保持相同的挂载", len(mapped), strings.Join(mapped, "、")))
	}
	return out, warns, len(src), nil
}

// ImportLucky 实现 luckyimport.Importer。
func (m *Module) ImportLucky(doc luckyimport.Document, opts luckyimport.Options) (*luckyimport.Report, error) {
	raw, ok := doc["SSLCertficateList"]
	if !ok {
		return nil, nil
	}
	m.mu.Lock()
	settings := m.cfg.Settings
	m.mu.Unlock()
	certs, warns, found, err := convertLuckyCerts(raw, settings)
	if err != nil {
		return nil, err
	}
	rep := &luckyimport.Report{Module: ID, Title: m.Title(), Found: found, Warnings: warns}
	m.mu.Lock()
	defer m.mu.Unlock()
	existing := map[string]bool{}
	for i := range m.cfg.Certs {
		existing[label(&m.cfg.Certs[i])] = true
	}
	var fresh []Cert
	for _, c := range certs {
		if name := label(&c); name != "" && existing[name] {
			rep.Warnings = append(rep.Warnings, luckyimport.Skipped(name))
			continue
		}
		if !opts.KeepEnabled {
			c.Enable = false
		}
		fresh = append(fresh, c)
	}
	rep.Imported = len(fresh)
	if !opts.KeepEnabled && rep.Imported > 0 {
		rep.Warnings = append(rep.Warnings, "导入的证书默认停用：启用后会把证书写入映射目录并执行脚本，请确认路径后再启用")
	}
	if opts.DryRun {
		return rep, nil
	}
	old := m.cfg.Certs
	for _, c := range fresh {
		c.Key = crud.NewKey()
		m.cfg.Certs = append(m.cfg.Certs, c)
	}
	if err := m.saveLocked(); err != nil {
		m.cfg.Certs = old
		return nil, err
	}
	// 已签发的证书原样使用：只刷新 TLS 证书表，不重新执行映射脚本，到期前也不会重新申请
	m.rebuildTLSLocked()
	return rep, nil
}
