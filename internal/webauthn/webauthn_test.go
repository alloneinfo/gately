package webauthn

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"testing"
)

// softKey 是测试用的软件验证器。
type softKey struct {
	priv   *ecdsa.PrivateKey
	credID []byte
}

func newSoftKey(t *testing.T) *softKey {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	id := make([]byte, 16)
	_, _ = rand.Read(id)
	return &softKey{priv: priv, credID: id}
}

func enc(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func cborBytes(b []byte) []byte {
	if len(b) < 24 {
		return append([]byte{0x40 | byte(len(b))}, b...)
	}
	return append([]byte{0x58, byte(len(b))}, b...)
}

func cborText(s string) []byte {
	return append([]byte{0x60 | byte(len(s))}, s...)
}

func (k *softKey) coseKey() []byte {
	pub, _ := k.priv.PublicKey.ECDH()
	raw := pub.Bytes() // 0x04 || X || Y
	var b []byte
	b = append(b, 0xa5)       // map(5)
	b = append(b, 0x01, 0x02) // kty: EC2
	b = append(b, 0x03, 0x26) // alg: -7
	b = append(b, 0x20, 0x01) // crv: P-256
	b = append(b, 0x21)       // x
	b = append(b, cborBytes(raw[1:33])...)
	b = append(b, 0x22) // y
	b = append(b, cborBytes(raw[33:65])...)
	return b
}

func buildAuthData(rpID string, flags byte, count uint32, attested []byte) []byte {
	h := sha256.Sum256([]byte(rpID))
	b := append([]byte{}, h[:]...)
	b = append(b, flags)
	b = binary.BigEndian.AppendUint32(b, count)
	return append(b, attested...)
}

func clientDataJSON(typ, challenge, origin string) string {
	b, _ := json.Marshal(map[string]string{"type": typ, "challenge": challenge, "origin": origin})
	return string(b)
}

func (k *softKey) register(rpID, challenge, origin string, flags byte) RegistrationResponse {
	attested := append(make([]byte, 16), 0, byte(len(k.credID)))
	attested = append(attested, k.credID...)
	attested = append(attested, k.coseKey()...)
	ad := buildAuthData(rpID, flags, 0, attested)
	var obj []byte
	obj = append(obj, 0xa3)
	obj = append(obj, cborText("fmt")...)
	obj = append(obj, cborText("none")...)
	obj = append(obj, cborText("attStmt")...)
	obj = append(obj, 0xa0)
	obj = append(obj, cborText("authData")...)
	obj = append(obj, cborBytes(ad)...)
	var r RegistrationResponse
	r.ID, r.Type = enc(k.credID), "public-key"
	r.Response.ClientDataJSON = enc([]byte(clientDataJSON("webauthn.create", challenge, origin)))
	r.Response.AttestationObject = enc(obj)
	return r
}

func (k *softKey) assert(rpID, challenge, origin string, flags byte, count uint32) AssertionResponse {
	ad := buildAuthData(rpID, flags, count, nil)
	cd := clientDataJSON("webauthn.get", challenge, origin)
	cdHash := sha256.Sum256([]byte(cd))
	digest := sha256.Sum256(append(append([]byte{}, ad...), cdHash[:]...))
	sig, _ := ecdsa.SignASN1(rand.Reader, k.priv, digest[:])
	var r AssertionResponse
	r.ID, r.Type = enc(k.credID), "public-key"
	r.Response.ClientDataJSON = enc([]byte(cd))
	r.Response.AuthenticatorData = enc(ad)
	r.Response.Signature = enc(sig)
	return r
}

func TestRegisterAndLogin(t *testing.T) {
	const host = "gately.example.com:16680"
	const origin = "https://gately.example.com:16680"
	k := newSoftKey(t)
	ch := NewChallenge()
	cred, err := ParseRegistration(k.register("gately.example.com", ch, origin, flagUserPresent|flagUserVerified|flagAttested), ch, host, "我的手机")
	if err != nil {
		t.Fatal(err)
	}
	if cred.ID != enc(k.credID) || cred.Name != "我的手机" {
		t.Fatalf("凭据信息不符: %+v", cred)
	}

	ch2 := NewChallenge()
	count, err := VerifyAssertion(cred, k.assert("gately.example.com", ch2, origin, flagUserPresent|flagUserVerified, 5), ch2, host)
	if err != nil || count != 5 {
		t.Fatalf("登录失败: %v %d", err, count)
	}
	cred.SignCount = count

	// 计数回退（克隆）
	ch3 := NewChallenge()
	if _, err := VerifyAssertion(cred, k.assert("gately.example.com", ch3, origin, flagUserPresent|flagUserVerified, 5), ch3, host); err == nil {
		t.Fatal("签名计数未递增应被拒绝")
	}
	// 计数恒为 0 的验证器（常见于平台密钥）允许
	cred.SignCount = 0
	ch4 := NewChallenge()
	if _, err := VerifyAssertion(cred, k.assert("gately.example.com", ch4, origin, flagUserPresent|flagUserVerified, 0), ch4, host); err != nil {
		t.Fatalf("计数恒为 0 应放行: %v", err)
	}
}

func TestRejections(t *testing.T) {
	const host = "gately.example.com"
	const origin = "https://gately.example.com"
	k := newSoftKey(t)
	ch := NewChallenge()
	cred, err := ParseRegistration(k.register("gately.example.com", ch, origin, 0x45), ch, host, "k")
	if err != nil {
		t.Fatal(err)
	}
	good := func() AssertionResponse { return k.assert("gately.example.com", ch, origin, 0x05, 1) }

	cases := map[string]func() error{
		"挑战不符": func() error { _, e := VerifyAssertion(cred, good(), "other", host); return e },
		"来源不符": func() error {
			_, e := VerifyAssertion(cred, k.assert("gately.example.com", ch, "https://evil.example", 0x05, 1), ch, host)
			return e
		},
		"RP ID 不符": func() error {
			_, e := VerifyAssertion(cred, k.assert("evil.example", ch, origin, 0x05, 1), ch, host)
			return e
		},
		"未做用户验证": func() error {
			_, e := VerifyAssertion(cred, k.assert("gately.example.com", ch, origin, flagUserPresent, 1), ch, host)
			return e
		},
		"签名被篡改": func() error {
			r := good()
			sig, _ := b64(r.Response.Signature)
			sig[len(sig)-1] ^= 1
			r.Response.Signature = enc(sig)
			_, e := VerifyAssertion(cred, r, ch, host)
			return e
		},
		"换了密钥": func() error {
			_, e := VerifyAssertion(cred, newSoftKey(t).assert("gately.example.com", ch, origin, 0x05, 1), ch, host)
			return e
		},
	}
	for name, fn := range cases {
		if fn() == nil {
			t.Errorf("%s 应被拒绝", name)
		}
	}
	// 注册时缺少用户验证
	if _, err := ParseRegistration(k.register("gately.example.com", ch, origin, flagUserPresent|flagAttested), ch, host, "k"); err == nil {
		t.Error("注册缺少用户验证应被拒绝")
	}
}

func TestRPIDAndCBOR(t *testing.T) {
	for in, want := range map[string]string{"a.com:8080": "a.com", "A.com": "a.com", "[::1]:16680": "::1"} {
		if got := RPID(in); got != want {
			t.Errorf("RPID(%q) = %q，期望 %q", in, got, want)
		}
	}
	// 不完整、不支持的 CBOR 不应 panic
	for _, bad := range [][]byte{{}, {0x5f}, {0xa1}, {0x82, 0x01}, {0xc1, 0x00}, {0xf9, 0x00, 0x00}, {0x5a, 0xff, 0xff, 0xff, 0xff}} {
		if _, _, err := cborDecode(bad); err == nil {
			t.Errorf("%x 应解码失败", bad)
		}
	}
}
