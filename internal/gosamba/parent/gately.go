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
