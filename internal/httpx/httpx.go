// Package httpx 提供管理 API 共用的 JSON 读写与错误响应。
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

// 请求体上限，防止恶意大包占满内存
const maxBodyBytes = 4 << 20

// ErrorBody 是统一的错误响应结构。
type ErrorBody struct {
	Error string `json:"error"`
}

// WriteJSON 以指定状态码输出 JSON。
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

// OK 输出 200 JSON。
func OK(w http.ResponseWriter, v any) {
	WriteJSON(w, http.StatusOK, v)
}

// Error 输出错误 JSON。
func Error(w http.ResponseWriter, status int, format string, args ...any) {
	WriteJSON(w, status, ErrorBody{Error: fmt.Sprintf(format, args...)})
}

// ReadJSON 解析请求体到 v，拒绝未知字段以尽早暴露前后端不一致。
func ReadJSON(r *http.Request, v any) error {
	body := http.MaxBytesReader(nil, r.Body, maxBodyBytes)
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("请求体为空")
		}
		return fmt.Errorf("请求体格式错误: %w", err)
	}
	return nil
}

// ClientIP 返回请求的对端 IP（不信任转发头，管理端口直接面向用户）。
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return strings.Trim(host, "[]")
}
