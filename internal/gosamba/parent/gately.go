//go:build linux || darwin

package parent

import (
	"github.com/wangwanjie/gately/internal/gosamba/config"
	"github.com/wangwanjie/gately/internal/gosamba/smb2"
	"github.com/wangwanjie/gately/internal/gosamba/smb3"
)

// 本文件为 Gately 新增，不属于上游 gosamba。

// sharesFor 返回会话可连接的共享：已登录用户带有专属共享列表时使用该列表，否则使用全局共享。
// Gately 按用户配置目录，不同用户的同名共享可以指向不同目录，因此共享查找与共享枚举都按用户进行。
func (d *Dispatcher) sharesFor(sess *Session) []config.ShareConfig {
	if sess != nil && !sess.IsGuest && sess.User.Shares != nil {
		return sess.User.Shares
	}
	return d.Shares
}

// ntlmsspSignature 是 NTLMSSP 消息开头的 8 字节签名。
var ntlmsspSignature = []byte("NTLMSSP\x00")

// wrapNTLMResp 按客户端请求的格式封装 SESSION_SETUP 回复：客户端发送裸 NTLMSSP（Linux 内核 cifs 的默认方式）时
// 回复也不带 SPNEGO，完成时安全缓冲区为空；否则按 SPNEGO 封装。
func wrapNTLMResp(raw bool, state smb2.SPNEGOState, tok []byte) []byte {
	if raw {
		return tok
	}
	return smb2.WrapNTLMResp(state, tok)
}

// verifySigned 校验请求签名。SMB 3.1.1 协商出 AES-256 加密算法时，客户端派生签名密钥所用的长度参数不一致：
// macOS 用 L=128，Linux 内核 cifs 用 L=256 再取前 16 字节。两种密钥都尝试，第一次验证通过后把会话锁定到该密钥，
// 之后的回复也用它签名。
func (s *Session) verifySigned(algo uint16, frame []byte) bool {
	s.signingMu.Lock()
	defer s.signingMu.Unlock()
	if smb3.VerifyMessage(algo, s.SigningKey, frame) {
		s.signingKeyAlt = nil
		return true
	}
	if s.signingKeyAlt != nil && smb3.VerifyMessage(algo, s.signingKeyAlt, frame) {
		s.SigningKey, s.signingKeyAlt = s.signingKeyAlt, nil
		return true
	}
	return false
}

// signingKey 返回会话当前使用的签名密钥。
func (s *Session) signingKey() []byte {
	s.signingMu.Lock()
	defer s.signingMu.Unlock()
	return s.SigningKey
}

// setSigningKeys 在锁内写入会话签名密钥与 AES-256 时的备用密钥。
// 会话可能已登记到跨连接共享的会话索引，读取方（签名、校验）都经 signingMu，写入也必须加锁。
func (s *Session) setSigningKeys(key, alt []byte) {
	s.signingMu.Lock()
	defer s.signingMu.Unlock()
	s.SigningKey, s.signingKeyAlt = key, alt
}

// CloseAll 关闭表中所有为断线续用保留的句柄：释放字节范围锁与共享模式占用并关闭文件，返回关闭的数量。
// Gately 每次保存配置都会重启 SMB 服务，旧服务的句柄表不会再被任何连接使用，
// 不主动关闭的话文件描述符与锁要等到 GC 才释放。调用方须保证使用该表的连接都已结束。
func (t *DurableTable) CloseAll() int {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	n := 0
	for k, e := range t.entries {
		// 与连接收尾一致：持有句柄锁再释放，避免与仍在收尾的异步操作并发
		unlock := lockOpenForRelease(e.open)
		releaseOpen(e.open)
		unlock()
		delete(t.entries, k)
		n++
	}
	return n
}

// IsTemporaryAcceptError 导出 isTemporaryAcceptError，供 Gately 判断监听是否已不可恢复地失败：
// Serve 遇到不可恢复的 Accept 错误后要等全部连接结束才返回，调用方需要更早知道服务已停止接受连接。
func IsTemporaryAcceptError(err error) bool {
	return isTemporaryAcceptError(err)
}
