package luckyimport

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"sync"

	"github.com/klauspost/compress/zstd"
)

// Lucky 3.0 的 .lkcf（V2）文件格式：
//
//	[0:4]    魔数 "LKCF"
//	[4]      格式版本，固定为 0x02
//	[5]      保留
//	[6]      标志位，bit0 表示明文经过 zstd 压缩
//	[7]      保留
//	[8:12]   载荷长度 N（大端）
//	[12:16]  原始数据长度（大端，仅作参考）
//	[16:18]  头部 CRC16-CCITT（大端），校验 [0:16]
//	[18:22]  载荷 CRC32-IEEE（大端）
//	[22:24]  保留
//	[24:24+N] 载荷：AES-256 密钥(32) + GCM 随机数(12) + 密文与认证标签
//	之后     RSA 签名等尾部数据，导入时忽略
//
// 密钥随文件保存，与 V1 一样无需用户口令即可解密。
const (
	magicV2        = "LKCF"
	versionV2      = 0x02
	headerLenV2    = 24
	minFileLenV2   = headerLenV2 + keySize + 12 + 16
	flagZstdV2     = 0x01
	gcmNonceSizeV2 = 12
)

var (
	zstdOnce sync.Once
	zstdDec  *zstd.Decoder
	zstdErr  error
)

// isV2 判断数据是否以 V2 魔数开头。
func isV2(data []byte) bool {
	return len(data) >= len(magicV2) && string(data[:len(magicV2)]) == magicV2
}

// crc16CCITT 计算 CRC16-CCITT（多项式 0x1021，初值 0）。
func crc16CCITT(b []byte) uint16 {
	var crc uint16
	for _, c := range b {
		crc ^= uint16(c) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// decryptV2 解密 Lucky 3.0 的 lkcf，返回 JSON 明文。
func decryptV2(data []byte) ([]byte, error) {
	if len(data) < minFileLenV2 {
		return nil, errors.New("文件太短，不是有效的 lkcf")
	}
	if crc16CCITT(data[:16]) != binary.BigEndian.Uint16(data[16:18]) {
		return nil, errors.New("lkcf 头部校验失败，文件可能已损坏")
	}
	if data[4] != versionV2 {
		return nil, fmt.Errorf("不支持的 lkcf 版本 %d", data[4])
	}
	n := uint64(binary.BigEndian.Uint32(data[8:12]))
	if n < keySize+gcmNonceSizeV2+16 || n > uint64(len(data)-headerLenV2) {
		return nil, errors.New("lkcf 载荷长度无效")
	}
	payload := data[headerLenV2 : headerLenV2+int(n)]
	if crc32.ChecksumIEEE(payload) != binary.BigEndian.Uint32(data[18:22]) {
		return nil, errors.New("lkcf 载荷校验失败，文件可能已损坏")
	}
	block, err := aes.NewCipher(payload[:keySize])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := payload[keySize : keySize+gcmNonceSizeV2]
	plain, err := gcm.Open(nil, nonce, payload[keySize+gcmNonceSizeV2:], nil)
	if err != nil {
		return nil, fmt.Errorf("lkcf 解密失败: %w", err)
	}
	if data[6]&flagZstdV2 != 0 {
		zstdOnce.Do(func() {
			zstdDec, zstdErr = zstd.NewReader(nil, zstd.WithDecoderMaxMemory(maxPlainSize))
		})
		if zstdErr != nil {
			return nil, zstdErr
		}
		plain, err = zstdDec.DecodeAll(plain, nil)
		if err != nil {
			return nil, fmt.Errorf("lkcf 解压失败: %w", err)
		}
	}
	return bytes.TrimSpace(plain), nil
}
