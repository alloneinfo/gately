package webhook

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRender(t *testing.T) {
	vars := map[string]string{"ip": "1.2.3.4", "domain": "a b"}
	cases := map[string]string{
		"{ip}":                 "1.2.3.4",
		"x={ip}&d={domain}":    "x=1.2.3.4&d=a b",
		"{unknown}{ip}":        "{unknown}1.2.3.4",
		"no vars":              "no vars",
		"{unclosed":            "{unclosed",
		`{"ip":"{ip}"}`:        `{"ip":"1.2.3.4"}`,
		"{ip}{ip}":             "1.2.3.41.2.3.4",
		"prefix-{domain}-tail": "prefix-a b-tail",
		"#{ip} #{nope}":        "1.2.3.4 #{nope}",
		"a#b{ip}":              "a#b1.2.3.4",
	}
	for in, want := range cases {
		if got := Render(in, vars, nil); got != want {
			t.Errorf("Render(%q)=%q want %q", in, got, want)
		}
	}
}

func TestSend(t *testing.T) {
	var gotQuery, gotBody, gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("d")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotHeader = r.Header.Get("X-Token")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	cfg := Config{
		Enable:         true,
		URL:            srv.URL + "/hook?d={domain}",
		Method:         "post",
		Headers:        []string{"X-Token: t-{ip}"},
		Body:           `{"ip":"{ip}"}`,
		SuccessContent: []string{`"ok":true`},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := Send(context.Background(), cfg, map[string]string{"ip": "1.2.3.4", "domain": "a&b"}); err != nil {
		t.Fatal(err)
	}
	if gotQuery != "a&b" || gotBody != `{"ip":"1.2.3.4"}` || gotHeader != "t-1.2.3.4" {
		t.Fatalf("请求内容不符: q=%q body=%q header=%q", gotQuery, gotBody, gotHeader)
	}

	cfg.SuccessContent = []string{"never"}
	if _, err := Send(context.Background(), cfg, nil); err == nil {
		t.Fatal("未包含成功标识应报错")
	}
}

func TestValidate(t *testing.T) {
	bad := []Config{
		{Enable: true, URL: "ftp://x"},
		{Enable: true, URL: "http://x", Method: "TRACE"},
		{Enable: true, URL: "http://x", Headers: []string{"novalue"}},
	}
	for _, c := range bad {
		if err := c.Validate(); err == nil {
			t.Errorf("应校验失败: %+v", c)
		}
	}
	if err := (&Config{Enable: false, URL: "garbage"}).Validate(); err != nil {
		t.Fatal("未启用时不应校验")
	}
}
