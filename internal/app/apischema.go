package app

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// schemaSet 把 Go 类型反射为 OpenAPI 的 JSON Schema，具名结构体放入 components 并以 $ref 引用。
type schemaSet struct {
	defs map[string]map[string]any
	// inProgress 用于递归类型，先占位再填充
	inProgress map[reflect.Type]string
}

func newSchemaSet() *schemaSet {
	return &schemaSet{defs: map[string]map[string]any{}, inProgress: map[reflect.Type]string{}}
}

var (
	timeType = reflect.TypeOf(time.Time{})
	rawType  = reflect.TypeOf(json.RawMessage{})
	nameSan  = regexp.MustCompile(`[^A-Za-z0-9_.]+`)
)

// refName 生成结构体的组件名：包名.类型名。
func refName(t reflect.Type) string {
	pkg := t.PkgPath()
	if i := strings.LastIndex(pkg, "/"); i >= 0 {
		pkg = pkg[i+1:]
	}
	return nameSan.ReplaceAllString(pkg+"."+t.Name(), "_")
}

// of 返回类型 t 的 schema（可能是 $ref）。
func (s *schemaSet) of(t reflect.Type) map[string]any {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch {
	case t == timeType:
		return map[string]any{"type": "string", "format": "date-time"}
	case t == rawType:
		return map[string]any{}
	}
	switch t.Kind() {
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if t == reflect.TypeOf(time.Duration(0)) {
			return map[string]any{"type": "integer", "description": "时长，单位纳秒"}
		}
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return map[string]any{"type": "string", "format": "byte"}
		}
		return map[string]any{"type": "array", "items": s.of(t.Elem())}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": s.of(t.Elem())}
	case reflect.Struct:
		return s.structRef(t)
	}
	// interface{} 等无法确定类型的值
	return map[string]any{}
}

func (s *schemaSet) structRef(t reflect.Type) map[string]any {
	name := refName(t)
	if t.Name() == "" {
		// 匿名结构体直接内联
		return s.structBody(t)
	}
	ref := map[string]any{"$ref": "#/components/schemas/" + name}
	if _, ok := s.defs[name]; ok {
		return ref
	}
	if _, ok := s.inProgress[t]; ok {
		return ref
	}
	s.inProgress[t] = name
	s.defs[name] = s.structBody(t)
	delete(s.inProgress, t)
	return ref
}

func (s *schemaSet) structBody(t reflect.Type) map[string]any {
	props := map[string]any{}
	s.collect(t, props)
	return map[string]any{"type": "object", "properties": props}
}

// collect 收集结构体字段，匿名嵌入且无 json 标签的结构体展开到当前层。
func (s *schemaSet) collect(t reflect.Type, props map[string]any) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if f.Anonymous && name == "" {
			ft := f.Type
			for ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				s.collect(ft, props)
				continue
			}
		}
		if !f.IsExported() {
			continue
		}
		if name == "" {
			name = f.Name
		}
		props[name] = s.of(f.Type)
	}
}
