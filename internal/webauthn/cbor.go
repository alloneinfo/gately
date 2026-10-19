package webauthn

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// 本文件实现 WebAuthn 所需的最小 CBOR（RFC 8949）解码：整数、字节串、文本串、数组、映射与简单值。
// 不支持标签、浮点数与不定长项目；遇到这些内容一律报错，而不是尝试解释。

const (
	cborMaxDepth = 8
	cborMaxItems = 1024
)

var errCBOR = errors.New("CBOR 数据无效")

// cborDecode 解码 data 开头的一个 CBOR 项目，返回值与消耗的字节数。
// 映射解码为 map[any]any（键为 int64 或 string），数组为 []any，字节串为 []byte。
func cborDecode(data []byte) (any, int, error) {
	return cborItem(data, 0, 0)
}

func cborItem(data []byte, pos, depth int) (any, int, error) {
	if depth > cborMaxDepth {
		return nil, 0, errCBOR
	}
	if pos >= len(data) {
		return nil, 0, errCBOR
	}
	major := data[pos] >> 5
	info := data[pos] & 0x1f
	pos++
	arg, n, err := cborArg(data, pos, info)
	if err != nil {
		return nil, 0, err
	}
	pos += n
	switch major {
	case 0: // 无符号整数
		if arg > 1<<62 {
			return nil, 0, errCBOR
		}
		return int64(arg), pos, nil
	case 1: // 负整数
		if arg > 1<<62 {
			return nil, 0, errCBOR
		}
		return -1 - int64(arg), pos, nil
	case 2, 3: // 字节串、文本串
		if arg > uint64(len(data)-pos) {
			return nil, 0, errCBOR
		}
		b := data[pos : pos+int(arg)]
		pos += int(arg)
		if major == 2 {
			return append([]byte{}, b...), pos, nil
		}
		return string(b), pos, nil
	case 4: // 数组
		if arg > cborMaxItems {
			return nil, 0, errCBOR
		}
		out := make([]any, 0, arg)
		for i := uint64(0); i < arg; i++ {
			v, np, err := cborItem(data, pos, depth+1)
			if err != nil {
				return nil, 0, err
			}
			out, pos = append(out, v), np
		}
		return out, pos, nil
	case 5: // 映射
		if arg > cborMaxItems {
			return nil, 0, errCBOR
		}
		out := make(map[any]any, arg)
		for i := uint64(0); i < arg; i++ {
			k, np, err := cborItem(data, pos, depth+1)
			if err != nil {
				return nil, 0, err
			}
			switch k.(type) {
			case int64, string:
			default:
				return nil, 0, fmt.Errorf("%w: 不支持的映射键类型", errCBOR)
			}
			v, np2, err := cborItem(data, np, depth+1)
			if err != nil {
				return nil, 0, err
			}
			out[k], pos = v, np2
		}
		return out, pos, nil
	case 7: // 简单值
		switch info {
		case 20:
			return false, pos, nil
		case 21:
			return true, pos, nil
		case 22:
			return nil, pos, nil
		}
	}
	return nil, 0, fmt.Errorf("%w: 不支持的数据类型", errCBOR)
}

// cborArg 读取头字节之后的参数值，返回参数与消耗的字节数。
func cborArg(data []byte, pos int, info byte) (uint64, int, error) {
	switch {
	case info < 24:
		return uint64(info), 0, nil
	case info == 24:
		if pos+1 > len(data) {
			return 0, 0, errCBOR
		}
		return uint64(data[pos]), 1, nil
	case info == 25:
		if pos+2 > len(data) {
			return 0, 0, errCBOR
		}
		return uint64(binary.BigEndian.Uint16(data[pos:])), 2, nil
	case info == 26:
		if pos+4 > len(data) {
			return 0, 0, errCBOR
		}
		return uint64(binary.BigEndian.Uint32(data[pos:])), 4, nil
	case info == 27:
		if pos+8 > len(data) {
			return 0, 0, errCBOR
		}
		return binary.BigEndian.Uint64(data[pos:]), 8, nil
	}
	// 28-30 保留，31 为不定长
	return 0, 0, errCBOR
}
