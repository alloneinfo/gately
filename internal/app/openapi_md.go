package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/wangwanjie/gately/internal/apitoken"
)

// markdown 把文档模型渲染为 Markdown，供官网直接发布。
func (m docModel) markdown() string {
	var b strings.Builder
	b.WriteString("# Gately 管理 API\n\n")
	b.WriteString(strings.ReplaceAll(apiIntro, "\n\n", "\n\n"))
	b.WriteString("\n\n官网：" + websiteURL + "\n\n")
	b.WriteString("机器可读的描述见同目录的 `openapi.json`（OpenAPI 3.0）。\n\n")

	b.WriteString("## 快速开始\n\n```bash\n")
	b.WriteString("# 查看系统状态（需要 status:read）\n")
	b.WriteString("curl -H \"Authorization: Bearer gtl_xxxx\" http://localhost:16680/api/status\n\n")
	b.WriteString("# 列出全部 DDNS 任务（需要 ddns:read）\n")
	b.WriteString("curl -H \"Authorization: Bearer gtl_xxxx\" http://localhost:16680/api/ddns/tasks\n\n")
	b.WriteString("# 立即同步一个 DDNS 任务（需要 ddns:write，key 取自上一步返回的列表）\n")
	b.WriteString("curl -X POST -H \"Authorization: Bearer gtl_xxxx\" http://localhost:16680/api/ddns/tasks/<key>/sync\n")
	b.WriteString("```\n\n")

	b.WriteString("## 权限\n\n")
	b.WriteString("| 权限域 | 名称 | 可用权限 |\n| --- | --- | --- |\n")
	for _, d := range m.Domains {
		levels := fmt.Sprintf("`%s:read`、`%s:write`", d.ID, d.ID)
		if d.WriteOnly {
			levels = fmt.Sprintf("`%s:write`", d.ID)
		}
		if d.ReadOnly {
			levels = fmt.Sprintf("`%s:read`", d.ID)
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s |\n", d.ID, d.Title, levels)
	}
	b.WriteString("\n通配写法：`*:read` 表示全部模块只读，`*:write` 或 `*` 表示全部模块读写。\n\n")

	for _, tag := range m.Tags {
		fmt.Fprintf(&b, "## %s\n\n| 方法 | 路径 | 所需权限 | 说明 |\n| --- | --- | --- | --- |\n", tag)
		for _, e := range m.Endpoints {
			if e.Tag != tag {
				continue
			}
			scope := "`" + e.Scope + "`"
			switch e.Scope {
			case apitoken.Denied:
				scope = "仅登录会话"
			case apitoken.Any:
				scope = "任意有效令牌"
			}
			fmt.Fprintf(&b, "| %s | `%s` | %s | %s |\n", e.Method, e.Path, scope, e.Summary)
		}
		b.WriteString("\n")
	}

	if len(m.Schemas) > 0 {
		b.WriteString("## 数据结构\n\n")
		names := make([]string, 0, len(m.Schemas))
		for n := range m.Schemas {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Fprintf(&b, "### %s\n\n| 字段 | 类型 |\n| --- | --- |\n", n)
			props, _ := m.Schemas[n]["properties"].(map[string]any)
			keys := make([]string, 0, len(props))
			for k := range props {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(&b, "| `%s` | %s |\n", k, schemaLabel(props[k]))
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

// schemaLabel 返回 schema 的简短类型描述。
func schemaLabel(v any) string {
	sc, _ := v.(map[string]any)
	if ref, ok := sc["$ref"].(string); ok {
		return "`" + ref[strings.LastIndex(ref, "/")+1:] + "`"
	}
	switch t, _ := sc["type"].(string); t {
	case "array":
		return "数组 of " + schemaLabel(sc["items"])
	case "object":
		if ap, ok := sc["additionalProperties"]; ok {
			return "映射 of " + schemaLabel(ap)
		}
		return "对象"
	case "":
		return "任意"
	default:
		if f, _ := sc["format"].(string); f == "date-time" {
			return "时间 (RFC 3339)"
		}
		return t
	}
}
