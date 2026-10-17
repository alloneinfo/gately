package ctl

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

const ctlUsage = `用法：
  gately ctl [连接选项] ls [关键字]                列出令牌可访问的接口
  gately ctl [连接选项] get   <路径>               GET 请求，如 get /api/status
  gately ctl [连接选项] post  <路径> [JSON|@文件]   POST 请求
  gately ctl [连接选项] put   <路径> [JSON|@文件]   PUT 请求
  gately ctl [连接选项] delete <路径> [JSON|@文件]  DELETE 请求
  gately ctl [连接选项] login <地址> <令牌>         保存连接配置到 %s

连接选项：
  -url <地址>     管理地址，默认读取 GATELY_URL，其次配置文件，最后 ` + DefaultURL + `
  -token <令牌>   API 令牌，默认读取 GATELY_TOKEN 或配置文件
  -insecure       跳过 HTTPS 证书校验
  -raw            直接输出响应原文（默认格式化 JSON）

示例：
  gately ctl get /api/ddns/tasks
  gately ctl post /api/wol/devices/abc123/wake
  gately ctl put /api/ddns/tasks/abc/enable '{"enable":false}'
  gately mcp                                       启动 MCP 服务（stdio）
`

// RunCtl 执行 `gately ctl` 子命令，返回退出码。
func RunCtl(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ctl", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cfg := LoadConfig()
	url := fs.String("url", cfg.URL, "")
	token := fs.String("token", cfg.Token, "")
	insecure := fs.Bool("insecure", cfg.Insecure, "")
	raw := fs.Bool("raw", false, "")
	fs.Usage = func() { fmt.Fprintf(stderr, ctlUsage, ConfigPath()) }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	rest := fs.Args()
	if len(rest) == 0 {
		fs.Usage()
		return 2
	}
	cmd, rest := strings.ToLower(rest[0]), rest[1:]

	if cmd == "login" {
		return runLogin(rest, *insecure, stdout, stderr)
	}
	client, err := NewClient(Config{URL: *url, Token: *token, Insecure: *insecure})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	ctx := context.Background()
	switch cmd {
	case "ls", "list":
		filter := ""
		if len(rest) > 0 {
			filter = rest[0]
		}
		return runList(ctx, client, filter, stdout, stderr)
	case "get", "post", "put", "delete", "patch":
		if len(rest) == 0 {
			fmt.Fprintf(stderr, "缺少路径\n")
			return 2
		}
		var body []byte
		if len(rest) > 1 {
			body, err = readBody(rest[1])
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 2
			}
		}
		return runCall(ctx, client, strings.ToUpper(cmd), rest[0], body, *raw, stdout, stderr)
	}
	fmt.Fprintf(stderr, "未知命令 %q\n\n", cmd)
	fs.Usage()
	return 2
}

// readBody 读取请求体：以 @ 开头表示文件，- 表示标准输入，否则视为 JSON 字面量。
func readBody(arg string) ([]byte, error) {
	var b []byte
	var err error
	switch {
	case arg == "-":
		b, err = io.ReadAll(os.Stdin)
	case strings.HasPrefix(arg, "@"):
		b, err = os.ReadFile(arg[1:])
	default:
		b = []byte(arg)
	}
	if err != nil {
		return nil, err
	}
	if !json.Valid(b) {
		return nil, errors.New("请求体不是有效的 JSON")
	}
	return b, nil
}

func runCall(ctx context.Context, c *Client, method, path string, body []byte, raw bool, stdout, stderr io.Writer) int {
	data, err := c.Do(ctx, method, path, body)
	if err != nil {
		fmt.Fprintln(stderr, "错误:", err)
		return 1
	}
	if len(data) == 0 {
		return 0
	}
	if !raw {
		var v any
		if json.Unmarshal(data, &v) == nil {
			if pretty, err := json.MarshalIndent(v, "", "  "); err == nil {
				data = pretty
			}
		}
	}
	fmt.Fprintln(stdout, strings.TrimRight(string(data), "\n"))
	return 0
}

func runList(ctx context.Context, c *Client, filter string, stdout, stderr io.Writer) int {
	ops, err := c.FetchOperations(ctx)
	if err != nil {
		fmt.Fprintln(stderr, "错误:", err)
		return 1
	}
	tw := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	for _, o := range ops {
		line := o.Method + "\t" + o.Path + "\t" + o.Summary
		if filter != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(filter)) {
			continue
		}
		fmt.Fprintln(tw, line)
	}
	_ = tw.Flush()
	return 0
}

func runLogin(args []string, insecure bool, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "用法：gately ctl login <地址> <令牌>")
		return 2
	}
	cfg := Config{URL: args[0], Token: args[1], Insecure: insecure}
	client, err := NewClient(cfg)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if _, err := client.Do(context.Background(), "GET", "/api/session", nil); err != nil {
		fmt.Fprintln(stderr, "验证失败:", err)
		return 1
	}
	p := ConfigPath()
	if p == "" {
		fmt.Fprintln(stderr, "无法确定配置目录")
		return 1
	}
	if err := os.MkdirAll(dirOf(p), 0o700); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	b, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(p, append(b, '\n'), 0o600); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "已保存连接配置到", p)
	return 0
}

func dirOf(p string) string {
	if i := strings.LastIndexAny(p, `/\`); i > 0 {
		return p[:i]
	}
	return "."
}
