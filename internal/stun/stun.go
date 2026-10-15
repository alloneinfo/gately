// Package stun 实现 RFC 5389 的 Binding 请求与响应，用于获取 NAT 映射后的公网地址。
//
// 只实现穿透所需的最小子集：Binding Request、XOR-MAPPED-ADDRESS / MAPPED-ADDRESS 解析，
// 支持 TCP（消息按长度自定界）与 UDP。另提供一个简单服务端，供测试与局域网自检使用。
package stun

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
)

const (
	headerSize  = 20
	magicCookie = 0x2112A442

	typeBindingRequest  = 0x0001
	typeBindingResponse = 0x0101
	typeBindingError    = 0x0111

	attrMappedAddress    = 0x0001
	attrXorMappedAddress = 0x0020
	// 部分老服务器使用的非标准值
	attrXorMappedAddressOld = 0x8020

	// 单条消息上限，防止恶意长度字段
	maxMessageSize = 2048
)

// TxID 是 96 位事务 ID。
type TxID [12]byte

// NewBindingRequest 生成 Binding 请求及其事务 ID。
func NewBindingRequest() ([]byte, TxID) {
	var id TxID
	_, _ = rand.Read(id[:])
	msg := make([]byte, headerSize)
	binary.BigEndian.PutUint16(msg[0:], typeBindingRequest)
	binary.BigEndian.PutUint16(msg[2:], 0)
	binary.BigEndian.PutUint32(msg[4:], magicCookie)
	copy(msg[8:], id[:])
	return msg, id
}

// IsMessage 粗略判断数据是否为 STUN 消息（前两位为 0 且魔数匹配），用于 UDP 收包分流。
func IsMessage(b []byte) bool {
	return len(b) >= headerSize && b[0]&0xC0 == 0 && binary.BigEndian.Uint32(b[4:]) == magicCookie
}

// MessageTxID 返回消息中的事务 ID。
func MessageTxID(b []byte) (TxID, bool) {
	var id TxID
	if !IsMessage(b) {
		return id, false
	}
	copy(id[:], b[8:20])
	return id, true
}

// ParseBindingResponse 解析 Binding 成功响应，返回映射地址。
func ParseBindingResponse(b []byte, want TxID) (netip.AddrPort, error) {
	if !IsMessage(b) {
		return netip.AddrPort{}, errors.New("不是 STUN 消息")
	}
	msgType := binary.BigEndian.Uint16(b[0:])
	length := int(binary.BigEndian.Uint16(b[2:]))
	if headerSize+length > len(b) {
		return netip.AddrPort{}, errors.New("STUN 消息长度不完整")
	}
	if !equalTx(b[8:20], want) {
		return netip.AddrPort{}, errors.New("事务 ID 不匹配")
	}
	if msgType == typeBindingError {
		return netip.AddrPort{}, errors.New("服务器返回 Binding 错误响应")
	}
	if msgType != typeBindingResponse {
		return netip.AddrPort{}, fmt.Errorf("意外的消息类型 0x%04x", msgType)
	}
	var mapped netip.AddrPort
	attrs := b[headerSize : headerSize+length]
	for len(attrs) >= 4 {
		at := binary.BigEndian.Uint16(attrs[0:])
		al := int(binary.BigEndian.Uint16(attrs[2:]))
		if 4+al > len(attrs) {
			break
		}
		val := attrs[4 : 4+al]
		switch at {
		case attrXorMappedAddress, attrXorMappedAddressOld:
			if ap, err := decodeAddress(val, true, b[4:20]); err == nil {
				// XOR-MAPPED-ADDRESS 优先
				return ap, nil
			}
		case attrMappedAddress:
			if ap, err := decodeAddress(val, false, nil); err == nil {
				mapped = ap
			}
		}
		// 属性按 4 字节对齐
		step := 4 + (al+3)&^3
		if step > len(attrs) {
			break
		}
		attrs = attrs[step:]
	}
	if mapped.IsValid() {
		return mapped, nil
	}
	return netip.AddrPort{}, errors.New("响应中没有映射地址")
}

func equalTx(b []byte, id TxID) bool {
	for i := range id {
		if b[i] != id[i] {
			return false
		}
	}
	return true
}

// decodeAddress 解析地址属性。xorKey 为 魔数(4) + 事务 ID(12)。
func decodeAddress(v []byte, xor bool, xorKey []byte) (netip.AddrPort, error) {
	if len(v) < 8 {
		return netip.AddrPort{}, errors.New("地址属性过短")
	}
	family := v[1]
	port := binary.BigEndian.Uint16(v[2:])
	if xor {
		port ^= uint16(magicCookie >> 16)
	}
	switch family {
	case 0x01:
		ip := make([]byte, 4)
		copy(ip, v[4:8])
		if xor {
			for i := range ip {
				ip[i] ^= xorKey[i]
			}
		}
		a, _ := netip.AddrFromSlice(ip)
		return netip.AddrPortFrom(a, port), nil
	case 0x02:
		if len(v) < 20 {
			return netip.AddrPort{}, errors.New("IPv6 地址属性过短")
		}
		ip := make([]byte, 16)
		copy(ip, v[4:20])
		if xor {
			for i := range ip {
				ip[i] ^= xorKey[i]
			}
		}
		a, _ := netip.AddrFromSlice(ip)
		return netip.AddrPortFrom(a, port), nil
	}
	return netip.AddrPort{}, fmt.Errorf("未知地址族 %d", family)
}

// EncodeBindingResponse 构造带 XOR-MAPPED-ADDRESS 的成功响应（服务端与测试使用）。
func EncodeBindingResponse(id TxID, addr netip.AddrPort) []byte {
	ip := addr.Addr().Unmap()
	var val []byte
	key := make([]byte, 16)
	binary.BigEndian.PutUint32(key, magicCookie)
	copy(key[4:], id[:])
	if ip.Is4() {
		val = make([]byte, 8)
		val[1] = 0x01
		a := ip.As4()
		for i := 0; i < 4; i++ {
			val[4+i] = a[i] ^ key[i]
		}
	} else {
		val = make([]byte, 20)
		val[1] = 0x02
		a := ip.As16()
		for i := 0; i < 16; i++ {
			val[4+i] = a[i] ^ key[i]
		}
	}
	binary.BigEndian.PutUint16(val[2:], addr.Port()^uint16(magicCookie>>16))

	msg := make([]byte, headerSize+4+len(val))
	binary.BigEndian.PutUint16(msg[0:], typeBindingResponse)
	binary.BigEndian.PutUint16(msg[2:], uint16(4+len(val)))
	binary.BigEndian.PutUint32(msg[4:], magicCookie)
	copy(msg[8:], id[:])
	binary.BigEndian.PutUint16(msg[20:], attrXorMappedAddress)
	binary.BigEndian.PutUint16(msg[22:], uint16(len(val)))
	copy(msg[24:], val)
	return msg
}

// ReadMessage 从流式连接中读取一条完整 STUN 消息。
func ReadMessage(r io.Reader) ([]byte, error) {
	head := make([]byte, headerSize)
	if _, err := io.ReadFull(r, head); err != nil {
		return nil, err
	}
	if !IsMessage(head) {
		return nil, errors.New("对端返回的不是 STUN 消息")
	}
	length := int(binary.BigEndian.Uint16(head[2:]))
	if headerSize+length > maxMessageSize {
		return nil, errors.New("STUN 消息过长")
	}
	msg := make([]byte, headerSize+length)
	copy(msg, head)
	if _, err := io.ReadFull(r, msg[headerSize:]); err != nil {
		return nil, err
	}
	return msg, nil
}

// BindingTCP 在已建立的 TCP 连接上执行一次 Binding，调用方负责设置超时。
func BindingTCP(conn net.Conn) (netip.AddrPort, error) {
	req, id := NewBindingRequest()
	if _, err := conn.Write(req); err != nil {
		return netip.AddrPort{}, err
	}
	resp, err := ReadMessage(conn)
	if err != nil {
		return netip.AddrPort{}, err
	}
	return ParseBindingResponse(resp, id)
}
