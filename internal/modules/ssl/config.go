package ssl

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/wangwanjie/gately/internal/dnsprovider"
)

// 证书来源
const (
	SourceACME   = "acme"
	SourceUpload = "upload"
	SourceFile   = "file"
)

// 预置 CA，自定义时使用 DirectoryURL
var caDirectories = map[string]string{
	"letsencrypt":         "https://acme-v02.api.letsencrypt.org/directory",
	"letsencrypt-staging": "https://acme-staging-v02.api.letsencrypt.org/directory",
	"zerossl":             "https://acme.zerossl.com/v2/DV90",
	"google":              "https://dv.acme-v02.api.pki.goog/directory",
}

var validKeyTypes = map[string]bool{"P256": true, "P384": true, "2048": true, "4096": true}

// ACMEConfig 是 ACME 自动签发参数，验证方式为 DNS-01。
type ACMEConfig struct {
	Domains []string `json:"domains"`
	Email   string   `json:"email"`
	// CA 为预置名称或 custom
	CA           string `json:"ca"`
	DirectoryURL string `json:"directoryUrl"`
	// EAB 用于 ZeroSSL、Google 等需要外部账户绑定的 CA
	EABKid  string `json:"eabKid"`
	EABHMAC string `json:"eabHmac"`
	KeyType string `json:"keyType"`
	// Provider 与 Credentials 为 DNS 服务商，用于写入 _acme-challenge TXT 记录
	Provider    string                  `json:"provider"`
	Credentials dnsprovider.Credentials `json:"credentials"`
	// Zone 显式指定主域名，留空自动推断
	Zone string `json:"zone"`
	// Resolvers 为检查 TXT 生效所用的递归 DNS，留空使用系统配置
	Resolvers []string `json:"resolvers"`
	// PropagationTimeout 为等待 TXT 生效的最长时间（秒）
	PropagationTimeout int `json:"propagationTimeout"`
	// SkipPropagationCheck 跳过本地生效检查，等待固定时间后直接通知 CA 验证
	SkipPropagationCheck bool `json:"skipPropagationCheck"`
	// DisableAuthoritativeCheck 只通过递归 DNS 检查 TXT 是否生效，不查询权威服务器
	DisableAuthoritativeCheck bool `json:"disableAuthoritativeCheck"`
}

// Mapping 把证书写到目录供其他服务使用，文件名为 <备注>.crt/.key/.pem/_issuerCertificate.crt（与 Lucky 一致）。
type Mapping struct {
	Enable bool   `json:"enable"`
	Dir    string `json:"dir"`
	// Script 在证书文件变化后执行，例如重载 nginx
	Script string `json:"script"`
}

// Cert 是一张证书。
type Cert struct {
	Key    string `json:"key"`
	Remark string `json:"remark"`
	Enable bool   `json:"enable"`
	Source string `json:"source"`

	ACME ACMEConfig `json:"acme"`
	// CertFile / KeyFile 为文件来源的路径，定期重新读取
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`

	// 证书内容（PEM）。CertPEM 为包含中间证书的完整链
	CertPEM   string `json:"certPem"`
	KeyPEM    string `json:"keyPem"`
	IssuerPEM string `json:"issuerPem"`

	Mapping Mapping `json:"mapping"`
	// UpdatedAt 为证书内容最近一次变化时间
	UpdatedAt time.Time `json:"updatedAt,omitzero"`
}

// Account 是缓存的 ACME 账户，按 CA 地址与邮箱区分。
type Account struct {
	Email        string `json:"email"`
	DirectoryURL string `json:"directoryUrl"`
	KeyPEM       string `json:"keyPem"`
	// Registration 为 lego registration.Resource 的 JSON
	Registration string `json:"registration"`
}

// Settings 是模块级配置。
type Settings struct {
	// Enable 为自动续签开关，关闭后不再自动签发（已有证书仍可使用）
	Enable bool `json:"enable"`
	// RenewBeforeDays 为到期前多少天续签
	RenewBeforeDays int `json:"renewBeforeDays"`
	// ShortLivedRenewHours 为短期证书（有效期不足 10 天）到期前多少小时续签
	ShortLivedRenewHours int    `json:"shortLivedRenewHours"`
	DefaultEmail         string `json:"defaultEmail"`
}

type fileConfig struct {
	Settings Settings           `json:"settings"`
	Certs    []Cert             `json:"certs"`
	Accounts map[string]Account `json:"accounts,omitempty"`
}

func defaultSettings() Settings {
	return Settings{Enable: true, RenewBeforeDays: 30, ShortLivedRenewHours: 24}
}

func (s *Settings) normalize() {
	if s.RenewBeforeDays <= 0 {
		s.RenewBeforeDays = 30
	}
	if s.ShortLivedRenewHours <= 0 {
		s.ShortLivedRenewHours = 24
	}
}

// 备注用作映射文件名，限制字符集防止路径穿越
var reRemark = regexp.MustCompile(`^[A-Za-z0-9*_.@-]+$`)

// fileBase 返回映射文件名前缀，通配符替换为下划线。
func (c *Cert) fileBase() string {
	return strings.ReplaceAll(c.Remark, "*", "_")
}

// directoryURL 返回 ACME 目录地址。
func (a *ACMEConfig) directoryURL() string {
	if a.CA == "custom" || a.CA == "" && a.DirectoryURL != "" {
		return a.DirectoryURL
	}
	if u, ok := caDirectories[a.CA]; ok {
		return u
	}
	return caDirectories["letsencrypt"]
}

// Validate 校验配置。上传来源要求证书与私钥匹配。
func (c *Cert) Validate(settings Settings) error {
	c.Remark = strings.TrimSpace(c.Remark)
	if c.Mapping.Enable {
		if c.Remark == "" {
			return errors.New("启用证书映射时备注不能为空（用作文件名）")
		}
		c.Mapping.Dir = strings.TrimSpace(c.Mapping.Dir)
		if c.Mapping.Dir == "" || !filepath.IsAbs(c.Mapping.Dir) {
			return errors.New("映射目录需为绝对路径")
		}
	}
	if c.Remark != "" && !reRemark.MatchString(c.Remark) {
		return errors.New("备注只能包含字母、数字以及 . _ - * @")
	}
	switch c.Source {
	case SourceACME:
		return c.ACME.validate(settings)
	case SourceUpload:
		if c.CertPEM == "" || c.KeyPEM == "" {
			return errors.New("请上传证书与私钥")
		}
		if _, err := tls.X509KeyPair([]byte(c.CertPEM), []byte(c.KeyPEM)); err != nil {
			return fmt.Errorf("证书与私钥不匹配或格式错误: %w", err)
		}
	case SourceFile:
		if !filepath.IsAbs(c.CertFile) || !filepath.IsAbs(c.KeyFile) {
			return errors.New("证书与私钥路径需为绝对路径")
		}
	default:
		return fmt.Errorf("未知的证书来源 %q", c.Source)
	}
	return nil
}

func (a *ACMEConfig) validate(settings Settings) error {
	var domains []string
	seen := map[string]bool{}
	for _, d := range a.Domains {
		d = strings.ToLower(strings.Trim(strings.TrimSpace(d), "."))
		if d == "" || seen[d] {
			continue
		}
		if strings.Count(d, "*") > 1 || (strings.Contains(d, "*") && !strings.HasPrefix(d, "*.")) {
			return fmt.Errorf("域名 %q 格式错误，通配符只能出现在最左侧", d)
		}
		seen[d] = true
		domains = append(domains, d)
	}
	if len(domains) == 0 {
		return errors.New("至少填写一个域名")
	}
	a.Domains = domains
	a.Email = strings.TrimSpace(a.Email)
	if a.Email == "" {
		a.Email = settings.DefaultEmail
	}
	if a.Email == "" || !strings.Contains(a.Email, "@") {
		return errors.New("请填写有效的 ACME 账户邮箱")
	}
	if a.CA == "" {
		a.CA = "letsencrypt"
	}
	if a.CA == "custom" && !strings.HasPrefix(a.DirectoryURL, "https://") {
		return errors.New("自定义 CA 需填写 https 开头的目录地址")
	}
	if a.CA != "custom" {
		if _, ok := caDirectories[a.CA]; !ok {
			return fmt.Errorf("未知的 CA %q", a.CA)
		}
	}
	if (a.CA == "zerossl" || a.CA == "google") && (a.EABKid == "" || a.EABHMAC == "") {
		return errors.New("该 CA 需要填写 EAB Key ID 与 HMAC Key")
	}
	if a.KeyType == "" {
		a.KeyType = "P256"
	}
	if !validKeyTypes[a.KeyType] {
		return fmt.Errorf("不支持的密钥类型 %q", a.KeyType)
	}
	if !dnsprovider.SupportsTXT(a.Provider) {
		return fmt.Errorf("DNS 服务商 %q 不支持 ACME DNS 验证", a.Provider)
	}
	if a.PropagationTimeout <= 0 {
		a.PropagationTimeout = 300
	}
	return nil
}

// CertInfo 是从证书内容解析出的信息。
type CertInfo struct {
	Domains     []string  `json:"domains"`
	Issuer      string    `json:"issuer"`
	NotBefore   time.Time `json:"notBefore"`
	NotAfter    time.Time `json:"notAfter"`
	Fingerprint string    `json:"fingerprint"`
}

// parseCert 解析 PEM 证书链中的叶子证书。
func parseCert(certPEM string) (*x509.Certificate, error) {
	rest := []byte(certPEM)
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			return nil, errors.New("没有找到 PEM 证书")
		}
		if block.Type == "CERTIFICATE" {
			return x509.ParseCertificate(block.Bytes)
		}
	}
}

func certInfo(certPEM string) (*CertInfo, error) {
	leaf, err := parseCert(certPEM)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(leaf.Raw)
	domains := leaf.DNSNames
	if len(domains) == 0 && leaf.Subject.CommonName != "" {
		domains = []string{leaf.Subject.CommonName}
	}
	return &CertInfo{
		Domains:     domains,
		Issuer:      leaf.Issuer.CommonName,
		NotBefore:   leaf.NotBefore,
		NotAfter:    leaf.NotAfter,
		Fingerprint: hex.EncodeToString(sum[:]),
	}, nil
}
