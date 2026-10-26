package ssl

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// selfSignedIP 生成只含 IP SAN、不带 CommonName 的自签名证书（与 Let's Encrypt 的 IP 证书一致）。
func selfSignedIP(t *testing.T, ips []string, notBefore, notAfter time.Time) (string, string) {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		NotBefore:    notBefore,
		NotAfter:     notAfter,
	}
	for _, s := range ips {
		tmpl.IPAddresses = append(tmpl.IPAddresses, net.ParseIP(s))
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	kder, _ := x509.MarshalECPrivateKey(key)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kder}))
}

func TestValidateIPCert(t *testing.T) {
	ok := ACMEConfig{
		Domains: []string{"8.8.8.8", "[2001:4860:4860:0:0:0:0:8888]", "8.8.8.8", "example.com"},
		Email:   "a@example.com", Challenge: ChallengeHTTP, Profile: " shortlived ",
	}
	if err := ok.validate(Settings{}); err != nil {
		t.Fatal(err)
	}
	want := []string{"8.8.8.8", "2001:4860:4860::8888", "example.com"}
	if strings.Join(ok.Domains, ",") != strings.Join(want, ",") {
		t.Fatalf("IP 地址应规范化并去重: %v", ok.Domains)
	}
	if ok.Profile != ProfileShortLived {
		t.Fatalf("证书配置应去掉空白: %q", ok.Profile)
	}

	base := func(mod func(a *ACMEConfig)) ACMEConfig {
		a := ACMEConfig{Domains: []string{"8.8.8.8"}, Email: "a@example.com", Challenge: ChallengeHTTP, Profile: ProfileShortLived}
		mod(&a)
		return a
	}
	bad := map[string]ACMEConfig{
		"IP 使用 DNS 验证":        base(func(a *ACMEConfig) { a.Challenge = ChallengeDNS; a.Provider = "alidns" }),
		"IP 未选短期证书":           base(func(a *ACMEConfig) { a.Profile = "" }),
		"Let's Encrypt 内网 IP": base(func(a *ACMEConfig) { a.Domains = []string{"192.168.1.2"} }),
		"回环地址":                base(func(a *ACMEConfig) { a.Domains = []string{"127.0.0.1"} }),
		"CGNAT 地址":            base(func(a *ACMEConfig) { a.Domains = []string{"100.100.1.1"} }),
		"IPv6 ULA":            base(func(a *ACMEConfig) { a.Domains = []string{"fd00::1"} }),
		"ZeroSSL 不支持 IP":      base(func(a *ACMEConfig) { a.CA = "zerossl"; a.EABKid = "k"; a.EABHMAC = "h" }),
		"证书配置名非法":             base(func(a *ACMEConfig) { a.Domains = []string{"example.com"}; a.Profile = "short lived" }),
		"IPv6 带 zone":         base(func(a *ACMEConfig) { a.Domains = []string{"fe80::1%eth0"} }),
		"无法解析的 IPv6":          base(func(a *ACMEConfig) { a.Domains = []string{"2001:db8::zz"} }),
		"前导零 IPv4":            base(func(a *ACMEConfig) { a.Domains = []string{"08.8.8.8"} }),
		"不完整 IPv4":            base(func(a *ACMEConfig) { a.Domains = []string{"8.8.8"} }),
	}
	for name, a := range bad {
		if err := a.validate(Settings{}); err == nil {
			t.Errorf("%s: 应报错", name)
		}
	}

	// 自建 CA 可以为内网地址签发，也不强制短期证书
	custom := base(func(a *ACMEConfig) {
		a.CA, a.DirectoryURL, a.Profile = "custom", "https://ca.lan/acme/directory", ""
		a.Domains = []string{"192.168.1.2"}
	})
	if err := custom.validate(Settings{}); err != nil {
		t.Fatalf("自建 CA 不应限制内网 IP: %v", err)
	}
	// 普通域名证书可以单独选择短期证书配置，使用 DNS 验证
	dns := ACMEConfig{Domains: []string{"example.com"}, Email: "a@example.com", Provider: "alidns", Profile: ProfileShortLived}
	if err := dns.validate(Settings{}); err != nil {
		t.Fatal(err)
	}
	if !hasIPIdentifier([]string{"a.com", "::1"}) || hasIPIdentifier([]string{"a.com"}) {
		t.Fatal("hasIPIdentifier 判断错误")
	}
}

func TestCertInfoIP(t *testing.T) {
	now := time.Now()
	c, _ := selfSignedIP(t, []string{"8.8.8.8", "2001:4860:4860::8888"}, now, now.Add(time.Hour))
	info, err := certInfo(c)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(info.Domains, ",") != "8.8.8.8,2001:4860:4860::8888" {
		t.Fatalf("应解析出 IP SAN: %v", info.Domains)
	}
	// 证书中的 IP 与配置一致时不应因“域名列表变化”重签
	if !sameDomains(info.Domains, []string{"2001:4860:4860::8888", "8.8.8.8"}) {
		t.Fatal("IP 列表比较错误")
	}
}

func TestNeedsRenewalProfile(t *testing.T) {
	m := newTestModule(t)
	now := time.Now()
	long, longKey := selfSigned(t, []string{"a.com"}, now.Add(-time.Hour), now.Add(80*24*time.Hour))
	c := &Cert{Source: SourceACME, CertPEM: long, KeyPEM: longKey, ACME: ACMEConfig{Domains: []string{"a.com"}}}
	if need, _ := m.needsRenewal(c, now); need {
		t.Fatal("配置未变时不应重签")
	}
	c.ACME.Profile = ProfileShortLived
	if need, reason := m.needsRenewal(c, now); !need || reason != "证书配置已变化" {
		t.Fatalf("改为短期证书后应重签: %v %s", need, reason)
	}
	// 签发后记录了证书配置，不再重复签发
	c.IssuedProfile = ProfileShortLived
	if need, _ := m.needsRenewal(c, now); need {
		t.Fatal("已按当前配置签发时不应重签")
	}
	// 改回默认配置同样需要重签
	c.ACME.Profile = ""
	if need, _ := m.needsRenewal(c, now); !need {
		t.Fatal("改回默认证书配置后应重签")
	}

	// IP 短期证书：剩 3 天不续，剩 10 小时续签
	short, shortKey := selfSignedIP(t, []string{"8.8.8.8"}, now.Add(-3*24*time.Hour), now.Add(3*24*time.Hour))
	ipc := &Cert{Source: SourceACME, CertPEM: short, KeyPEM: shortKey, IssuedProfile: ProfileShortLived,
		ACME: ACMEConfig{Domains: []string{"8.8.8.8"}, Profile: ProfileShortLived}}
	if need, reason := m.needsRenewal(ipc, now); need {
		t.Fatalf("短期证书剩 3 天不应续签: %s", reason)
	}
	if need, _ := m.needsRenewal(ipc, now.Add(62*time.Hour)); !need {
		t.Fatal("短期证书剩 10 小时应续签")
	}
}

func TestEditKeepsIssuedProfile(t *testing.T) {
	m := newTestModule(t)
	now := time.Now()
	c, k := selfSignedIP(t, []string{"8.8.8.8"}, now, now.Add(6*24*time.Hour))
	m.cfg.Certs = []Cert{{Key: "ip", Enable: false, Source: SourceACME, CertPEM: c, KeyPEM: k, IssuedProfile: ProfileShortLived,
		ACME: ACMEConfig{Domains: []string{"8.8.8.8"}, Email: "a@example.com", Challenge: ChallengeHTTP, Profile: ProfileShortLived}}}
	srv := newTestServer(t, m)
	// 前端提交的配置不含签发记录，编辑时应保留
	body := `{"key":"ip","remark":"ip","enable":false,"source":"acme","acme":{"domains":["8.8.8.8"],"email":"a@example.com","challenge":"http","profile":"shortlived"}}`
	if code, out := srv.do("PUT", "/certs/ip", body); code != 200 {
		t.Fatalf("保存失败 %d: %s", code, out)
	}
	if got := m.cfg.Certs[0]; got.IssuedProfile != ProfileShortLived || got.CertPEM != c || got.Remark != "ip" {
		t.Fatalf("编辑后应保留已签发的证书与证书配置: %+v", got)
	}
}

func TestGetCertificateIP(t *testing.T) {
	m := newTestModule(t)
	now := time.Now()
	d, dk := selfSigned(t, []string{"example.com"}, now, now.Add(24*time.Hour))
	v4, v4k := selfSignedIP(t, []string{"8.8.8.8"}, now, now.Add(24*time.Hour))
	v6, v6k := selfSignedIP(t, []string{"2001:db8::1"}, now, now.Add(24*time.Hour))
	m.cfg.Certs = []Cert{
		{Key: "d", Enable: true, Source: SourceUpload, CertPEM: d, KeyPEM: dk},
		{Key: "v4", Enable: true, Source: SourceUpload, CertPEM: v4, KeyPEM: v4k},
		{Key: "v6", Enable: true, Source: SourceUpload, CertPEM: v6, KeyPEM: v6k},
	}
	m.rebuildTLSLocked()
	pick := func(hello *tls.ClientHelloInfo) string {
		c, err := m.GetCertificate(hello)
		if err != nil {
			return ""
		}
		leaf, _ := x509.ParseCertificate(c.Certificate[0])
		if len(leaf.IPAddresses) > 0 {
			return leaf.IPAddresses[0].String()
		}
		return leaf.DNSNames[0]
	}
	// 没有 SNI 时按连接的本机地址选择
	if got := pick(&tls.ClientHelloInfo{Conn: fakeConn{local: "[2001:db8::1]:443"}}); got != "2001:db8::1" {
		t.Fatalf("应按本机 IPv6 地址选择证书，实际 %q", got)
	}
	if got := pick(&tls.ClientHelloInfo{Conn: fakeConn{local: "[::ffff:8.8.8.8]:443"}}); got != "8.8.8.8" {
		t.Fatalf("应识别 IPv4 映射地址，实际 %q", got)
	}
	// 经 NAT 时本机地址是内网 IP，使用第一张 IP 证书
	if got := pick(&tls.ClientHelloInfo{Conn: fakeConn{local: "192.168.1.10:443"}}); got != "8.8.8.8" {
		t.Fatalf("找不到本机地址时应使用第一张 IP 证书，实际 %q", got)
	}
	if got := pick(&tls.ClientHelloInfo{}); got != "8.8.8.8" {
		t.Fatalf("没有连接信息时应使用第一张 IP 证书，实际 %q", got)
	}
	// SNI 中带 IP 地址时按规范写法匹配
	if got := pick(&tls.ClientHelloInfo{ServerName: "2001:DB8:0::1"}); got != "2001:db8::1" {
		t.Fatalf("SNI 中的 IP 应匹配，实际 %q", got)
	}
	// 有 SNI 但不匹配的域名仍然报错，不退回 IP 证书
	if got := pick(&tls.ClientHelloInfo{ServerName: "other.com"}); got != "" {
		t.Fatalf("不匹配的域名不应选中 IP 证书，实际 %q", got)
	}
	if !m.HasCertificate("8.8.8.8") || !m.HasCertificate("example.com") {
		t.Fatal("HasCertificate 应识别 IP 与域名")
	}

	// 没有 IP 证书时，无 SNI 的连接仍然报错
	m.cfg.Certs = m.cfg.Certs[:1]
	m.rebuildTLSLocked()
	if got := pick(&tls.ClientHelloInfo{}); got != "" {
		t.Fatalf("没有 IP 证书时不应返回证书，实际 %q", got)
	}
}

// testServer 把模块接口挂到 ServeMux 上，便于按 HTTP 请求测试。
type testServer struct{ mux *http.ServeMux }

// HandleFunc 实现 module.Router。
func (s *testServer) HandleFunc(pattern string, h http.HandlerFunc) {
	s.mux.HandleFunc(pattern, h)
}

func newTestServer(t *testing.T, m *Module) *testServer {
	t.Helper()
	s := &testServer{mux: http.NewServeMux()}
	m.Routes(s)
	return s
}

func (s *testServer) do(method, path, body string) (int, string) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.mux.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

// fakeConn 只提供本机地址，用于模拟 TLS 握手中的连接。
type fakeConn struct {
	net.Conn
	local string
}

func (c fakeConn) LocalAddr() net.Addr {
	ap, err := net.ResolveTCPAddr("tcp", c.local)
	if err != nil {
		panic(err)
	}
	return ap
}

func TestShortLivedExpiryWarn(t *testing.T) {
	m := newTestModule(t)
	now := time.Now()
	c, k := selfSignedIP(t, []string{"8.8.8.8"}, now.Add(-3*24*time.Hour), now.Add(3*24*time.Hour))
	m.cfg.Certs = []Cert{{Key: "ip", Enable: true, Source: SourceACME, CertPEM: c, KeyPEM: k}}
	cert := &m.cfg.Certs[0]
	// 总有效期 6 天的证书剩 3 天属于正常状态，不应提醒
	m.warnExpiringLocked(cert, now)
	if !m.stateLocked("ip").lastExpiryWarn.IsZero() {
		t.Fatal("短期证书剩 3 天不应发出到期提醒")
	}
	// 剩余不到续签提前量（24 小时）的一半，说明续签失败，应提醒
	m.warnExpiringLocked(cert, now.Add(3*24*time.Hour-11*time.Hour))
	if m.stateLocked("ip").lastExpiryWarn.IsZero() {
		t.Fatal("短期证书剩 11 小时应发出到期提醒")
	}
	info, _ := certInfo(c)
	if msg := expiryMessage(info, 11*time.Hour, true); !strings.Contains(msg, "11 小时") {
		t.Fatalf("短期证书应按小时提示: %s", msg)
	}

	// 普通证书仍按 7 天提醒
	long, longKey := selfSigned(t, []string{"a.com"}, now.Add(-80*24*time.Hour), now.Add(10*24*time.Hour))
	m.cfg.Certs = append(m.cfg.Certs, Cert{Key: "long", Enable: true, Source: SourceACME, CertPEM: long, KeyPEM: longKey})
	m.warnExpiringLocked(&m.cfg.Certs[1], now)
	if !m.stateLocked("long").lastExpiryWarn.IsZero() {
		t.Fatal("普通证书剩 10 天不应提醒")
	}
	m.warnExpiringLocked(&m.cfg.Certs[1], now.Add(4*24*time.Hour))
	if m.stateLocked("long").lastExpiryWarn.IsZero() {
		t.Fatal("普通证书剩 6 天应提醒")
	}
}

func TestShortLivedRetryBackoff(t *testing.T) {
	m := newTestModule(t)
	// 指向不可达的 CA，签发会立即失败
	acme := func(profile string) ACMEConfig {
		return ACMEConfig{Domains: []string{"a.example.com"}, Email: "a@example.com", CA: "custom",
			DirectoryURL: "https://127.0.0.1:1/directory", Challenge: ChallengeHTTP, KeyType: "P256", Profile: profile}
	}
	m.cfg.Certs = []Cert{
		{Key: "short", Enable: true, Source: SourceACME, ACME: acme(ProfileShortLived)},
		{Key: "long", Enable: true, Source: SourceACME, ACME: acme("")},
	}
	for _, key := range []string{"short", "long"} {
		m.stateLocked(key).Obtaining = true
		m.obtainNow(key)
	}
	short := time.Until(m.stateLocked("short").NextRetry)
	long := time.Until(m.stateLocked("long").NextRetry)
	if m.stateLocked("short").LastError == "" || short > shortLivedBackoff || short < shortLivedBackoff-time.Minute {
		t.Fatalf("短期证书失败后应在 %v 后重试，实际 %v（%s）", shortLivedBackoff, short, m.stateLocked("short").LastError)
	}
	if long < failureBackoff-time.Minute {
		t.Fatalf("普通证书失败退避应保持 %v，实际 %v", failureBackoff, long)
	}
}

func TestConvertLuckyIPCert(t *testing.T) {
	now := time.Now()
	c, k := selfSignedIP(t, []string{"8.8.8.8"}, now, now.Add(6*24*time.Hour))
	raw, _ := json.Marshal([]map[string]any{
		{
			"Enable": true, "AddFrom": "acme", "Remark": "ip", "CertBase64": encodeB64(c), "KeyBase64": encodeB64(k),
			"ExtParams": map[string]any{
				"acmeCADirURL": "https://acme-v02.api.letsencrypt.org/directory", "acmeDomains": []string{"8.8.8.8"}, "acmeEmail": "a@b.c",
				"acmeDNSServer": "http", "acmeKeyType": "P256", "acmeProfile": "shortlived",
			},
		},
		{
			"Enable": true, "AddFrom": "acme", "Remark": "ip2",
			"ExtParams": map[string]any{
				"acmeCADirURL": "https://acme-v02.api.letsencrypt.org/directory", "acmeDomains": []string{"8.8.4.4"}, "acmeEmail": "a@b.c",
				"acmeDNSServer": "https", "acmeKeyType": "P256", "acmeProfile": "shortlived",
			},
		},
	})
	certs, warns, _, err := convertLuckyCerts(raw, defaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	if len(certs) != 2 {
		t.Fatalf("应导入 2 张 IP 证书: %v", warns)
	}
	got := certs[0]
	if got.ACME.Challenge != ChallengeHTTP || got.ACME.Provider != "" || got.ACME.Profile != ProfileShortLived || got.IssuedProfile != ProfileShortLived {
		t.Fatalf("Lucky 的 HTTP 验证与短期证书配置映射错误: %+v", got.ACME)
	}
	m := newTestModule(t)
	if need, reason := m.needsRenewal(&got, now); need {
		t.Fatalf("导入的有效 IP 证书不应立即重签: %s", reason)
	}
	// 尚未签发的证书不记录签发配置
	if certs[1].IssuedProfile != "" || !strings.Contains(strings.Join(warns, "\n"), "HTTPS") {
		t.Fatalf("HTTPS 验证应按 HTTP-01 导入并提示: %+v %v", certs[1], warns)
	}
}

// TestPebbleIPShortLived 用 Pebble 走一遍 IP 地址短期证书的 HTTP-01 签发，并按 IP 地址完成一次 TLS 握手。
// 需要设置 GATELY_PEBBLE_BIN 与 GATELY_PEBBLE_SRC，Pebble 需支持证书配置（profiles，v2.7 及以上）。
func TestPebbleIPShortLived(t *testing.T) {
	bin, src := os.Getenv("GATELY_PEBBLE_BIN"), os.Getenv("GATELY_PEBBLE_SRC")
	if bin == "" || src == "" {
		t.Skip("未设置 GATELY_PEBBLE_BIN / GATELY_PEBBLE_SRC")
	}
	acmePort, pebbleMgmt, httpPort := freePort(t), freePort(t), freePort(t)
	cfgPath := filepath.Join(t.TempDir(), "pebble.json")
	cfg := fmt.Sprintf(`{"pebble":{"listenAddress":"127.0.0.1:%d","managementListenAddress":"127.0.0.1:%d",
		"certificate":"%s/test/certs/localhost/cert.pem","privateKey":"%s/test/certs/localhost/key.pem",
		"httpPort":%d,"tlsPort":5001,"ocspResponderURL":"","externalAccountBindingRequired":false,
		"profiles":{"default":{"description":"default","validityPeriod":7776000},
		"shortlived":{"description":"short","validityPeriod":518400}}}}`, acmePort, pebbleMgmt, src, src, httpPort)
	_ = os.WriteFile(cfgPath, []byte(cfg), 0o600)
	pebble := exec.Command(filepath.Join(bin, "pebble"), "-config", cfgPath)
	pebble.Env = append(os.Environ(), "PEBBLE_VA_NOSLEEP=1", "PEBBLE_WFE_NONCEREJECT=0", "PEBBLE_AUTHZREUSE=0")
	if err := pebble.Start(); err != nil {
		t.Fatal(err)
	}
	defer pebble.Process.Kill()
	waitTCP(t, fmt.Sprintf("127.0.0.1:%d", acmePort))
	waitTCP(t, fmt.Sprintf("127.0.0.1:%d", pebbleMgmt))

	ca, err := os.ReadFile(filepath.Join(src, "test/certs/pebble.minica.pem"))
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(ca)
	httpsClient := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{Proxy: nil, TLSClientConfig: &tls.Config{RootCAs: pool}}}
	acmeHTTPClient = func() *http.Client { return httpsClient }
	defer func() { acmeHTTPClient = nil }()

	m := newTestModule(t)
	m.ctx = context.Background()
	// 自建 CA（custom）允许回环地址，便于本机验证
	c := Cert{
		Key: "ip", Remark: "ip", Enable: true, Source: SourceACME,
		ACME: ACMEConfig{
			Domains: []string{"127.0.0.1", "[::1]"}, Email: "test@gately.test", CA: "custom",
			DirectoryURL: fmt.Sprintf("https://127.0.0.1:%d/dir", acmePort),
			Challenge:    ChallengeHTTP, HTTPPort: httpPort, Profile: ProfileShortLived,
		},
	}
	if err := c.Validate(m.cfg.Settings); err != nil {
		t.Fatal(err)
	}
	m.cfg.Certs = []Cert{c}
	m.obtainNow("ip")
	if st := m.stateLocked("ip"); st.LastError != "" {
		t.Fatalf("签发失败: %s", st.LastError)
	}
	got := m.cfg.Certs[0]
	info, err := certInfo(got.CertPEM)
	if err != nil {
		t.Fatal(err)
	}
	if !sameDomains(info.Domains, []string{"127.0.0.1", "::1"}) {
		t.Fatalf("证书应包含两个 IP 地址: %v", info.Domains)
	}
	if life := info.NotAfter.Sub(info.NotBefore); life > 7*24*time.Hour {
		t.Fatalf("应签发短期证书，实际有效期 %v", life)
	}
	leaf, _ := parseCert(got.CertPEM)
	if leaf.Subject.CommonName != "" {
		t.Fatalf("IP 证书不应带 CommonName: %q", leaf.Subject.CommonName)
	}
	if got.IssuedProfile != ProfileShortLived {
		t.Fatalf("应记录签发时的证书配置: %q", got.IssuedProfile)
	}
	if need, reason := m.needsRenewal(&got, time.Now()); need {
		t.Fatalf("刚签发的短期证书不应续签: %s", reason)
	}

	// 用 Pebble 的根证书校验：客户端按 IP 地址连接（不发送 SNI），服务端应选中这张 IP 证书
	resp, err := httpsClient.Get(fmt.Sprintf("https://127.0.0.1:%d/roots/0", pebbleMgmt))
	if err != nil {
		t.Fatal(err)
	}
	rootPEM, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(rootPEM) {
		t.Fatalf("读取 Pebble 根证书失败: %s", rootPEM)
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{GetCertificate: m.GetCertificate})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.(*tls.Conn).Handshake()
			conn.Close()
		}
	}()
	conn, err := tls.Dial("tcp", ln.Addr().String(), &tls.Config{RootCAs: roots})
	if err != nil {
		t.Fatalf("按 IP 地址握手失败: %v", err)
	}
	conn.Close()

	// 改回默认证书配置后应重新签发，得到普通有效期的证书
	m.cfg.Certs[0].ACME.Profile = ""
	m.checkAll(true)
	info2, _ := certInfo(m.cfg.Certs[0].CertPEM)
	if info2.NotAfter.Sub(info2.NotBefore) < 30*24*time.Hour || m.cfg.Certs[0].IssuedProfile != "" {
		t.Fatalf("改回默认配置后应重签普通证书: %v %q（%s）", info2.NotAfter.Sub(info2.NotBefore), m.cfg.Certs[0].IssuedProfile, m.stateLocked("ip").LastError)
	}
}

// TestShortLivedRenewHoursCap 校验短期证书续签提前小时数被限制在有效期一半以内，避免每次检查都重签
func TestShortLivedRenewHoursCap(t *testing.T) {
	s := Settings{ShortLivedRenewHours: 168}
	s.normalize()
	if s.ShortLivedRenewHours != maxShortLivedRenewHours {
		t.Fatalf("提前小时数应限制为 %d，实际 %d", maxShortLivedRenewHours, s.ShortLivedRenewHours)
	}
	s = Settings{ShortLivedRenewHours: 48}
	s.normalize()
	if s.ShortLivedRenewHours != 48 {
		t.Fatalf("合理的提前小时数不应改变: %d", s.ShortLivedRenewHours)
	}
}

// TestHasCertificateEmptyName 校验空主机名不算有证书，即使存在可用于无 SNI 握手的 IP 证书
func TestHasCertificateEmptyName(t *testing.T) {
	m := newTestModule(t)
	now := time.Now()
	v4, v4k := selfSignedIP(t, []string{"8.8.8.8"}, now, now.Add(24*time.Hour))
	m.cfg.Certs = []Cert{{Key: "v4", Enable: true, Source: SourceUpload, CertPEM: v4, KeyPEM: v4k}}
	m.rebuildTLSLocked()
	if m.HasCertificate("") {
		t.Fatal("空主机名不应视为有证书")
	}
	if !m.HasCertificate("8.8.8.8") {
		t.Fatal("IP 主机名应有证书")
	}
	if _, err := m.GetCertificate(&tls.ClientHelloInfo{}); err != nil {
		t.Fatalf("无 SNI 握手仍应回退到 IP 证书: %v", err)
	}
}
