package webterminal

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// startSFTPServer 启动一个只支持 sftp 子系统的 SSH 服务端，直接操作本机文件系统。
func startSFTPServer(t *testing.T, clientPub ssh.PublicKey) string {
	t.Helper()
	_, hostPriv, _ := ed25519.GenerateKey(rand.Reader)
	hostSigner, _ := ssh.NewSignerFromKey(hostPriv)
	cfg := &ssh.ServerConfig{PublicKeyCallback: func(_ ssh.ConnMetadata, k ssh.PublicKey) (*ssh.Permissions, error) {
		if string(k.Marshal()) == string(clientPub.Marshal()) {
			return nil, nil
		}
		return nil, io.EOF
	}}
	cfg.AddHostKey(hostSigner)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			nc, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				_, chans, reqs, err := ssh.NewServerConn(nc, cfg)
				if err != nil {
					return
				}
				go ssh.DiscardRequests(reqs)
				for nch := range chans {
					if nch.ChannelType() != "session" {
						_ = nch.Reject(ssh.UnknownChannelType, "unsupported")
						continue
					}
					ch, creqs, err := nch.Accept()
					if err != nil {
						continue
					}
					go func() {
						for req := range creqs {
							ok := req.Type == "subsystem" && len(req.Payload) > 4 && string(req.Payload[4:]) == "sftp"
							_ = req.Reply(ok, nil)
							if ok {
								go func() {
									srv, err := sftp.NewServer(ch)
									if err != nil {
										return
									}
									_ = srv.Serve()
									_ = srv.Close()
								}()
							}
						}
					}()
				}
			}()
		}
	}()
	return ln.Addr().String()
}

// sftpEnv 描述一个测试环境：key 为连接 key，root 为可读写的临时目录。
type sftpEnv struct {
	t    *testing.T
	m    *Module
	srv  *httptest.Server
	key  string
	root string
}

func newLocalEnv(t *testing.T) *sftpEnv {
	if runtime.GOOS == "windows" {
		t.Skip("本机文件管理测试仅在 Unix 运行")
	}
	m, srv := newModule(t)
	m.cfg.Security.AgreementAccepted = true
	m.cfg.Settings.AllowLocalShell = true
	m.cfg.Connections = []Connection{{Key: "l", Name: "local", Type: TypeLocal}}
	return &sftpEnv{t: t, m: m, srv: srv, key: "l", root: t.TempDir()}
}

func newSSHEnv(t *testing.T) *sftpEnv {
	if runtime.GOOS == "windows" {
		t.Skip("SFTP 测试仅在 Unix 运行")
	}
	keyPEM, pub := newKey(t)
	addr := startSFTPServer(t, pub)
	host, port, _ := net.SplitHostPort(addr)
	m, srv := newModule(t)
	m.cfg.Security.AgreementAccepted = true
	c := Connection{Key: "s", Name: "nas", SSH: SSHConfig{Host: host, Username: "root", AuthType: "key", PrivateKey: keyPEM}}
	c.SSH.Port, _ = strconv.Atoi(port)
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	m.cfg.Connections = []Connection{c}
	return &sftpEnv{t: t, m: m, srv: srv, key: "s", root: t.TempDir()}
}

func (e *sftpEnv) url(op string) string {
	return e.srv.URL + "/api/webterminal/sftp/" + e.key + "/" + op
}

// call 发送 JSON 请求，返回状态码与响应体。
func (e *sftpEnv) call(method, op string, body any) (int, []byte) {
	e.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, e.url(op), rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}

func (e *sftpEnv) mustOK(method, op string, body any) []byte {
	e.t.Helper()
	code, data := e.call(method, op, body)
	if code != 200 {
		e.t.Fatalf("%s %s 状态 %d: %s", method, op, code, data)
	}
	return data
}

func (e *sftpEnv) join(parts ...string) string {
	return filepath.ToSlash(filepath.Join(append([]string{e.root}, parts...)...))
}

// upload 以 multipart 上传，files 的 key 为文件名（可含相对路径）。
func (e *sftpEnv) upload(dir string, files map[string]string, relPaths []string) (int, []byte) {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("path", dir)
	for _, rp := range relPaths {
		_ = mw.WriteField("relativePaths", rp)
	}
	for name, content := range files {
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", `form-data; name="files"; filename="`+name+`"`)
		h.Set("Content-Type", "application/octet-stream")
		pw, _ := mw.CreatePart(h)
		_, _ = pw.Write([]byte(content))
	}
	_ = mw.Close()
	req, _ := http.NewRequest("POST", e.url("upload"), &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}

func (e *sftpEnv) get(op, query string) (int, http.Header, []byte) {
	e.t.Helper()
	resp, err := http.Get(e.url(op) + "?" + query)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, data
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// exerciseBasic 覆盖 list/read/write/upload/mkdir/touch/remove 的基本流程，本机与 SFTP 共用。
func exerciseBasic(t *testing.T, e *sftpEnv) {
	// write + read 预览
	f := e.join("hello.txt")
	e.mustOK("POST", "write", map[string]any{"path": f, "content": "你好 gately"})
	if readFile(t, f) != "你好 gately" {
		t.Fatal("写入内容不一致")
	}
	_, _, data := e.get("read", "path="+f)
	var rd struct {
		Content string `json:"content"`
	}
	_ = json.Unmarshal(data, &rd)
	if rd.Content != "你好 gately" {
		t.Fatalf("预览内容不一致: %s", data)
	}

	// mkdir / touch / 已存在冲突
	e.mustOK("POST", "mkdir", map[string]any{"path": e.join("sub", "deep")})
	e.mustOK("POST", "touch", map[string]any{"path": e.join("sub", "a.txt")})
	if code, _ := e.call("POST", "touch", map[string]any{"path": e.join("sub", "a.txt")}); code != 409 {
		t.Fatalf("重复 touch 应 409: %d", code)
	}

	// 上传：普通文件、相对路径目录、路径穿越
	if code, body := e.upload(e.join("sub"), map[string]string{"up.txt": "u1"}, nil); code != 200 {
		t.Fatalf("上传失败 %d: %s", code, body)
	}
	if readFile(t, e.join("sub", "up.txt")) != "u1" {
		t.Fatal("上传内容不一致")
	}
	if code, body := e.upload(e.join("sub"), map[string]string{"dir1/x.txt": "u2"}, nil); code != 200 {
		t.Fatalf("目录上传失败 %d: %s", code, body)
	}
	if readFile(t, e.join("sub", "dir1", "x.txt")) != "u2" {
		t.Fatal("目录上传内容不一致")
	}
	if code, body := e.upload(e.join("sub"), map[string]string{"y.txt": "u3"}, []string{"rel/y.txt"}); code != 200 {
		t.Fatalf("relativePaths 上传失败 %d: %s", code, body)
	}
	if readFile(t, e.join("sub", "rel", "y.txt")) != "u3" {
		t.Fatal("relativePaths 上传位置错误")
	}
	if code, _ := e.upload(e.join("sub"), map[string]string{"../../evil.txt": "x"}, nil); code != 400 {
		t.Fatalf("路径穿越应 400: %d", code)
	}
	if _, err := os.Stat(e.join("evil.txt")); err == nil {
		t.Fatal("路径穿越文件不应被写出")
	}

	// list：目录在前
	_, _, data = e.get("list", "path="+e.join("sub"))
	var ls struct {
		Path    string      `json:"path"`
		Parent  string      `json:"parent"`
		Entries []fileEntry `json:"entries"`
	}
	if err := json.Unmarshal(data, &ls); err != nil {
		t.Fatal(err)
	}
	if ls.Path != e.join("sub") || ls.Parent != e.root || len(ls.Entries) < 4 {
		t.Fatalf("list 结果异常: %s", data)
	}
	if !ls.Entries[0].IsDir || ls.Entries[len(ls.Entries)-1].IsDir {
		t.Fatalf("目录应排在前面: %s", data)
	}

	// 下载与 Range
	code, hdr, body := e.get("read", "download=1&path="+e.join("sub", "up.txt"))
	if code != 200 || string(body) != "u1" || !strings.Contains(hdr.Get("Content-Disposition"), "up.txt") {
		t.Fatalf("下载异常: %d %v %q", code, hdr, body)
	}

	// 不存在 404
	if code, _, _ := e.get("read", "path="+e.join("nope")); code != 404 {
		t.Fatalf("不存在应 404: %d", code)
	}

	// 删除：根目录被拒，目录递归删除
	if code, _ := e.call("DELETE", "remove", map[string]any{"paths": []string{"/"}}); code != 403 {
		t.Fatalf("删除根目录应 403: %d", code)
	}
	e.mustOK("DELETE", "remove", map[string]any{"paths": []string{e.join("sub")}})
	if _, err := os.Stat(e.join("sub")); err == nil {
		t.Fatal("目录应已被递归删除")
	}
}

func TestSFTPLocalBasic(t *testing.T) {
	exerciseBasic(t, newLocalEnv(t))
}

func TestSFTPRemoteBasic(t *testing.T) {
	exerciseBasic(t, newSSHEnv(t))
}

func TestSFTPLocalGateAndTypes(t *testing.T) {
	e := newLocalEnv(t)
	m, srv := newModule(t)
	m.cfg.Security.AgreementAccepted = true
	m.cfg.Connections = []Connection{{Key: "l", Name: "local", Type: TypeLocal}}
	resp, err := http.Get(srv.URL + "/api/webterminal/sftp/l/list?path=/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Fatalf("未开启本机 Shell 应 403: %d", resp.StatusCode)
	}
	if code, _, _ := e.get("list", "path=relative"); code != 400 {
		t.Fatalf("相对路径应 400: %d", code)
	}
	resp, _ = http.Get(e.srv.URL + "/api/webterminal/sftp/nokey/list?path=/")
	resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("未知连接应 404: %d", resp.StatusCode)
	}
}

func TestSFTPPreviewLimits(t *testing.T) {
	e := newLocalEnv(t)
	big := e.join("big.txt")
	_ = os.WriteFile(big, bytes.Repeat([]byte("a"), previewLimit+1), 0o644)
	if code, _, _ := e.get("read", "path="+big); code != 413 {
		t.Fatalf("过大应 413: %d", code)
	}
	if code, _, body := e.get("read", "download=1&path="+big); code != 200 || len(body) != previewLimit+1 {
		t.Fatalf("大文件应可下载: %d", code)
	}
	bin := e.join("bin.dat")
	_ = os.WriteFile(bin, []byte{0x00, 0x01, 0x02}, 0o644)
	if code, _, _ := e.get("read", "path="+bin); code != 415 {
		t.Fatalf("二进制应 415: %d", code)
	}
}

func TestSFTPRenameCopyChmod(t *testing.T) {
	e := newLocalEnv(t)
	_ = os.MkdirAll(e.join("d", "inner"), 0o755)
	_ = os.WriteFile(e.join("d", "inner", "f.txt"), []byte("x"), 0o644)
	_ = os.WriteFile(e.join("a.txt"), []byte("a"), 0o644)
	_ = os.WriteFile(e.join("b.txt"), []byte("b"), 0o644)

	e.mustOK("POST", "rename", map[string]any{"from": e.join("a.txt"), "to": e.join("a2.txt")})
	if code, _ := e.call("POST", "rename", map[string]any{"from": e.join("a2.txt"), "to": e.join("b.txt")}); code != 409 {
		t.Fatalf("重命名到已存在应 409: %d", code)
	}
	if code, _ := e.call("POST", "rename", map[string]any{"from": e.join("d"), "to": e.join("d", "inner", "d2")}); code != 400 {
		t.Fatalf("移动到自身内部应 400: %d", code)
	}

	// 递归复制；重复复制自动改名
	e.mustOK("POST", "copy", map[string]any{"from": []string{e.join("d")}, "to": e.root})
	if readFile(t, e.join("d 副本", "inner", "f.txt")) != "x" {
		t.Fatal("递归复制失败")
	}
	e.mustOK("POST", "copy", map[string]any{"from": []string{e.join("b.txt")}, "to": e.root})
	if readFile(t, e.join("b 副本.txt")) != "b" {
		t.Fatal("文件复制自动改名失败")
	}
	if code, _ := e.call("POST", "copy", map[string]any{"from": []string{e.join("d")}, "to": e.join("d", "inner")}); code != 400 {
		t.Fatalf("复制到自身内部应 400: %d", code)
	}

	// chmod 递归
	e.mustOK("POST", "chmod", map[string]any{"path": e.join("d"), "mode": "700", "recursive": true})
	fi, _ := os.Stat(e.join("d", "inner", "f.txt"))
	if fi.Mode().Perm() != 0o700 {
		t.Fatalf("递归 chmod 未生效: %v", fi.Mode())
	}
	if code, _ := e.call("POST", "chmod", map[string]any{"path": e.join("d"), "mode": "999"}); code != 400 {
		t.Fatalf("非法权限应 400: %d", code)
	}
}

func TestSFTPCompressDecompress(t *testing.T) {
	for _, tc := range []struct{ format, name string }{{"zip", "out.zip"}, {"tar.gz", "out.tar.gz"}} {
		t.Run(tc.format, func(t *testing.T) {
			e := newLocalEnv(t)
			_ = os.MkdirAll(e.join("src", "sub"), 0o755)
			_ = os.WriteFile(e.join("src", "sub", "f.txt"), []byte("内容"), 0o644)
			_ = os.WriteFile(e.join("single.txt"), []byte("s"), 0o644)
			archive := e.join(tc.name)
			e.mustOK("POST", "compress", map[string]any{
				"paths": []string{e.join("src"), e.join("single.txt")}, "dest": archive, "format": tc.format,
			})
			_, _, data := e.get("preview-archive", "path="+archive)
			var items []struct {
				Name  string `json:"name"`
				IsDir bool   `json:"isDir"`
			}
			_ = json.Unmarshal(data, &items)
			names := map[string]bool{}
			for _, it := range items {
				names[it.Name] = true
			}
			if !names["src/sub/f.txt"] || !names["single.txt"] || !names["src"] {
				t.Fatalf("压缩包条目异常: %s", data)
			}
			out := e.join("out")
			e.mustOK("POST", "decompress", map[string]any{"path": archive, "dest": out})
			if readFile(t, e.join("out", "src", "sub", "f.txt")) != "内容" || readFile(t, e.join("out", "single.txt")) != "s" {
				t.Fatal("解压内容不一致")
			}
			if code, _ := e.call("POST", "compress", map[string]any{"paths": []string{e.join("src")}, "dest": archive, "format": tc.format}); code != 409 {
				t.Fatalf("目标压缩包已存在应 409: %d", code)
			}
		})
	}
}

// buildSlipZip 构造含 "../" 条目的恶意 zip。
func buildSlipZip(t *testing.T, p string) {
	var buf bytes.Buffer
	zw := newZipWriter(&buf)
	w, _ := zw.Create("../evil.txt")
	_, _ = w.Write([]byte("evil"))
	_ = zw.Close()
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSFTPZipSlipAndTar(t *testing.T) {
	e := newLocalEnv(t)
	_ = os.MkdirAll(e.join("work"), 0o755)
	slip := e.join("work", "slip.zip")
	buildSlipZip(t, slip)
	dest := e.join("work", "dest")
	if code, _ := e.call("POST", "decompress", map[string]any{"path": slip, "dest": dest}); code != 400 {
		t.Fatalf("zip-slip 应 400: %d", code)
	}
	if _, err := os.Stat(e.join("work", "evil.txt")); err == nil {
		t.Fatal("zip-slip 文件不应被写出")
	}

	// 未压缩 tar：含目录、文件与符号链接（符号链接应被跳过）
	var buf bytes.Buffer
	writeTar(&buf)
	tarPath := e.join("t.tar")
	_ = os.WriteFile(tarPath, buf.Bytes(), 0o644)
	e.mustOK("POST", "decompress", map[string]any{"path": tarPath, "dest": e.join("tout")})
	if readFile(t, e.join("tout", "d", "f.txt")) != "tar" {
		t.Fatal("tar 解压失败")
	}
	if _, err := os.Lstat(e.join("tout", "d", "link")); err == nil {
		t.Fatal("符号链接条目应被跳过")
	}
}

func TestSFTPRemoteCompress(t *testing.T) {
	e := newSSHEnv(t)
	_ = os.MkdirAll(e.join("src"), 0o755)
	_ = os.WriteFile(e.join("src", "f.txt"), []byte("远端"), 0o644)
	archive := e.join("r.zip")
	e.mustOK("POST", "compress", map[string]any{"paths": []string{e.join("src")}, "dest": archive, "format": "zip"})
	e.mustOK("POST", "decompress", map[string]any{"path": archive, "dest": e.join("o")})
	if readFile(t, e.join("o", "src", "f.txt")) != "远端" {
		t.Fatal("远端压缩解压失败")
	}
}
