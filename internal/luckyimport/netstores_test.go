package luckyimport

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestConvertFileUsers 覆盖 Lucky 真实配置中的写法：同一用户名重复出现、每次挂载一个目录。
func TestConvertFileUsers(t *testing.T) {
	raw := json.RawMessage(`[
	  {"Username":"VanJay","Password":"pw","MountList":[{"Type":"local","Param":"/data","DisplayName":"data","Writable":true}]},
	  {"Username":"VanJay","Password":"pw","MountList":[{"Type":"local","Param":"/etc/nginx","DisplayName":"nginx","Writable":false}]},
	  {"Username":"VanJay","Password":"pw","MountList":[{"Type":"local","Param":"/srv/data","DisplayName":"data","Writable":true}]},
	  {"Username":"VanJay","Password":"other","MountList":[{"Type":"local","Param":"/x","DisplayName":"x","Writable":true}]},
	  {"Username":"guest","Password":"g","MountList":[
	    {"Type":"store","Param":"k1","DisplayName":"","Writable":true},
	    {"Type":"store","Param":"k2","DisplayName":"cloud","Writable":true},
	    {"Type":"rclone","Param":"r1","DisplayName":"drive","Writable":true}]},
	  {"Username":"empty","Password":"e","MountList":[{"Type":"rclone","Param":"r1","DisplayName":"drive","Writable":true}]}
	]`)
	stores := map[string]StorageItem{}
	for _, it := range []struct{ key, typ, path string }{{"k1", StoreLocal, "/mnt/share"}, {"k2", StoreWebDAV, ""}} {
		var si StorageItem
		si.Key, si.Type, si.Remark, si.Params.LocalPath = it.key, it.typ, it.key, it.path
		stores[it.key] = si
	}
	users, warnings, err := ConvertFileUsers(raw, Options{ResolveStore: StoreResolver(stores, nil)})
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Fatalf("应合并为 2 个用户，实际 %+v", users)
	}
	v := users[0]
	if v.Name != "VanJay" || len(v.Mounts) != 3 {
		t.Fatalf("同名用户的挂载应合并: %+v", v)
	}
	if v.Mounts[1].Name != "nginx" || !v.Mounts[1].ReadOnly || v.Mounts[2].Name != "data-2" {
		t.Fatalf("挂载名称或只读映射错误: %+v", v.Mounts)
	}
	g := users[1]
	if len(g.Mounts) != 1 || g.Mounts[0].Path != "/mnt/share" || g.Mounts[0].Name != "share" {
		t.Fatalf("存储引用应解析为本地目录: %+v", g.Mounts)
	}
	joined := strings.Join(warnings, "\n")
	for _, want := range []string{"密码不同", "仅支持本地目录", "rclone", "empty 没有可迁移的目录"} {
		if !strings.Contains(joined, want) {
			t.Errorf("缺少提示 %q:\n%s", want, joined)
		}
	}
}

func TestListTypes(t *testing.T) {
	doc := Document{"List": json.RawMessage(`[{"Type":"tunnel"},{"Type":"access"}]`)}
	if !AllTypes(ListTypes(doc), "tunnel", "access") || AllTypes(ListTypes(doc), "client", "server") {
		t.Fatal("实例类型识别错误")
	}
	if AllTypes(ListTypes(Document{"List": json.RawMessage(`null`)}), "tunnel") {
		t.Fatal("空列表不应被任何模块认领")
	}
}
