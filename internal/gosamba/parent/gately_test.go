//go:build linux || darwin

package parent

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/wangwanjie/gately/internal/gosamba/smb2"
	"github.com/wangwanjie/gately/internal/gosamba/smb3"
)

// 本文件为 Gately 新增。

func TestWrapNTLMRespFollowsRequestFormat(t *testing.T) {
	tok := append(append([]byte{}, ntlmsspSignature...), 2, 0, 0, 0)
	if got := wrapNTLMResp(true, smb2.SPNEGOAcceptIncomplete, tok); !bytes.Equal(got, tok) {
		t.Fatalf("裸 NTLMSSP 请求应原样回复 NTLMSSP 消息，得到 %x", got)
	}
	if got := wrapNTLMResp(true, smb2.SPNEGOAcceptCompleted, nil); len(got) != 0 {
		t.Fatalf("裸 NTLMSSP 完成时安全缓冲区应为空，得到 %x", got)
	}
	wrapped := wrapNTLMResp(false, smb2.SPNEGOAcceptIncomplete, tok)
	if bytes.HasPrefix(wrapped, ntlmsspSignature) || !bytes.Contains(wrapped, tok) {
		t.Fatalf("SPNEGO 请求应以 SPNEGO 封装回复，得到 %x", wrapped)
	}
	if inner, err := smb2.UnwrapNTLM(wrapped); err != nil || !bytes.Equal(inner, tok) {
		t.Fatalf("SPNEGO 回复解不出原消息: %x %v", inner, err)
	}
}

func TestVerifySignedLocksToMatchingKey(t *testing.T) {
	k128 := bytes.Repeat([]byte{1}, 16)
	k256 := bytes.Repeat([]byte{2}, 16)
	signed := func(key []byte) []byte {
		msg := make([]byte, 100)
		msg[0] = 0xfe
		smb3.SignMessage(smb3.SignAlgoAESCMAC, key, msg)
		return msg
	}

	// Linux 内核：用 L=256 的密钥签名，校验通过后会话改用该密钥
	s := &Session{SigningKey: k128, signingKeyAlt: k256}
	if !s.verifySigned(smb3.SignAlgoAESCMAC, signed(k256)) {
		t.Fatal("L=256 密钥签名应通过")
	}
	if !bytes.Equal(s.signingKey(), k256) || s.signingKeyAlt != nil {
		t.Fatal("应锁定到 L=256 密钥")
	}
	if s.verifySigned(smb3.SignAlgoAESCMAC, signed(k128)) {
		t.Fatal("锁定后不应再接受另一种密钥")
	}

	// macOS：用 L=128 的密钥签名，保持原密钥
	s = &Session{SigningKey: k128, signingKeyAlt: k256}
	if !s.verifySigned(smb3.SignAlgoAESCMAC, signed(k128)) || !bytes.Equal(s.signingKey(), k128) || s.signingKeyAlt != nil {
		t.Fatal("L=128 密钥签名应通过并保持原密钥")
	}

	// 没有备用密钥时只接受主密钥
	s = &Session{SigningKey: k128}
	if s.verifySigned(smb3.SignAlgoAESCMAC, signed(k256)) {
		t.Fatal("没有备用密钥时不应接受其他密钥")
	}
}

// TestDurableTableCloseAll 验证服务重启时 CloseAll 关闭全部保留的句柄（已断线与未断线的都关闭），并清空表。
func TestDurableTableCloseAll(t *testing.T) {
	tbl := NewDurableTable()
	var files []*os.File
	for i := 0; i < 3; i++ {
		f, err := os.CreateTemp(t.TempDir(), "closeall-*.txt")
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
		var cg, crg [16]byte
		crg[0] = byte(i + 1)
		tbl.Register(cg, crg, &Open{File: f}, time.Minute, "share", "alice")
		if i > 0 {
			// 其中两个已断线，正在等待续用
			tbl.Detach(cg, crg)
		}
	}
	if n := tbl.CloseAll(); n != 3 {
		t.Fatalf("CloseAll 关闭 %d 个，期望 3", n)
	}
	if tbl.len() != 0 {
		t.Fatalf("CloseAll 后表中还有 %d 项", tbl.len())
	}
	for i, f := range files {
		if err := f.Close(); err == nil {
			t.Fatalf("第 %d 个文件应已被 CloseAll 关闭", i)
		}
	}
	// 再次调用与 nil 表都不应出错
	if n := tbl.CloseAll(); n != 0 {
		t.Fatalf("空表 CloseAll 返回 %d", n)
	}
	var nilTbl *DurableTable
	if n := nilTbl.CloseAll(); n != 0 {
		t.Fatalf("nil 表 CloseAll 返回 %d", n)
	}
}
