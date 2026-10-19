package dnsprovider

import (
	"context"
	"crypto/sha1" // #nosec G505 -- OVH API 签名规定使用 SHA-1
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// OVH。ID 为 Application Key，Secret 为 Application Secret，Consumer Key 与接入点在额外参数中填写。
// 文档：https://api.ovh.com/createToken/
func init() {
	register(Meta{
		Name:        "ovh",
		Title:       "OVHcloud",
		IDLabel:     "Application Key",
		SecretLabel: "Application Secret",
		Extra: []Field{
			{Key: "consumerKey", Label: "Consumer Key", Required: true},
			{Key: "endpoint", Label: "接入点", Placeholder: "ovh-eu（默认）、ovh-ca、ovh-us 或完整 API 地址"},
		},
		DocURL:       "https://api.ovh.com/createToken/",
		SupportsACME: true,
	}, func(c Credentials, hc *http.Client) (Provider, error) {
		if err := requireCreds(c, true, true); err != nil {
			return nil, err
		}
		consumer := strings.TrimSpace(c.Extra["consumerKey"])
		if consumer == "" {
			return nil, errors.New("缺少 Consumer Key")
		}
		ep, err := ovhEndpoint(c.Extra["endpoint"])
		if err != nil {
			return nil, err
		}
		return &ovh{app: c.ID, secret: c.Secret, consumer: consumer, hc: hc, endpoint: ep}, nil
	})
}

func ovhEndpoint(name string) (string, error) {
	switch strings.TrimSpace(name) {
	case "", "ovh-eu":
		return "https://eu.api.ovh.com/1.0", nil
	case "ovh-ca":
		return "https://ca.api.ovh.com/1.0", nil
	case "ovh-us":
		return "https://api.us.ovhcloud.com/1.0", nil
	}
	u, err := url.Parse(name)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errors.New("OVH 接入点无效")
	}
	return strings.TrimRight(name, "/"), nil
}

type ovh struct {
	app, secret, consumer string
	hc                    *http.Client
	endpoint              string
	// delta 为本机与 OVH 服务器的时间差（秒），首次请求时校准
	delta     int64
	synced    bool
	nowForSig func() time.Time
}

type ovhRecord struct {
	ID        int64  `json:"id"`
	FieldType string `json:"fieldType"`
	SubDomain string `json:"subDomain"`
	Target    string `json:"target"`
	TTL       int    `json:"ttl"`
}

func (o *ovh) sync(ctx context.Context) error {
	if o.synced {
		return nil
	}
	req, err := ctxReq(ctx, http.MethodGet, o.endpoint+"/auth/time", nil)
	if err != nil {
		return err
	}
	var server int64
	if err := doJSON(o.hc, req, &server); err != nil {
		return fmt.Errorf("OVH: 获取服务器时间失败: %w", err)
	}
	now := time.Now()
	if o.nowForSig != nil {
		now = o.nowForSig()
	}
	o.delta = server - now.Unix()
	o.synced = true
	return nil
}

func (o *ovh) call(ctx context.Context, method, path string, q url.Values, body, out any) error {
	if err := o.sync(ctx); err != nil {
		return err
	}
	full := o.endpoint + path
	if len(q) > 0 {
		full += "?" + q.Encode()
	}
	var payload string
	var rd io.Reader
	if body != nil {
		b, _ := io.ReadAll(jsonBody(body))
		payload = string(b)
		rd = strings.NewReader(payload)
	}
	req, err := ctxReq(ctx, method, full, rd)
	if err != nil {
		return err
	}
	now := time.Now()
	if o.nowForSig != nil {
		now = o.nowForSig()
	}
	ts := strconv.FormatInt(now.Unix()+o.delta, 10)
	sum := sha1.Sum([]byte(strings.Join([]string{o.secret, o.consumer, method, full, payload, ts}, "+"))) // #nosec G401
	req.Header.Set("X-Ovh-Application", o.app)
	req.Header.Set("X-Ovh-Consumer", o.consumer)
	req.Header.Set("X-Ovh-Timestamp", ts)
	req.Header.Set("X-Ovh-Signature", "$1$"+hex.EncodeToString(sum[:]))
	req.Header.Set("Content-Type", "application/json")
	if err := doJSON(o.hc, req, out); err != nil {
		return fmt.Errorf("OVH: %w", err)
	}
	return nil
}

func ovhSub(sub string) string {
	if sub == "@" {
		return ""
	}
	return sub
}

// find 返回同名同类型的全部记录。
func (o *ovh) find(ctx context.Context, zone, sub, typ string) ([]ovhRecord, error) {
	var ids []int64
	q := url.Values{"fieldType": {typ}, "subDomain": {ovhSub(sub)}}
	if err := o.call(ctx, http.MethodGet, "/domain/zone/"+zone+"/record", q, nil, &ids); err != nil {
		return nil, err
	}
	recs := make([]ovhRecord, 0, len(ids))
	for _, id := range ids {
		var rec ovhRecord
		if err := o.call(ctx, http.MethodGet, "/domain/zone/"+zone+"/record/"+strconv.FormatInt(id, 10), nil, nil, &rec); err != nil {
			return nil, err
		}
		recs = append(recs, rec)
	}
	return recs, nil
}

// refresh 让修改在 DNS 区域中生效。
func (o *ovh) refresh(ctx context.Context, zone string) error {
	return o.call(ctx, http.MethodPost, "/domain/zone/"+zone+"/refresh", nil, nil, nil)
}

func (o *ovh) Upsert(ctx context.Context, r Record) (string, error) {
	recs, err := o.find(ctx, r.Zone, r.RR(), r.Type)
	if err != nil {
		return "", err
	}
	if len(recs) > 0 {
		cur := recs[0]
		if cur.Target == r.Value && (r.TTL <= 0 || cur.TTL == r.TTL) {
			return ActionUnchanged, nil
		}
		body := map[string]any{"target": r.Value}
		if r.TTL > 0 {
			body["ttl"] = r.TTL
		}
		if err := o.call(ctx, http.MethodPut, "/domain/zone/"+r.Zone+"/record/"+strconv.FormatInt(cur.ID, 10), nil, body, nil); err != nil {
			return "", err
		}
		return ActionUpdated, o.refresh(ctx, r.Zone)
	}
	body := map[string]any{"fieldType": r.Type, "subDomain": ovhSub(r.RR()), "target": r.Value}
	if r.TTL > 0 {
		body["ttl"] = r.TTL
	}
	if err := o.call(ctx, http.MethodPost, "/domain/zone/"+r.Zone+"/record", nil, body, nil); err != nil {
		return "", err
	}
	return ActionCreated, o.refresh(ctx, r.Zone)
}

func (o *ovh) AddTXT(ctx context.Context, zone, sub, value string) error {
	body := map[string]any{"fieldType": "TXT", "subDomain": ovhSub(Record{Zone: zone, Sub: sub}.RR()), "target": `"` + value + `"`, "ttl": 60}
	if err := o.call(ctx, http.MethodPost, "/domain/zone/"+zone+"/record", nil, body, nil); err != nil {
		return err
	}
	return o.refresh(ctx, zone)
}

func (o *ovh) DeleteTXT(ctx context.Context, zone, sub, value string) error {
	recs, err := o.find(ctx, zone, Record{Zone: zone, Sub: sub}.RR(), "TXT")
	if err != nil {
		return err
	}
	for _, rec := range recs {
		if unquote(rec.Target) == value {
			if err := o.call(ctx, http.MethodDelete, "/domain/zone/"+zone+"/record/"+strconv.FormatInt(rec.ID, 10), nil, nil, nil); err != nil {
				return err
			}
			return o.refresh(ctx, zone)
		}
	}
	return nil
}

var _ TXTProvider = (*ovh)(nil)
