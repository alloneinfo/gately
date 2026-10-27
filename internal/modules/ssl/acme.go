package ssl

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"

	"github.com/wangwanjie/gately/internal/acmehttp"
	"github.com/wangwanjie/gately/internal/dnsprovider"
	"github.com/wangwanjie/gately/internal/logbuf"
)

// acmeUser 实现 lego 的 registration.User。
type acmeUser struct {
	email string
	key   crypto.PrivateKey
	reg   *registration.Resource
}

func (u *acmeUser) GetEmail() string                        { return u.email }
func (u *acmeUser) GetRegistration() *registration.Resource { return u.reg }
func (u *acmeUser) GetPrivateKey() crypto.PrivateKey        { return u.key }

func accountKey(dirURL, email string) string {
	h := sha256.Sum256([]byte(dirURL + "|" + strings.ToLower(email)))
	return hex.EncodeToString(h[:8])
}

// dnsSolver 把 DNS-01 验证落到 Gately 的 DNS 服务商实现上。
type dnsSolver struct {
	txt     dnsprovider.TXTProvider
	zone    string
	timeout time.Duration
	log     *logbuf.Logger
}

// locate 计算验证记录所在的主域名与主机记录。
func (s *dnsSolver) locate(domain, keyAuth string) (zone, sub, value string, err error) {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	fqdn := strings.TrimSuffix(info.EffectiveFQDN, ".")
	if s.zone != "" {
		sub, ok := dnsprovider.SubForZone(fqdn, s.zone)
		if !ok {
			return "", "", "", fmt.Errorf("验证记录 %s 不属于主域名 %s", fqdn, s.zone)
		}
		return s.zone, sub, info.Value, nil
	}
	// 未指定主域名时先按 SOA 查询验证记录所在的权威区域，能正确处理单独托管的子域名（NS 委派）与多级后缀；
	// 查询失败（例如域名尚未生效）时退回按域名段数推断
	if z, err := findZone(fqdn); err == nil {
		if sub, ok := dnsprovider.SubForZone(fqdn, z); ok && sub != "@" {
			return z, sub, info.Value, nil
		}
	} else if s.log != nil {
		s.log.Warnf("查询 %s 的权威区域失败，按域名推断主域名: %v", fqdn, err)
	}
	sub, zone = dnsprovider.SplitFQDN(fqdn)
	return zone, sub, info.Value, nil
}

// findZone 返回 fqdn 所在的权威区域（不带末尾的点），测试中可替换。
// lego 会缓存 SOA 结果，同一次签发的添加与清理不会重复查询；查询使用设置的 DNS 服务器
var findZone = func(fqdn string) (string, error) {
	z, err := dns01.FindZoneByFqdn(dns01.ToFqdn(fqdn))
	if err != nil {
		return "", err
	}
	return strings.ToLower(strings.TrimSuffix(z, ".")), nil
}

func (s *dnsSolver) Present(domain, _, keyAuth string) error {
	zone, sub, value, err := s.locate(domain, keyAuth)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	s.log.Infof("添加验证记录 TXT %s.%s", sub, zone)
	return s.txt.AddTXT(ctx, zone, sub, value)
}

func (s *dnsSolver) CleanUp(domain, _, keyAuth string) error {
	zone, sub, value, err := s.locate(domain, keyAuth)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := s.txt.DeleteTXT(ctx, zone, sub, value); err != nil {
		s.log.Warnf("清理验证记录失败（可手动删除 TXT %s.%s）: %v", sub, zone, err)
	}
	return nil
}

// Timeout 实现 challenge.ProviderTimeout：等待 TXT 生效的超时与轮询间隔。
func (s *dnsSolver) Timeout() (time.Duration, time.Duration) {
	return s.timeout, 5 * time.Second
}

// obtainResult 是签发结果。
type obtainResult struct {
	CertPEM, KeyPEM, IssuerPEM string
	Account                    *Account
}

// acmeHTTPClient 允许测试注入信任测试 CA 的客户端。
var acmeHTTPClient func() *http.Client

// skipPropagationWait 是跳过生效检查时的固定等待时间。
var skipPropagationWait = 20 * time.Second

// obtain 使用 ACME 签发证书。acc 为已缓存账户，可为 nil；新注册的账户在结果中返回。
func obtain(cfg ACMEConfig, acc *Account, log *logbuf.Logger) (*obtainResult, error) {
	dirURL := cfg.directoryURL()
	user := &acmeUser{email: cfg.Email}
	res := &obtainResult{}
	if acc != nil && acc.KeyPEM != "" && acc.Registration != "" {
		key, err := certcrypto.ParsePEMPrivateKey([]byte(acc.KeyPEM))
		if err == nil {
			var reg registration.Resource
			if json.Unmarshal([]byte(acc.Registration), &reg) == nil {
				user.key, user.reg = key, &reg
			}
		}
	}
	if user.key == nil {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}
		user.key = key
	}

	lc := lego.NewConfig(user)
	lc.CADirURL = dirURL
	lc.Certificate.KeyType = certcrypto.KeyType(cfg.KeyType)
	// IP 地址不能作为 CommonName（Let's Encrypt 的 IP 证书不带 CN），含 IP 时只用 SAN
	lc.Certificate.DisableCommonName = hasIPIdentifier(cfg.Domains)
	lc.UserAgent = "Gately"
	if acmeHTTPClient != nil {
		lc.HTTPClient = acmeHTTPClient()
	}
	client, err := lego.NewClient(lc)
	if err != nil {
		return nil, fmt.Errorf("连接 ACME 服务失败: %w", err)
	}

	if user.reg == nil {
		log.Infof("注册 ACME 账户 %s", cfg.Email)
		var reg *registration.Resource
		if cfg.EABKid != "" {
			reg, err = client.Registration.RegisterWithExternalAccountBinding(registration.RegisterEABOptions{
				TermsOfServiceAgreed: true, Kid: cfg.EABKid, HmacEncoded: cfg.EABHMAC,
			})
		} else {
			reg, err = client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		}
		if err != nil {
			return nil, fmt.Errorf("注册 ACME 账户失败: %w", err)
		}
		user.reg = reg
		keyDER, err := x509.MarshalPKCS8PrivateKey(user.key)
		if err != nil {
			return nil, err
		}
		regJSON, _ := json.Marshal(reg)
		res.Account = &Account{
			Email:        cfg.Email,
			DirectoryURL: dirURL,
			KeyPEM:       string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})),
			Registration: string(regJSON),
		}
	}

	if cfg.Challenge == ChallengeHTTP {
		// HTTP-01：由 Web 服务模块或临时监听应答，无需 DNS 服务商
		prov := &acmehttp.Provider{Port: cfg.HTTPPort, Logf: log.Infof}
		if err := client.Challenge.SetHTTP01Provider(prov); err != nil {
			return nil, err
		}
	} else if err := setupDNS01(client, cfg, log); err != nil {
		return nil, err
	}

	if cfg.Profile != "" {
		log.Infof("申请证书 %s（CA: %s，证书配置: %s）", strings.Join(cfg.Domains, ", "), dirURL, cfg.Profile)
	} else {
		log.Infof("申请证书 %s（CA: %s）", strings.Join(cfg.Domains, ", "), dirURL)
	}
	cert, err := client.Certificate.Obtain(certificate.ObtainRequest{Domains: cfg.Domains, Bundle: true, Profile: cfg.Profile})
	if err != nil {
		return nil, fmt.Errorf("签发失败: %w", err)
	}
	if len(cert.Certificate) == 0 || len(cert.PrivateKey) == 0 {
		return nil, errors.New("CA 返回的证书为空")
	}
	res.CertPEM = string(cert.Certificate)
	res.KeyPEM = string(cert.PrivateKey)
	res.IssuerPEM = string(cert.IssuerCertificate)
	return res, nil
}

// setupDNS01 为客户端配置 DNS-01 验证。
func setupDNS01(client *lego.Client, cfg ACMEConfig, log *logbuf.Logger) error {
	p, err := dnsprovider.New(cfg.Provider, cfg.Credentials, &http.Client{Timeout: 30 * time.Second})
	if err != nil {
		return err
	}
	txt, err := dnsprovider.AsTXT(p)
	if err != nil {
		return err
	}
	solver := &dnsSolver{txt: txt, zone: strings.Trim(cfg.Zone, "."), timeout: time.Duration(cfg.PropagationTimeout) * time.Second, log: log}
	opts := []dns01.ChallengeOption{}
	if len(cfg.Resolvers) > 0 {
		opts = append(opts, dns01.AddRecursiveNameservers(dns01.ParseNameservers(cfg.Resolvers)))
	}
	if cfg.DisableAuthoritativeCheck {
		opts = append(opts, dns01.DisableAuthoritativeNssPropagationRequirement())
	}
	if cfg.SkipPropagationCheck {
		// 跳过本地检查，留出固定时间让 TXT 生效后再交给 CA 验证
		opts = append(opts, dns01.PropagationWait(skipPropagationWait, true))
	}
	return client.Challenge.SetDNS01Provider(solver, opts...)
}
