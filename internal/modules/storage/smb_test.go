package storage

import (
	"os"
	"testing"
)

func TestParseSMB(t *testing.T) {
	cases := []struct {
		in, addr, share, sub string
		ok                   bool
	}{
		{"smb://nas/share", "nas:445", "share", "", true},
		{"smb://192.168.1.2:4450/data/backup/gately/", "192.168.1.2:4450", "data", "backup/gately", true},
		{"SMB://[fd00::1]/s", "[fd00::1]:445", "s", "", true},
		{"smb://nas/a/../../b", "nas:445", "b", "", true},
		{"smb://nas", "", "", "", false},
		{"smb://nas/", "", "", "", false},
		{"http://nas/share", "", "", "", false},
		{"nas/share", "", "", "", false},
	}
	for _, c := range cases {
		addr, share, sub, err := parseSMB(c.in)
		if (err == nil) != c.ok || addr != c.addr || share != c.share || sub != c.sub {
			t.Errorf("%s: 得到 %q %q %q %v", c.in, addr, share, sub, err)
		}
	}
}

func TestSMBRel(t *testing.T) {
	b, _ := newSMB("smb://nas/share/sub/dir", "", "", "")
	if got := b.rel("/"); got != "sub/dir" {
		t.Fatalf("根路径: %q", got)
	}
	if got := b.rel("/../../x/y"); got != "sub/dir/x/y" {
		t.Fatalf("路径穿越应被限制在子目录内: %q", got)
	}
	b, _ = newSMB("smb://nas/share", "", "", "")
	if got := b.rel("/"); got != "" {
		t.Fatalf("共享根目录应为空串: %q", got)
	}
}

func TestMountValidateSMB(t *testing.T) {
	m := Mount{Name: "x", Type: TypeSMB, URL: " smb://nas:445//share//a/ ", Path: "/tmp", SkipVerify: true, Domain: " WORKGROUP "}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	if m.URL != "smb://nas/share/a" || m.Path != "" || m.SkipVerify || m.Domain != "WORKGROUP" {
		t.Fatalf("规范化结果不正确: %+v", m)
	}
	m = Mount{Name: "x", Type: TypeSMB, URL: "smb://[fd00::1]:4450/s"}
	if err := m.Validate(); err != nil || m.URL != "smb://[fd00::1]:4450/s" {
		t.Fatalf("非默认端口应保留: %v %s", err, m.URL)
	}
	m = Mount{Name: "x", Type: TypeSMB, URL: "smb://nas"}
	if err := m.Validate(); err == nil {
		t.Fatal("缺少共享名应报错")
	}
}

// TestMountValidateSMBEscaped 确认含特殊字符的子目录与 IPv6 zone 规范化后仍能解析回原值，
// 且再次校验结果不变（编辑保存不会把地址越改越坏）。
func TestMountValidateSMBEscaped(t *testing.T) {
	cases := []struct {
		in, addr, share, sub string
	}{
		{"smb://nas/share/a%23b", "nas:445", "share", "a#b"},
		{"smb://nas/share/what%3F/x", "nas:445", "share", "what?/x"},
		{"smb://nas/share/100%25", "nas:445", "share", "100%"},
		{"smb://nas/my%20share/dir%20a", "nas:445", "my share", "dir a"},
		{"smb://[fe80::1%25en0]/s", "[fe80::1%en0]:445", "s", ""},
		{"smb://[fe80::1%25en0]:4450/s/d", "[fe80::1%en0]:4450", "s", "d"},
	}
	for _, c := range cases {
		m := Mount{Name: "x", Type: TypeSMB, URL: c.in}
		if err := m.Validate(); err != nil {
			t.Fatalf("%s: %v", c.in, err)
		}
		addr, share, sub, err := parseSMB(m.URL)
		if err != nil || addr != c.addr || share != c.share || sub != c.sub {
			t.Errorf("%s → %s: 得到 %q %q %q %v", c.in, m.URL, addr, share, sub, err)
		}
		first := m.URL
		if err := m.Validate(); err != nil || m.URL != first {
			t.Errorf("%s: 再次校验结果变化 %q → %q %v", c.in, first, m.URL, err)
		}
	}
}

// TestSMBBackend 连接真实的 SMB 服务做完整读写，需设置 GATELY_TEST_SMB_URL（指向一个可写的空目录），
// 可选 GATELY_TEST_SMB_USER / GATELY_TEST_SMB_PASS；未设置时跳过。
func TestSMBBackend(t *testing.T) {
	url := os.Getenv("GATELY_TEST_SMB_URL")
	if url == "" {
		t.Skip("未设置 GATELY_TEST_SMB_URL")
	}
	b, err := newSMB(url, os.Getenv("GATELY_TEST_SMB_USER"), os.Getenv("GATELY_TEST_SMB_PASS"), "")
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	exercise(t, b)

	// 连接被意外断开后，下一次操作应自动重连
	b.mu.Lock()
	_ = b.conn.Close()
	b.mu.Unlock()
	if _, err := b.List("/"); err != nil {
		t.Fatal("断线后应自动重连:", err)
	}
	if _, err := b.List("/no-such-dir"); err != errNotFound {
		t.Fatalf("不存在的目录应返回 errNotFound: %v", err)
	}
}
