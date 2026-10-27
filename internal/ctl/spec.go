package ctl

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Operation 是 OpenAPI 文档中的一个接口。
type Operation struct {
	Method  string
	Path    string
	Tag     string
	Summary string
	Desc    string
	// Scope 为所需令牌权限（如 ddns:write），空表示任意令牌
	Scope string
	// PathParams 为路径中的参数名
	PathParams []string
	// Body 为请求体的 JSON Schema（已展开引用），nil 表示无请求体
	Body any
	// HasBody 表示接口接受请求体（Body 可能为空对象）
	HasBody bool
}

// IsRead 判断是否为只读接口。
func (o Operation) IsRead() bool { return o.Method == "GET" || o.Method == "HEAD" }

// IsReadOnly 判断接口是否真正只读：GET 但需要写权限的接口（取回密码明文的 secrets、
// 容器 exec、私钥下载等）不算只读，MCP 默认只读模式下不暴露。
func (o Operation) IsReadOnly() bool {
	return o.IsRead() && !strings.HasSuffix(o.Scope, ":write")
}

var pathParamRe = regexp.MustCompile(`\{([^}/]+)\}`)

// FetchOperations 从实例读取 OpenAPI 文档，返回令牌可访问的接口。
func (c *Client) FetchOperations(ctx context.Context) ([]Operation, error) {
	data, err := c.Do(ctx, "GET", "/api/openapi.json", nil)
	if err != nil {
		return nil, err
	}
	return ParseOperations(data)
}

// ParseOperations 解析 OpenAPI JSON。
func ParseOperations(data []byte) ([]Operation, error) {
	var doc struct {
		Paths      map[string]map[string]json.RawMessage `json:"paths"`
		Components struct {
			Schemas map[string]any `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("解析 OpenAPI 文档失败: %w", err)
	}
	var ops []Operation
	for path, item := range doc.Paths {
		for method, raw := range item {
			var op struct {
				Tags         []string `json:"tags"`
				Summary      string   `json:"summary"`
				Description  string   `json:"description"`
				Scope        string   `json:"x-required-scope"`
				TokenAllowed *bool    `json:"x-token-allowed"`
				RequestBody  *struct {
					Content map[string]struct {
						Schema any `json:"schema"`
					} `json:"content"`
				} `json:"requestBody"`
			}
			if err := json.Unmarshal(raw, &op); err != nil {
				continue
			}
			if op.TokenAllowed != nil && !*op.TokenAllowed {
				continue
			}
			o := Operation{Method: strings.ToUpper(method), Path: path, Summary: op.Summary, Desc: op.Description, Scope: op.Scope}
			if len(op.Tags) > 0 {
				o.Tag = op.Tags[0]
			}
			for _, m := range pathParamRe.FindAllStringSubmatch(path, -1) {
				o.PathParams = append(o.PathParams, m[1])
			}
			if op.RequestBody != nil {
				o.HasBody = true
				if js, ok := op.RequestBody.Content["application/json"]; ok {
					o.Body = resolveSchema(js.Schema, doc.Components.Schemas, 0)
				}
			}
			ops = append(ops, o)
		}
	}
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Path != ops[j].Path {
			return ops[i].Path < ops[j].Path
		}
		return ops[i].Method < ops[j].Method
	})
	return ops, nil
}

// resolveSchema 把 $ref 展开为内联 schema。refDepth 只统计引用展开的层数，超过上限的引用降级为通用对象，
// 以避免递归结构无限膨胀。
func resolveSchema(s any, defs map[string]any, refDepth int) any {
	switch v := s.(type) {
	case map[string]any:
		if ref, ok := v["$ref"].(string); ok {
			name := strings.TrimPrefix(ref, "#/components/schemas/")
			def, found := defs[name]
			if !found || refDepth >= 6 {
				return map[string]any{"type": "object"}
			}
			return resolveSchema(def, defs, refDepth+1)
		}
		out := make(map[string]any, len(v))
		for k, val := range v {
			out[k] = resolveSchema(val, defs, refDepth)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, val := range v {
			out[i] = resolveSchema(val, defs, refDepth)
		}
		return out
	}
	return s
}
