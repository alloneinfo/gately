// Package httpx 提供管理 API 共用的 JSON 读写与错误响应。
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
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
// 要求 Content-Type 为 application/json：浏览器跨源发送该类型会触发预检，可阻止表单类 CSRF。
func ReadJSON(r *http.Request, v any) error {
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		return errors.New("Content-Type 必须为 application/json")
	}
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

// SameOrigin 拒绝来自其他源的状态变更请求（同站不同端口、兄弟子域同样被拒绝）。
// 浏览器会为跨源请求带上 Origin 或 Sec-Fetch-Site；命令行工具通常两者都不带，予以放行。
func SameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || !strings.EqualFold(u.Host, r.Host) {
				Error(w, http.StatusForbidden, "拒绝跨源请求")
				return
			}
		} else if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
			Error(w, http.StatusForbidden, "拒绝跨源请求")
			return
		}
		next.ServeHTTP(w, r)
	})
}
