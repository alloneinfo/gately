// Package ctl 实现 Gately 的命令行客户端与 MCP 服务器：二者都通过 HTTP + API 令牌调用运行中的实例，
// 不重复任何业务逻辑，接口清单直接来自实例的 OpenAPI 文档。
package ctl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wangwanjie/gately/internal/i18n"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultURL 是未指定地址时连接的本机管理地址。
const DefaultURL = "http://127.0.0.1:16680"

// Config 是连接配置。
type Config struct {
	URL   string `json:"url"`
	Token string `json:"token"`
	// Insecure 跳过 HTTPS 证书校验，仅用于自签名证书的内网实例
	Insecure bool `json:"insecure,omitempty"`
}

// ConfigPath 返回配置文件路径（~/.config/gately/ctl.json 或系统等价目录）。
func ConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gately", "ctl.json")
}

// LoadConfig 依次读取配置文件、环境变量（GATELY_URL / GATELY_TOKEN），后者优先。
func LoadConfig() Config {
	var c Config
	if p := ConfigPath(); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			_ = json.Unmarshal(b, &c)
		}
	}
	if v := os.Getenv("GATELY_URL"); v != "" {
		c.URL = v
	}
	if v := os.Getenv("GATELY_TOKEN"); v != "" {
		c.Token = v
	}
	if c.URL == "" {
		c.URL = DefaultURL
	}
	return c
}

// Client 是调用管理 API 的客户端。
type Client struct {
	base  *url.URL
	token string
	http  *http.Client
}

// NewClient 创建客户端。
func NewClient(c Config) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(c.URL), "/"))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("地址 %q 无效，需为 http(s)://主机:端口", c.URL)
	}
	hc := &http.Client{Timeout: 60 * time.Second}
	if c.Insecure {
		hc.Transport = insecureTransport()
	}
	return &Client{base: u, token: c.Token, http: hc}, nil
}

// APIError 是服务端返回的错误。
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.Status, e.Message)
}

// Do 发送请求并返回响应体。path 以 / 开头，可包含查询串。非 2xx 状态返回 *APIError。
func (c *Client) Do(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	// 只允许访问管理 API，避免把令牌发往其他路径
	if !strings.HasPrefix(path, "/api/") {
		return nil, errors.New("路径需以 /api/ 开头")
	}
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), c.base.String()+path, rd)
	if err != nil {
		return nil, err
	}
	// 让服务端按命令行所在系统的语言返回错误说明与状态文案
	req.Header.Set("Accept-Language", i18n.FromEnv())
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(data))
		var e struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(data, &e) == nil && e.Error != "" {
			msg = e.Error
		}
		return data, &APIError{Status: resp.StatusCode, Message: msg}
	}
	return data, nil
}
