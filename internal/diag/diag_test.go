package diag

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestTextRedaction(t *testing.T) {
	r := NewRedactor("/Users/alice/gately-data")
	cases := []struct{ in, want string }{
		{"连接 192.168.1.20:8080 失败", "连接 <ipv4:private>:8080 失败"},
		{"外网地址 8.8.8.8", "外网地址 <ipv4:public>"},
		{"CGNAT 100.100.1.1", "CGNAT <ipv4:cgnat>"},
		{"监听 [2001:db8::1]:443", "监听 [<ipv6:public>]:443"},
		{"解析 home.example.com 失败", "解析 <domain> 失败"},
		{"访问 api.github.com 超时", "访问 api.github.com 超时"},
		{"读取 /Users/alice/gately-data/ddns.json", "读取 <config-dir>/ddns.json"},
		{"打开 /home/bob/.ssh/id_rsa", "打开 /home/<user>/.ssh/id_rsa"},
		{"通知 me@example.com", "通知 <email>"},
		{"请求 https://dav.example.com/path?token=abc", "请求 https://<url>"},
		{"连接 unix:///var/run/docker.sock", "连接 unix:///var/run/docker.sock"},
		{"password=hunter2 ok", "password=<redacted> ok"},
		{"Authorization: Bearer abcdef", "Authorization: <redacted> <redacted>"},
		{"mac aa:bb:cc:dd:ee:ff", "mac <mac>"},
		{"密钥 " + strings.Repeat("a1B2", 10), "密钥 <secret>"},
		{"时间 10:30:45 正常", "时间 10:30:45 正常"},
		{"见 main.go:12", "见 main.go:12"},
	}
	for _, c := range cases {
		if got := r.Text(c.in); got != c.want {
			t.Errorf("Text(%q) = %q，期望 %q", c.in, got, c.want)
		}
	}
}

func TestAddr(t *testing.T) {
	r := NewRedactor("")
	cases := map[string]string{
		":16680":                  ":16680",
		"192.168.1.5:80":          "<ipv4:private>:80",
		"nas.local:22":            "<host>:22",
		"my-nas:22":               "<host>:22",
		"tcp://1.2.3.4:5000/path": "tcp://<ipv4:public>:5000",
		"[fe80::1]:53":            "<ipv6:linklocal>:53",
		"10000-10010":             "10000-10010",
		"user:pw@example.com:21":  "<host>:21",
	}
	for in, want := range cases {
		if got := r.Addr(in); got != want {
			t.Errorf("Addr(%q) = %q，期望 %q", in, got, want)
		}
	}
}

func TestSanitizeJSONNoLeak(t *testing.T) {
	r := NewRedactor("")
	raw := `{
	  "settings": {"enable": true, "authSecret": "aff37ff1e099451d97cb2bc50e406f59ea540d93744439325bd3caa05216018d", "autoBanSeconds": 30},
	  "listeners": [{"id": "a1b2c3d4e5f6a7b8", "name": "家里的NAS", "listen": ":443", "protocol": "tcp",
	    "target": "192.168.1.9:5000", "password": "hunter2", "remark": "my secret note", "domains": ["nas.example.com"],
	    "lastError": "dial 10.0.0.2:80 refused", "notAfter": "2027-01-02T03:04:05Z", "pin": 123456}],
	  "headers": {"X-Api-Key": "k1", "nas.example.com": {"type": "http"}}
	}`
	out, err := r.SanitizeJSON([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, leak := range []string{"aff37ff1", "家里的NAS", "hunter2", "my secret note", "nas.example.com", "192.168.1.9", "10.0.0.2", "a1b2c3d4e5f6a7b8", "X-Api-Key", "k1\"", "123456"} {
		if strings.Contains(s, leak) {
			t.Errorf("脱敏结果仍含 %q:\n%s", leak, s)
		}
	}
	for _, keep := range []string{`"protocol": "tcp"`, `"listen": ":443"`, `<ipv4:private>:80`, `"notAfter": "2027-01-02T03:04:05Z"`, `"autoBanSeconds": 30`, `"enable": true`, `"type": "http"`} {
		if !strings.Contains(s, keep) {
			t.Errorf("脱敏结果应保留 %q:\n%s", keep, s)
		}
	}
}

func TestSanitizeJSONArrayCap(t *testing.T) {
	r := NewRedactor("")
	var sb strings.Builder
	sb.WriteString(`{"ports":[`)
	for i := 0; i < 150; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("80")
	}
	sb.WriteString("]}")
	out, err := r.SanitizeJSON([]byte(sb.String()))
	if err != nil || !strings.Contains(string(out), "50 more items omitted") {
		t.Fatalf("应截断过长数组: %v\n%s", err, out)
	}
}

func TestBundleZip(t *testing.T) {
	b := &Bundle{}
	b.Add("a.txt", []byte("hello"))
	data, err := b.Zip(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(zr.File) != 1 || zr.File[0].Name != "a.txt" {
		t.Fatalf("zip 内容异常: %v", err)
	}
}
