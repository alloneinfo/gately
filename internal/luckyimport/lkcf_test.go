package luckyimport

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"testing"
)

// encryptForTest 按 Lucky 的格式构造 lkcf，用于验证解密实现。
func encryptForTest(t *testing.T, plain []byte) []byte {
	t.Helper()
	key := make([]byte, keySize)
	iv := make([]byte, ivSize)
	_, _ = rand.Read(key)
	_, _ = rand.Read(iv)
	pad := aes.BlockSize - len(plain)%aes.BlockSize
	padded := append(append([]byte{}, plain...), bytes.Repeat([]byte{byte(pad)}, pad)...)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	ct := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ct, padded)
	out := []byte{versionV1}
	out = append(out, key...)
	out = append(out, iv...)
	return append(out, ct...)
}

func TestDecryptRoundTrip(t *testing.T) {
	plain := []byte(`{"WOLDeviceList":[{"DeviceName":"pc"}]}`)
	got, err := Decrypt(encryptForTest(t, plain))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("got %s", got)
	}
	doc, err := Parse(encryptForTest(t, plain))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["WOLDeviceList"]; !ok {
		t.Fatal("顶层字段丢失")
	}
}

func TestDecryptGzipPayload(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte(`{"a":1}`))
	_ = zw.Close()
	got, err := Decrypt(encryptForTest(t, buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"a":1}` {
		t.Fatalf("got %s", got)
	}
}

func TestDecryptPlainJSONPassthrough(t *testing.T) {
	got, err := Decrypt([]byte("  {\"x\":true}\n"))
	if err != nil || string(got) != `{"x":true}` {
		t.Fatalf("got %s %v", got, err)
	}
}

func TestDecryptRejectsGarbage(t *testing.T) {
	if _, err := Decrypt([]byte{1, 2, 3}); err == nil {
		t.Fatal("过短输入应报错")
	}
	bad := encryptForTest(t, []byte(`{"a":1}`))
	bad[0] = 2
	if _, err := Decrypt(bad); err == nil {
		t.Fatal("未知版本应报错")
	}
	// 篡改密钥后填充校验或 JSON 校验应失败
	tampered := encryptForTest(t, []byte(`{"a":1}`))
	tampered[5] ^= 0xff
	if _, err := Decrypt(tampered); err == nil {
		t.Fatal("篡改后应解密失败")
	}
}
