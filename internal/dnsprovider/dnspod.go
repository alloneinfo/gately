package dnsprovider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// DNSPod 传统 Token 接口（dnsapi.cn / api.dnspod.com），ID 与 Token 拼接为 login_token。
// 文档：https://docs.dnspod.cn/api/
func init() {
	register(Meta{
		Name:         "dnspod",
		Title:        "DNSPod (国内站 Token)",
		IDLabel:      "ID",
		SecretLabel:  "Token",
		SupportsLine: true,
		DocURL:       "https://console.dnspod.cn/account/token/token",
	}, func(c Credentials, hc *http.Client) (Provider, error) {
		if err := requireCreds(c, true, true); err != nil {
			return nil, err
		}
		return &dnspod{token: c.ID + "," + c.Secret, hc: hc, endpoint: "https://dnsapi.cn", defaultLine: "默认"}, nil
	})
	register(Meta{
		Name:        "dnspodcom",
		Title:       "DNSPod (国际站 Token)",
		IDLabel:     "ID",
		SecretLabel: "Token",
		DocURL:      "https://www.dnspod.com/",
	}, func(c Credentials, hc *http.Client) (Provider, error) {
		if err := requireCreds(c, true, true); err != nil {
			return nil, err
		}
		return &dnspod{token: c.ID + "," + c.Secret, hc: hc, endpoint: "https://api.dnspod.com", defaultLine: "default"}, nil
	})
}

type dnspod struct {
	token       string
	hc          *http.Client
	endpoint    string
	defaultLine string
}

type dnspodStatus struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (d *dnspod) call(ctx context.Context, action string, form url.Values, out any) (dnspodStatus, error) {
	form.Set("login_token", d.token)
	form.Set("format", "json")
	form.Set("lang", "cn")
	req, err := ctxReq(ctx, http.MethodPost, d.endpoint+"/"+action, strings.NewReader(form.Encode()))
	if err != nil {
		return dnspodStatus{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// 官方要求设置可识别的 UA
	req.Header.Set("User-Agent", "Gately DDNS Client/1.0 (https://github.com/wangwanjie/gately)")
	var env struct {
		Status dnspodStatus `json:"status"`
	}
	if out == nil {
		out = &env
	}
	if err := doJSON(d.hc, req, out); err != nil {
		return dnspodStatus{}, err
	}
	if o, ok := out.(interface{ status() dnspodStatus }); ok {
		return o.status(), nil
	}
	return env.Status, nil
}

type dnspodList struct {
	Status  dnspodStatus `json:"status"`
	Records []struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Type  string `json:"type"`
		Value string `json:"value"`
		Line  string `json:"line"`
		TTL   string `json:"ttl"`
	} `json:"records"`
}

func (l *dnspodList) status() dnspodStatus { return l.Status }

func (d *dnspod) Upsert(ctx context.Context, r Record) (string, error) {
	var list dnspodList
	st, err := d.call(ctx, "Record.List", url.Values{
		"domain": {r.Zone}, "sub_domain": {r.RR()}, "record_type": {r.Type},
	}, &list)
	if err != nil {
		return "", err
	}
	// 10 表示没有记录
	if st.Code != "1" && st.Code != "10" {
		return "", fmt.Errorf("DNSPod Record.List: %s (%s)", st.Message, st.Code)
	}
	line := r.Line
	if line == "" {
		line = d.defaultLine
	}
	form := url.Values{
		"domain": {r.Zone}, "sub_domain": {r.RR()}, "record_type": {r.Type}, "record_line": {line}, "value": {r.Value},
	}
	if r.TTL > 0 {
		form.Set("ttl", strconv.Itoa(r.TTL))
	}
	for _, rec := range list.Records {
		if rec.Type != r.Type || rec.Line != line {
			continue
		}
		if rec.Value == r.Value && (r.TTL == 0 || rec.TTL == strconv.Itoa(r.TTL)) {
			return ActionUnchanged, nil
		}
		form.Set("record_id", rec.ID)
		st, err := d.call(ctx, "Record.Modify", form, nil)
		if err != nil {
			return "", err
		}
		if st.Code != "1" {
			return "", fmt.Errorf("DNSPod Record.Modify: %s (%s)", st.Message, st.Code)
		}
		return ActionUpdated, nil
	}
	st, err = d.call(ctx, "Record.Create", form, nil)
	if err != nil {
		return "", err
	}
	if st.Code != "1" {
		return "", fmt.Errorf("DNSPod Record.Create: %s (%s)", st.Message, st.Code)
	}
	return ActionCreated, nil
}
