package ctl

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

const specJSON = `{
 "paths": {
  "/api/status": {"get": {"summary": "状态", "x-required-scope": "status:read", "x-token-allowed": true}},
  "/api/ddns/tasks": {
    "get": {"summary": "任务列表", "tags": ["DDNS"], "x-required-scope": "ddns:read", "x-token-allowed": true},
    "post": {"summary": "新增任务", "x-required-scope": "ddns:write", "x-token-allowed": true,
      "requestBody": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/Task"}}}}}
  },
  "/api/ddns/tasks/{key}": {"delete": {"summary": "删除任务", "x-required-scope": "ddns:write", "x-token-allowed": true}},
  "/api/account": {"put": {"summary": "改密码", "x-required-scope": "!denied", "x-token-allowed": false}}
 },
 "components": {"schemas": {"Task": {"type": "object", "properties": {"name": {"type": "string"}, "inner": {"$ref": "#/components/schemas/Inner"}}},
   "Inner": {"type": "object", "properties": {"x": {"type": "integer"}}}}}
}`

type call struct{ method, path, auth, body string }

func mockServer(t *testing.T, calls *[]call) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*calls = append(*calls, call{r.Method, r.URL.RequestURI(), r.Header.Get("Authorization"), string(b)})
		switch {
		case r.URL.Path == "/api/openapi.json":
			_, _ = w.Write([]byte(specJSON))
		case r.URL.Path == "/api/ddns/tasks/boom":
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"error":"任务不存在"}`))
		default:
			_, _ = w.Write([]byte(`{"ok":true}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestParseOperations(t *testing.T) {
	ops, err := ParseOperations([]byte(specJSON))
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 4 {
		t.Fatalf("应过滤掉令牌不可访问的接口，实际 %d 个", len(ops))
	}
	var post *Operation
	for i := range ops {
		if ops[i].Method == "POST" {
			post = &ops[i]
		}
	}
	body, _ := json.Marshal(post.Body)
	if !post.HasBody || !strings.Contains(string(body), `"x":{"type":"integer"}`) {
		t.Fatalf("请求体 schema 应展开引用: %s", body)
	}
}

func TestParseRealSpec(t *testing.T) {
	data, err := os.ReadFile("../../docs/api/openapi.json")
	if err != nil {
		t.Skip("尚未生成 docs/api/openapi.json（make api-docs）")
	}
	ops, err := ParseOperations(data)
	if err != nil || len(ops) < 50 {
		t.Fatalf("真实文档解析异常: %d %v", len(ops), err)
	}
	seen := map[string]bool{}
	for _, tl := range BuildTools(ops, MCPOptions{AllowWrite: true}) {
		if len(tl.Name) > 64 || seen[tl.Name] || strings.ContainsAny(tl.Name, " /{}") {
			t.Fatalf("工具名不合法或重复: %q", tl.Name)
		}
		seen[tl.Name] = true
	}
}

func TestBuildToolsWriteGating(t *testing.T) {
	ops, _ := ParseOperations([]byte(specJSON))
	ro := BuildTools(ops, MCPOptions{})
	for _, tl := range ro {
		if tl.op.Method != "GET" {
			t.Fatalf("默认只应暴露只读工具: %s", tl.Name)
		}
	}
	all := BuildTools(ops, MCPOptions{AllowWrite: true})
	if len(all) != 4 || len(ro) != 2 {
		t.Fatalf("工具数量不符: ro=%d all=%d", len(ro), len(all))
	}
	names := map[string]bool{}
	for _, tl := range all {
		names[tl.Name] = true
	}
	for _, want := range []string{"get_status", "get_ddns_tasks", "post_ddns_tasks", "delete_ddns_tasks_key"} {
		if !names[want] {
			t.Errorf("缺少工具 %s: %v", want, names)
		}
	}
}

func rpc(t *testing.T, c *Client, opts MCPOptions, msgs ...string) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	in := strings.NewReader(strings.Join(msgs, "\n") + "\n")
	if err := ServeMCP(context.Background(), c, opts, in, &out, io.Discard); err != nil {
		t.Fatal(err)
	}
	var res []map[string]any
	dec := json.NewDecoder(&out)
	for dec.More() {
		var m map[string]any
		if err := dec.Decode(&m); err != nil {
			t.Fatal(err)
		}
		res = append(res, m)
	}
	return res
}

func TestMCPSession(t *testing.T) {
	var calls []call
	srv := mockServer(t, &calls)
	c, _ := NewClient(Config{URL: srv.URL, Token: "gtl_abc"})
	res := rpc(t, c, MCPOptions{AllowWrite: true},
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"delete_ddns_tasks_key","arguments":{"key":"a/b c"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"post_ddns_tasks","arguments":{"body":{"name":"n"}}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"delete_ddns_tasks_key","arguments":{"key":"boom"}}}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"delete_ddns_tasks_key","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"nope"}}`,
		`{"jsonrpc":"2.0","id":8,"method":"bogus"}`,
		`not json`,
	)
	if len(res) != 9 {
		t.Fatalf("通知不应有回复，应有 9 条响应，实际 %d: %v", len(res), res)
	}
	if res[0]["result"].(map[string]any)["serverInfo"].(map[string]any)["name"] != "gately" {
		t.Fatalf("initialize 响应不符: %v", res[0])
	}
	tools := res[1]["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 4 {
		t.Fatalf("tools/list 应有 4 个工具: %d", len(tools))
	}
	// 路径参数被转义，令牌随请求发送
	var del call
	for _, cl := range calls {
		if cl.method == "DELETE" {
			del = cl
			break
		}
	}
	if del.path != "/api/ddns/tasks/a%2Fb%20c" || del.auth != "Bearer gtl_abc" {
		t.Fatalf("删除请求不符: %+v", del)
	}
	var post call
	for _, cl := range calls {
		if cl.method == "POST" {
			post = cl
		}
	}
	if post.body != `{"name":"n"}` {
		t.Fatalf("POST 请求体不符: %q", post.body)
	}
	isErr := func(r map[string]any) bool {
		v, _ := r["result"].(map[string]any)["isError"].(bool)
		return v
	}
	if isErr(res[3]) {
		t.Fatalf("POST 应成功: %v", res[3])
	}
	if !isErr(res[4]) || !strings.Contains(res[4]["result"].(map[string]any)["content"].([]any)[0].(map[string]any)["text"].(string), "任务不存在") {
		t.Fatalf("服务端错误应作为工具错误返回: %v", res[4])
	}
	if !isErr(res[5]) {
		t.Fatal("缺少路径参数应返回工具错误")
	}
	if res[6]["error"] == nil || res[7]["error"] == nil || res[8]["error"] == nil {
		t.Fatal("未知工具、未知方法、坏 JSON 应返回 JSON-RPC 错误")
	}
}

func TestMCPReadOnlyRejectsWrite(t *testing.T) {
	var calls []call
	srv := mockServer(t, &calls)
	c, _ := NewClient(Config{URL: srv.URL})
	res := rpc(t, c, MCPOptions{},
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"delete_ddns_tasks_key","arguments":{"key":"a"}}}`)
	if res[0]["error"] == nil {
		t.Fatal("只读模式下写工具不应存在")
	}
	for _, cl := range calls {
		if cl.method == "DELETE" {
			t.Fatal("只读模式不应发出写请求")
		}
	}
}

func TestCLI(t *testing.T) {
	var calls []call
	srv := mockServer(t, &calls)
	var out, errb bytes.Buffer
	if code := RunCtl([]string{"-url", srv.URL, "-token", "t", "get", "/api/status"}, &out, &errb); code != 0 || !strings.Contains(out.String(), `"ok": true`) {
		t.Fatalf("get 失败 %d: %s %s", code, out.String(), errb.String())
	}
	out.Reset()
	if code := RunCtl([]string{"-url", srv.URL, "post", "/api/ddns/tasks", `{"name":"x"}`}, &out, &errb); code != 0 {
		t.Fatalf("post 失败 %d: %s", code, errb.String())
	}
	if last := calls[len(calls)-1]; last.method != "POST" || last.body != `{"name":"x"}` {
		t.Fatalf("请求不符: %+v", last)
	}
	out.Reset()
	if code := RunCtl([]string{"-url", srv.URL, "ls", "ddns"}, &out, &errb); code != 0 || !strings.Contains(out.String(), "/api/ddns/tasks") || strings.Contains(out.String(), "/api/status") {
		t.Fatalf("ls 过滤失败 %d: %s", code, out.String())
	}
	errb.Reset()
	if code := RunCtl([]string{"-url", srv.URL, "delete", "/api/ddns/tasks/boom"}, &out, &errb); code != 1 || !strings.Contains(errb.String(), "任务不存在") {
		t.Fatalf("错误应返回 1: %d %s", code, errb.String())
	}
	if code := RunCtl([]string{"-url", srv.URL, "get", "/etc/passwd"}, &out, &errb); code != 1 {
		t.Fatal("非 /api/ 路径应被拒绝")
	}
	if code := RunCtl([]string{"-url", srv.URL, "post", "/api/x", "{bad"}, &out, &errb); code != 2 {
		t.Fatal("非法 JSON 应返回 2")
	}
	if code := RunCtl([]string{"-url", "ftp://x", "ls"}, &out, &errb); code != 2 {
		t.Fatal("非法地址应返回 2")
	}
}

// 需要写权限的 GET 接口（如取回密码明文的 secrets）在默认只读模式下不应暴露，
// 开启 --write 后暴露但不标记为只读。
func TestBuildToolsHidesSecretsWithoutWrite(t *testing.T) {
	spec := `{"paths": {
  "/api/ddns/tasks": {"get": {"summary": "任务列表", "x-required-scope": "ddns:read", "x-token-allowed": true}},
  "/api/ddns/tasks/{key}/secrets": {"get": {"summary": "查看密钥", "x-required-scope": "ddns:write", "x-token-allowed": true}}
}}`
	ops, err := ParseOperations([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	ro := BuildTools(ops, MCPOptions{})
	if len(ro) != 1 || ro[0].op.Path != "/api/ddns/tasks" {
		t.Fatalf("只读模式不应暴露 secrets 接口: %+v", ro)
	}
	all := BuildTools(ops, MCPOptions{AllowWrite: true})
	if len(all) != 2 {
		t.Fatalf("写模式应暴露 2 个工具: %d", len(all))
	}
	for _, tl := range all {
		hint, _ := tl.Annotations["readOnlyHint"].(bool)
		if tl.op.Path == "/api/ddns/tasks/{key}/secrets" && hint {
			t.Fatal("secrets 接口不应标记为只读")
		}
	}
}
