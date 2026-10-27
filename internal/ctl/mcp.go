package ctl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/wangwanjie/gately/internal/version"
)

// supportedProtocol 是实现所依据的 MCP 协议版本；客户端请求其他版本时仍返回它，由客户端决定是否兼容。
const supportedProtocol = "2024-11-05"

// MCPOptions 控制暴露哪些工具。
type MCPOptions struct {
	// AllowWrite 为 false 时只暴露只读（GET）接口
	AllowWrite bool
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// Tool 是暴露给 MCP 客户端的一个工具，对应一个管理接口。
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Annotations map[string]any `json:"annotations,omitempty"`

	op Operation
}

var nonAlnum = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// BuildTools 把接口清单转换为工具列表。
func BuildTools(ops []Operation, opts MCPOptions) []Tool {
	var tools []Tool
	used := map[string]bool{}
	for _, o := range ops {
		if !opts.AllowWrite && !o.IsReadOnly() {
			continue
		}
		name := toolName(o, used)
		props := map[string]any{}
		var required []string
		for _, p := range o.PathParams {
			props[p] = map[string]any{"type": "string", "description": "路径参数 " + p}
			required = append(required, p)
		}
		if o.IsRead() {
			props["query"] = map[string]any{"type": "string", "description": "可选的查询串，如 since=10"}
		}
		if o.HasBody {
			body := o.Body
			if body == nil {
				body = map[string]any{"type": "object"}
			}
			props["body"] = body
		}
		schema := map[string]any{"type": "object", "properties": props}
		if len(required) > 0 {
			schema["required"] = required
		}
		desc := o.Summary
		if desc == "" {
			desc = o.Method + " " + o.Path
		}
		desc += fmt.Sprintf("（%s %s", o.Method, o.Path)
		if o.Scope != "" {
			desc += "，权限 " + o.Scope
		}
		desc += "）"
		tools = append(tools, Tool{
			Name: name, Description: desc, InputSchema: schema, op: o,
			Annotations: map[string]any{"readOnlyHint": o.IsReadOnly(), "destructiveHint": o.Method == "DELETE"},
		})
	}
	return tools
}

// toolName 由方法与路径生成 ≤64 字符的工具名，如 get_ddns_tasks、put_ddns_tasks_key_enable。
func toolName(o Operation, used map[string]bool) string {
	p := strings.TrimPrefix(o.Path, "/api/")
	p = strings.NewReplacer("{", "", "}", "").Replace(p)
	base := strings.ToLower(o.Method) + "_" + strings.Trim(nonAlnum.ReplaceAllString(p, "_"), "_")
	if len(base) > 60 {
		base = base[:60]
	}
	name := base
	for i := 2; used[name]; i++ {
		name = fmt.Sprintf("%s_%d", base, i)
	}
	used[name] = true
	return name
}

// ServeMCP 在 r/w 上运行 MCP 服务（每行一个 JSON-RPC 消息），直到输入关闭。
func ServeMCP(ctx context.Context, c *Client, opts MCPOptions, r io.Reader, w io.Writer, logw io.Writer) error {
	s := &mcpServer{client: c, opts: opts, out: json.NewEncoder(w), log: logw}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.reply(nil, nil, &rpcError{Code: -32700, Message: "无法解析 JSON"})
			continue
		}
		s.handle(ctx, req)
	}
	return sc.Err()
}

type mcpServer struct {
	client *Client
	opts   MCPOptions
	out    *json.Encoder
	log    io.Writer
	tools  []Tool
	loaded bool
}

func (s *mcpServer) reply(id json.RawMessage, result any, e *rpcError) {
	if id == nil {
		id = json.RawMessage("null")
	}
	_ = s.out.Encode(rpcResponse{JSONRPC: "2.0", ID: id, Result: result, Error: e})
}

// loadTools 首次需要时从实例读取接口清单；失败时不缓存，下次重试。
func (s *mcpServer) loadTools(ctx context.Context) error {
	if s.loaded {
		return nil
	}
	ops, err := s.client.FetchOperations(ctx)
	if err != nil {
		return err
	}
	s.tools, s.loaded = BuildTools(ops, s.opts), true
	return nil
}

func (s *mcpServer) handle(ctx context.Context, req rpcRequest) {
	// 没有 id 的是通知，不回复
	isNotification := len(req.ID) == 0
	switch req.Method {
	case "initialize":
		s.reply(req.ID, map[string]any{
			"protocolVersion": supportedProtocol,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "gately", "version": version.Version},
			"instructions":    "通过 Gately 管理 API 查看和管理网关。写操作默认关闭，需以 --write 启动。",
		}, nil)
	case "ping":
		s.reply(req.ID, map[string]any{}, nil)
	case "tools/list":
		if err := s.loadTools(ctx); err != nil {
			s.reply(req.ID, nil, &rpcError{Code: -32603, Message: "读取接口清单失败: " + err.Error()})
			return
		}
		list := make([]Tool, len(s.tools))
		copy(list, s.tools)
		s.reply(req.ID, map[string]any{"tools": list}, nil)
	case "tools/call":
		s.call(ctx, req)
	default:
		if !isNotification {
			s.reply(req.ID, nil, &rpcError{Code: -32601, Message: "不支持的方法 " + req.Method})
		}
	}
}

func textResult(text string, isErr bool) map[string]any {
	r := map[string]any{"content": []map[string]any{{"type": "text", "text": text}}}
	if isErr {
		r["isError"] = true
	}
	return r
}

func (s *mcpServer) call(ctx context.Context, req rpcRequest) {
	var p struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil {
		s.reply(req.ID, nil, &rpcError{Code: -32602, Message: "参数无效"})
		return
	}
	if err := s.loadTools(ctx); err != nil {
		s.reply(req.ID, textResult("读取接口清单失败: "+err.Error(), true), nil)
		return
	}
	var tool *Tool
	for i := range s.tools {
		if s.tools[i].Name == p.Name {
			tool = &s.tools[i]
			break
		}
	}
	if tool == nil {
		s.reply(req.ID, nil, &rpcError{Code: -32602, Message: "未知工具 " + p.Name})
		return
	}
	path := tool.op.Path
	for _, name := range tool.op.PathParams {
		v, _ := p.Arguments[name].(string)
		if v == "" {
			s.reply(req.ID, textResult("缺少路径参数 "+name, true), nil)
			return
		}
		path = strings.ReplaceAll(path, "{"+name+"}", url.PathEscape(v))
	}
	if q, _ := p.Arguments["query"].(string); q != "" && tool.op.IsRead() {
		path += "?" + strings.TrimPrefix(q, "?")
	}
	var body []byte
	if b, ok := p.Arguments["body"]; ok && tool.op.HasBody {
		body, _ = json.Marshal(b)
	}
	data, err := s.client.Do(ctx, tool.op.Method, path, body)
	if err != nil {
		msg := err.Error()
		if len(data) > 0 && len(data) < 4096 {
			msg += "\n" + string(data)
		}
		s.reply(req.ID, textResult(msg, true), nil)
		return
	}
	text := string(data)
	if len(text) == 0 {
		text = "OK"
	}
	// 过长的输出截断，避免撑爆模型上下文
	const maxText = 200000
	if len(text) > maxText {
		text = text[:maxText] + "\n…（输出过长已截断）"
	}
	s.reply(req.ID, textResult(text, false), nil)
}

// RunMCP 执行 `gately mcp` 子命令。
func RunMCP(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	cfg := LoadConfig()
	url, token, insecure, write := cfg.URL, cfg.Token, cfg.Insecure, false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--write", "-write":
			write = true
		case "--insecure", "-insecure":
			insecure = true
		case "--url", "-url":
			i++
			if i < len(args) {
				url = args[i]
			}
		case "--token", "-token":
			i++
			if i < len(args) {
				token = args[i]
			}
		case "-h", "--help":
			fmt.Fprintln(stderr, "用法：gately mcp [--url 地址] [--token 令牌] [--write] [--insecure]\n通过 stdio 提供 MCP 服务。默认只暴露只读工具，--write 同时暴露写操作（令牌权限仍然生效）。")
			return 0
		default:
			fmt.Fprintf(stderr, "未知参数 %q\n", args[i])
			return 2
		}
	}
	client, err := NewClient(Config{URL: url, Token: token, Insecure: insecure})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := ServeMCP(context.Background(), client, MCPOptions{AllowWrite: write}, stdin, stdout, stderr); err != nil {
		fmt.Fprintln(stderr, "MCP 服务退出:", err)
		return 1
	}
	return 0
}
