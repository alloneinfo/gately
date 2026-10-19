package luckyimport

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"hash/crc32"
	"os"
	"testing"

	"github.com/klauspost/compress/zstd"
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

// encryptV2ForTest 按 Lucky 3.0 的格式构造 lkcf。
func encryptV2ForTest(t *testing.T, plain []byte, compress bool) []byte {
	t.Helper()
	flags := byte(0)
	body := plain
	if compress {
		zw, err := zstd.NewWriter(nil)
		if err != nil {
			t.Fatal(err)
		}
		body = zw.EncodeAll(plain, nil)
		flags = flagZstdV2
	}
	key := make([]byte, keySize)
	nonce := make([]byte, gcmNonceSizeV2)
	_, _ = rand.Read(key)
	_, _ = rand.Read(nonce)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	payload := append(append([]byte{}, key...), nonce...)
	payload = gcm.Seal(payload, nonce, body, nil)
	head := make([]byte, headerLenV2)
	copy(head, magicV2)
	head[4] = versionV2
	head[5] = 1
	head[6] = flags
	binary.BigEndian.PutUint32(head[8:], uint32(len(payload)))
	binary.BigEndian.PutUint32(head[12:], uint32(len(plain)))
	binary.BigEndian.PutUint16(head[16:], crc16CCITT(head[:16]))
	binary.BigEndian.PutUint32(head[18:], crc32.ChecksumIEEE(payload))
	out := append(head, payload...)
	// 尾部追加模拟签名数据，导入时应被忽略
	return append(out, bytes.Repeat([]byte{0xab}, 256)...)
}

func TestDecryptV2RoundTrip(t *testing.T) {
	plain := []byte(`{"DDNSTaskList":[{"TaskName":"a"}]}`)
	for _, compress := range []bool{false, true} {
		doc, err := Parse(encryptV2ForTest(t, plain, compress))
		if err != nil {
			t.Fatalf("compress=%v: %v", compress, err)
		}
		if _, ok := doc["DDNSTaskList"]; !ok {
			t.Fatalf("compress=%v: 顶层字段丢失", compress)
		}
	}
}

// Lucky 3.0.0 实际生成的默认 DDNS 配置
func TestDecryptV2RealSample(t *testing.T) {
	data, err := os.ReadFile("testdata/lucky3_ddns_default.lkcf")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["DDNSConfigure"]; !ok {
		t.Fatal("缺少 DDNSConfigure")
	}
}

func TestDecryptV2RejectsTampered(t *testing.T) {
	good := encryptV2ForTest(t, []byte(`{"a":1}`), false)
	for name, mutate := range map[string]func(b []byte){
		"头部":  func(b []byte) { b[6] ^= 1 },
		"载荷":  func(b []byte) { b[headerLenV2+40] ^= 1 },
		"版本号": func(b []byte) { b[4] = 3; binary.BigEndian.PutUint16(b[16:], crc16CCITT(b[:16])) },
	} {
		bad := append([]byte{}, good...)
		mutate(bad)
		if _, err := Decrypt(bad); err == nil {
			t.Fatalf("%s被篡改后应解密失败", name)
		}
	}
	if _, err := Decrypt(good[:30]); err == nil {
		t.Fatal("截断文件应报错")
	}
}
