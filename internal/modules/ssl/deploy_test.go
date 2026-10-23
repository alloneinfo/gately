package ssl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeDeployers 记录部署调用，可让 SSH 推送返回指定错误。
type fakeDeployers struct {
	mu       sync.Mutex
	pushes   []string
	commands []string
	files    map[string]RemoteFile
	restarts [][]string
	pushErr  error
}

func (f *fakeDeployers) deployers() Deployers {
	return Deployers{
		Push: func(_ context.Context, conn, dir string, files []RemoteFile, command string) (string, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			if f.pushErr != nil {
				return "boom output", f.pushErr
			}
			f.pushes = append(f.pushes, conn+":"+dir)
			f.commands = append(f.commands, command)
			f.files = map[string]RemoteFile{}
			for _, x := range files {
				f.files[x.Name] = x
			}
			return "reloaded", nil
		},
		Restart: func(_ context.Context, names []string) error {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.restarts = append(f.restarts, names)
			return nil
		},
	}
}

func (f *fakeDeployers) count() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.pushes), len(f.restarts)
}

// waitDeploys 等待后台部署全部结束。
func waitDeploys(t *testing.T, m *Module) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		running := false
		for _, st := range m.states {
			for _, d := range st.Deploys {
				running = running || d.Running
			}
		}
		m.mu.Unlock()
		if !running {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("部署未在预期时间内结束")
}

func TestDeployValidate(t *testing.T) {
	cases := []struct {
		remark string
		d      Deploy
		ok     bool
	}{
		{"a.com", Deploy{Type: DeploySSH, Connection: "c", Dir: "/etc/ssl/a/"}, true},
		{"", Deploy{Type: DeploySSH, Connection: "c", Dir: "/etc/ssl"}, false},
		{"a.com", Deploy{Type: DeploySSH, Dir: "/etc/ssl"}, false},
		{"a.com", Deploy{Type: DeploySSH, Connection: "c", Dir: "etc/ssl"}, false},
		{"", Deploy{Type: DeployDocker, Containers: []string{" nginx ", ""}}, true},
		{"", Deploy{Type: DeployDocker, Containers: []string{" "}}, false},
		{"", Deploy{Type: "ftp"}, false},
		// 本机目录
		{"a.com", Deploy{Type: DeployLocal, Dir: "/etc/ssl/a/"}, true},
		{"", Deploy{Type: DeployLocal, Dir: "/etc/ssl"}, false},
		{"a.com", Deploy{Type: DeployLocal, Dir: "etc/ssl"}, false},
		// 自定义文件名：填写后备注可为空，需同时填写、合法且互不相同
		{"", Deploy{Type: DeployLocal, Dir: "/etc/ssl", CertName: "fullchain.pem", KeyName: "privkey.pem"}, true},
		{"", Deploy{Type: DeploySSH, Connection: "c", Dir: "/etc/ssl", CertName: "tls.crt", KeyName: "tls.key"}, true},
		{"", Deploy{Type: DeployLocal, Dir: "/etc/ssl", CertName: "fullchain.pem"}, false},
		{"", Deploy{Type: DeployLocal, Dir: "/etc/ssl", CertName: "../x.pem", KeyName: "k.pem"}, false},
		{"", Deploy{Type: DeployLocal, Dir: "/etc/ssl", CertName: "..", KeyName: "k.pem"}, false},
		{"", Deploy{Type: DeployLocal, Dir: "/etc/ssl", CertName: "a.pem", KeyName: "a.pem"}, false},
	}
	for i, c := range cases {
		err := c.d.validate(c.remark)
		if (err == nil) != c.ok {
			t.Errorf("用例 %d: err=%v", i, err)
		}
	}
	d := Deploy{Type: DeployDocker, Containers: []string{" nginx ", ""}, Dir: "/x", Command: "x"}
	_ = d.validate("")
	if len(d.Containers) != 1 || d.Containers[0] != "nginx" || d.Dir != "" || d.Command != "" {
		t.Fatalf("应整理容器名并清除无关字段: %+v", d)
	}
	d = Deploy{Type: DeployLocal, Dir: " /etc/ssl/a/ ", Connection: "c", Containers: []string{"x"}, CertName: " a.pem ", KeyName: "b.pem"}
	if err := d.validate(""); err != nil || d.Dir != "/etc/ssl/a" || d.Connection != "" || d.Containers != nil || d.CertName != "a.pem" {
		t.Fatalf("本机目录应整理路径与文件名并清除无关字段: %+v %v", d, err)
	}
}

// 已有部署目标的摘要应与旧版本一致，避免升级后全部重新部署
func TestDeployHashCompatible(t *testing.T) {
	c := &Cert{Remark: "a.com", CertPEM: "C", KeyPEM: "K"}
	d := &Deploy{Type: DeploySSH, Connection: "hk", Dir: "/etc/ssl", Command: "reload"}
	if got := deployHash(c, d); got != legacyDeployHash(c, d) {
		t.Fatalf("摘要与旧版本不一致: %s", got)
	}
	custom := *d
	custom.CertName, custom.KeyName = "a.pem", "b.pem"
	if deployHash(c, &custom) == deployHash(c, d) {
		t.Fatal("自定义文件名变化后应重新部署")
	}
}

// legacyDeployHash 是旧版本的摘要算法
func legacyDeployHash(c *Cert, d *Deploy) string {
	h := sha256.New()
	for _, s := range []string{c.CertPEM, c.KeyPEM, c.IssuerPEM, c.fileBase(), d.Type, d.Connection, d.Dir, d.Command, strings.Join(d.Containers, "\n")} {
		h.Write([]byte(s))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func TestDeployLocal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	m := newTestModule(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		m.wg.Wait()
	})
	m.ctx = ctx
	m.SetDeployers(Deployers{})
	now := time.Now()
	c, k := selfSigned(t, []string{"nas.example.com"}, now, now.Add(24*time.Hour))
	dirA := filepath.Join(t.TempDir(), "a")
	dirB := filepath.Join(t.TempDir(), "b")
	logFile := filepath.Join(t.TempDir(), "cmd.log")
	cert := Cert{
		Key: "a", Remark: "nas.example.com", Enable: true, Source: SourceUpload, CertPEM: c, KeyPEM: k,
		Deploys: []Deploy{
			// 默认文件名，每个目录各自执行命令
			{Enable: true, Type: DeployLocal, Dir: dirA, Command: `echo "A $certName $certFile $keyFile" >> ` + logFile},
			// 自定义文件名
			{Enable: true, Type: DeployLocal, Dir: dirB, CertName: "fullchain.pem", KeyName: "privkey.pem", Command: `echo "B $GATELY_CERTFILE $keyFile" >> ` + logFile},
		},
	}
	if err := cert.Validate(m.cfg.Settings); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	m.cfg.Certs = []Cert{cert}
	m.changedLocked("a")
	m.mu.Unlock()
	waitDeploys(t, m)

	for _, name := range []string{"nas.example.com.crt", "nas.example.com.pem", "nas.example.com.key"} {
		if _, err := os.Stat(filepath.Join(dirA, name)); err != nil {
			t.Fatalf("目录 A 缺少 %s: %v", name, err)
		}
	}
	entries, _ := os.ReadDir(dirB)
	if len(entries) != 2 {
		t.Fatalf("自定义文件名时只应写入 2 个文件: %v", entries)
	}
	key, err := os.Stat(filepath.Join(dirB, "privkey.pem"))
	if err != nil || key.Mode().Perm() != 0o600 {
		t.Fatalf("私钥权限应为 600: %v %v", key, err)
	}
	if data, _ := os.ReadFile(filepath.Join(dirB, "fullchain.pem")); string(data) != string(pemBytes(c)) {
		t.Fatal("证书内容错误")
	}
	readLog := func() string {
		data, _ := os.ReadFile(logFile)
		return string(data)
	}
	// 两个目标并发执行，只检查各自的输出行
	log := readLog()
	for _, line := range []string{
		"A nas.example.com " + filepath.Join(dirA, "nas.example.com.crt") + " " + filepath.Join(dirA, "nas.example.com.key") + "\n",
		"B " + filepath.Join(dirB, "fullchain.pem") + " " + filepath.Join(dirB, "privkey.pem") + "\n",
	} {
		if !strings.Contains(log, line) {
			t.Fatalf("命令输出缺少 %q:\n%s", line, log)
		}
	}

	// 内容未变化时不重复执行
	m.checkAll(false)
	waitDeploys(t, m)
	if n := strings.Count(readLog(), "\n"); n != 2 {
		t.Fatalf("未变化时不应重复部署: %d", n)
	}

	// 目标目录的文件被删除后，定时检查会重新复制
	if err := os.Remove(filepath.Join(dirB, "privkey.pem")); err != nil {
		t.Fatal(err)
	}
	m.checkAll(false)
	waitDeploys(t, m)
	if _, err := os.Stat(filepath.Join(dirB, "privkey.pem")); err != nil {
		t.Fatalf("文件被删除后应重新复制: %v", err)
	}
	if n := strings.Count(readLog(), "B "); n != 2 {
		t.Fatalf("重新复制后应再次执行该目录的命令: %d", n)
	}
	if n := strings.Count(readLog(), "A "); n != 1 {
		t.Fatalf("其他目录不应受影响: %d", n)
	}

	// 命令失败时记录错误并退避
	m.mu.Lock()
	m.cfg.Certs[0].Deploys[0].Command = "echo bad >&2; exit 3"
	m.changedLocked("a")
	m.mu.Unlock()
	waitDeploys(t, m)
	m.mu.Lock()
	st := m.stateLocked("a").Deploys[0]
	m.mu.Unlock()
	if !strings.Contains(st.LastError, "bad") || st.NextRetry.IsZero() {
		t.Fatalf("命令失败时应记录错误并退避: %+v", st)
	}
}

func TestDeployOnChange(t *testing.T) {
	m := newTestModule(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		m.wg.Wait()
	})
	m.ctx = ctx
	fake := &fakeDeployers{}
	m.SetDeployers(fake.deployers())
	now := time.Now()
	c, k := selfSigned(t, []string{"nas.example.com"}, now, now.Add(24*time.Hour))
	cert := Cert{
		Key: "a", Remark: "nas.example.com", Enable: true, Source: SourceUpload, CertPEM: c, KeyPEM: k,
		Deploys: []Deploy{
			{Enable: true, Type: DeploySSH, Connection: "hk", Dir: "/etc/ocserv/certs", Command: "systemctl reload ocserv"},
			{Enable: true, Type: DeployDocker, Containers: []string{"anylink"}},
			{Enable: false, Type: DeployDocker, Containers: []string{"off"}},
		},
	}
	if err := cert.Validate(m.cfg.Settings); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	m.cfg.Certs = []Cert{cert}
	m.changedLocked("a")
	m.mu.Unlock()
	waitDeploys(t, m)
	if p, r := fake.count(); p != 1 || r != 1 {
		t.Fatalf("应各部署一次: push=%d restart=%d", p, r)
	}
	if fake.pushes[0] != "hk:/etc/ocserv/certs" || fake.restarts[0][0] != "anylink" {
		t.Fatalf("部署目标错误: %v %v", fake.pushes, fake.restarts)
	}
	want := "certName='nas.example.com'; certDir='/etc/ocserv/certs'; certFile='/etc/ocserv/certs/nas.example.com.crt'; keyFile='/etc/ocserv/certs/nas.example.com.key'; systemctl reload ocserv"
	if fake.commands[0] != want {
		t.Fatalf("远程命令错误:\n%s\n%s", fake.commands[0], want)
	}
	if f := fake.files["nas.example.com.key"]; f.Mode != 0o600 || string(f.Data) != k+"\n" && string(f.Data) != k {
		t.Fatalf("私钥文件错误: %+v", f.Mode)
	}
	if len(fake.files) != 3 {
		t.Fatalf("没有中间证书时应上传 3 个文件: %d", len(fake.files))
	}
	m.mu.Lock()
	hash := m.cfg.Certs[0].Deploys[0].DeployedHash
	m.mu.Unlock()
	if hash == "" {
		t.Fatal("部署成功后应记录摘要")
	}

	// 内容未变化时不重复部署
	m.mu.Lock()
	m.changedLocked("a")
	m.mu.Unlock()
	waitDeploys(t, m)
	if p, r := fake.count(); p != 1 || r != 1 {
		t.Fatalf("内容未变化不应重复部署: push=%d restart=%d", p, r)
	}

	// 编辑时保留部署记录；目标设置变化后重新部署
	old := m.cfg.Certs[0]
	updated := old
	updated.Deploys = append([]Deploy(nil), old.Deploys...)
	updated.Deploys[0].DeployedHash = ""
	updated.Deploys[1].Containers = []string{"anylink", "ocserv"}
	keepDeployRecords(&old, &updated)
	if updated.Deploys[0].DeployedHash != hash || updated.Deploys[1].DeployedHash != "" {
		t.Fatal("应只保留未变化目标的部署记录")
	}

	// 证书内容变化后全部重新部署；失败时进入退避并记录错误
	c2, k2 := selfSigned(t, []string{"nas.example.com"}, now, now.Add(48*time.Hour))
	fake.mu.Lock()
	fake.pushErr = errors.New("connection refused")
	fake.mu.Unlock()
	m.mu.Lock()
	m.cfg.Certs[0].CertPEM, m.cfg.Certs[0].KeyPEM = c2, k2
	m.changedLocked("a")
	m.mu.Unlock()
	waitDeploys(t, m)
	m.mu.Lock()
	st := m.stateLocked("a").Deploys[0]
	m.mu.Unlock()
	if !strings.Contains(st.LastError, "connection refused") || st.NextRetry.IsZero() {
		t.Fatalf("失败时应记录错误并退避: %+v", st)
	}
	if _, r := fake.count(); r != 2 {
		t.Fatalf("证书变化后应再次重启容器: %d", r)
	}
	// 退避期间定时检查不重试
	m.checkAll(false)
	waitDeploys(t, m)
	if p, _ := fake.count(); p != 1 {
		t.Fatalf("退避期间不应重试: %d", p)
	}

	// 手动部署忽略退避，成功后清除错误
	fake.mu.Lock()
	fake.pushErr = nil
	fake.mu.Unlock()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/certs/a/deploy", nil)
	req.SetPathValue("key", "a")
	m.handleDeploy(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("手动部署失败: %d %s", rec.Code, rec.Body.String())
	}
	m.mu.Lock()
	st = m.stateLocked("a").Deploys[0]
	m.mu.Unlock()
	if st.LastError != "" {
		t.Fatalf("手动部署成功后应清除错误: %+v", st)
	}
}
