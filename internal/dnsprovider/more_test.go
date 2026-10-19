package dnsprovider

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDynDNS2(t *testing.T) {
	cur := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "user" || p != "pw" {
			w.WriteHeader(401)
			return
		}
		if r.URL.Query().Get("hostname") != "home.example.com" {
			t.Errorf("hostname = %s", r.URL.Query().Get("hostname"))
		}
		ip := r.URL.Query().Get("myip")
		if ip == cur {
			_, _ = w.Write([]byte("nochg " + ip))
			return
		}
		cur = ip
		_, _ = w.Write([]byte("good " + ip))
	}))
	defer srv.Close()
	p, err := New("dyndns2", Credentials{ID: "user", Secret: "pw", Extra: map[string]string{"server": srv.URL}}, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	rec := Record{Zone: "example.com", Sub: "home", Type: "A", Value: "203.0.113.1"}
	if got, err := p.Upsert(context.Background(), rec); err != nil || got != ActionUpdated {
		t.Fatalf("got %s %v", got, err)
	}
	if got, _ := p.Upsert(context.Background(), rec); got != ActionUnchanged {
		t.Fatalf("got %s", got)
	}
	bad, _ := New("dyndns2", Credentials{ID: "user", Secret: "x", Extra: map[string]string{"server": srv.URL}}, srv.Client())
	if _, err := bad.Upsert(context.Background(), rec); err == nil {
		t.Fatal("密码错误应报错")
	}
	if _, err := New("dyndns2", Credentials{ID: "u", Secret: "p", Extra: map[string]string{"server": "ftp://x"}}, nil); err == nil {
		t.Fatal("非法更新地址应报错")
	}
}

func TestNamecheap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("host") != "home" || q.Get("domain") != "example.com" || q.Get("password") != "pw" || q.Get("ip") != "203.0.113.1" {
			t.Errorf("参数错误: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`<?xml version="1.0"?><interface-response><ErrCount>0</ErrCount></interface-response>`))
	}))
	defer srv.Close()
	p, _ := New("namecheap", Credentials{Secret: "pw"}, srv.Client())
	p.(*namecheap).endpoint = srv.URL
	if got, err := p.Upsert(context.Background(), Record{Zone: "example.com", Sub: "home", Type: "A", Value: "203.0.113.1"}); err != nil || got != ActionUpdated {
		t.Fatalf("got %s %v", got, err)
	}
	if _, err := p.Upsert(context.Background(), Record{Zone: "example.com", Sub: "home", Type: "AAAA", Value: "::1"}); err == nil {
		t.Fatal("不支持 AAAA")
	}
	errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<interface-response><ErrCount>1</ErrCount><errors><Err1>Passwords do not match</Err1></errors></interface-response>`))
	}))
	defer errSrv.Close()
	p.(*namecheap).endpoint = errSrv.URL
	if _, err := p.Upsert(context.Background(), Record{Zone: "example.com", Sub: "home", Type: "A", Value: "1.1.1.1"}); err == nil || !strings.Contains(err.Error(), "Passwords") {
		t.Fatalf("应返回服务商错误: %v", err)
	}
}

func TestGandi(t *testing.T) {
	var values []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer pat" {
			w.WriteHeader(401)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/domains/example.com/records/") {
			t.Errorf("路径错误 %s", r.URL.Path)
		}
		switch r.Method {
		case "GET":
			if len(values) == 0 {
				w.WriteHeader(404)
				_, _ = w.Write([]byte(`{"message":"not found"}`))
				return
			}
			b, _ := json.Marshal(map[string]any{"rrset_values": values, "rrset_ttl": 300})
			_, _ = w.Write(b)
		case "PUT":
			var body struct {
				Values []string `json:"rrset_values"`
			}
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &body)
			values = body.Values
			_, _ = w.Write([]byte(`{"message":"ok"}`))
		case "DELETE":
			values = nil
			_, _ = w.Write([]byte(`{"message":"ok"}`))
		}
	}))
	defer srv.Close()
	p, _ := New("gandi", Credentials{Secret: "pat"}, srv.Client())
	p.(*gandi).endpoint = srv.URL
	runUpsertSequence(t, p, Record{Zone: "example.com", Sub: "home", Type: "A"})

	// TXT：新增两个值，删除其一后仍保留另一个
	values = nil
	txt := p.(TXTProvider)
	_ = txt.AddTXT(context.Background(), "example.com", "_acme-challenge", "v1")
	_ = txt.AddTXT(context.Background(), "example.com", "_acme-challenge", "v2")
	if len(values) != 2 || values[0] != `"v1"` {
		t.Fatalf("TXT 值 = %v", values)
	}
	_ = txt.DeleteTXT(context.Background(), "example.com", "_acme-challenge", "v1")
	if len(values) != 1 || values[0] != `"v2"` {
		t.Fatalf("删除后 = %v", values)
	}
	_ = txt.DeleteTXT(context.Background(), "example.com", "_acme-challenge", "v2")
	if len(values) != 0 {
		t.Fatalf("应整体删除: %v", values)
	}
}

func TestNS1(t *testing.T) {
	var rec *ns1Record
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-NSONE-Key") != "k" {
			w.WriteHeader(401)
			return
		}
		switch r.Method {
		case "GET":
			if rec == nil {
				w.WriteHeader(404)
				_, _ = w.Write([]byte(`{"message":"record not found"}`))
				return
			}
			_ = json.NewEncoder(w).Encode(rec)
		case "PUT", "POST":
			var body ns1Record
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &body)
			rec = &body
			_, _ = w.Write([]byte(`{}`))
		case "DELETE":
			rec = nil
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()
	p, _ := New("ns1", Credentials{Secret: "k"}, srv.Client())
	p.(*ns1).endpoint = srv.URL
	runUpsertSequence(t, p, Record{Zone: "example.com", Sub: "home", Type: "A"})
	rec = nil
	txt := p.(TXTProvider)
	_ = txt.AddTXT(context.Background(), "example.com", "_acme-challenge", "v1")
	_ = txt.AddTXT(context.Background(), "example.com", "_acme-challenge", "v2")
	if rec == nil || len(rec.Answers) != 2 {
		t.Fatalf("answers = %+v", rec)
	}
	_ = txt.DeleteTXT(context.Background(), "example.com", "_acme-challenge", "v1")
	_ = txt.DeleteTXT(context.Background(), "example.com", "_acme-challenge", "v2")
	if rec != nil {
		t.Fatal("应整体删除")
	}
}

func TestBunny(t *testing.T) {
	var records []bunnyRecord
	next := int64(100)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("AccessKey") != "k" {
			w.WriteHeader(401)
			return
		}
		switch {
		case r.Method == "GET" && r.URL.Path == "/dnszone":
			_, _ = w.Write([]byte(`{"Items":[{"Id":7,"Domain":"example.com"}]}`))
		case r.Method == "GET" && r.URL.Path == "/dnszone/7":
			_ = json.NewEncoder(w).Encode(map[string]any{"Records": records})
		case r.Method == "PUT" && r.URL.Path == "/dnszone/7/records":
			var rec bunnyRecord
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &rec)
			next++
			rec.ID = next
			records = append(records, rec)
			w.WriteHeader(201)
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/dnszone/7/records/"):
			var rec bunnyRecord
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &rec)
			for i := range records {
				if records[i].ID == rec.ID {
					records[i] = rec
				}
			}
			w.WriteHeader(204)
		case r.Method == "DELETE":
			records = nil
			w.WriteHeader(204)
		default:
			t.Errorf("意外请求 %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()
	p, _ := New("bunny", Credentials{Secret: "k"}, srv.Client())
	p.(*bunny).endpoint = srv.URL
	runUpsertSequence(t, p, Record{Zone: "example.com", Sub: "home", Type: "A"})
	records = nil
	txt := p.(TXTProvider)
	_ = txt.AddTXT(context.Background(), "example.com", "_acme-challenge", "v1")
	if len(records) != 1 || records[0].Type != bunnyTXT {
		t.Fatalf("records = %+v", records)
	}
	_ = txt.DeleteTXT(context.Background(), "example.com", "_acme-challenge", "v1")
	if len(records) != 0 {
		t.Fatal("应已删除")
	}
}

// AWS 官方 SigV4 测试向量（get-vanilla）
func TestSignV4Vector(t *testing.T) {
	ts := time.Date(2015, 8, 30, 12, 36, 0, 0, time.UTC)
	auth := signV4("GET", "/", "", map[string]string{"host": "example.amazonaws.com", "x-amz-date": "20150830T123600Z"}, nil,
		"AKIDEXAMPLE", "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY", "us-east-1", "service", ts)
	want := "AWS4-HMAC-SHA256 Credential=AKIDEXAMPLE/20150830/us-east-1/service/aws4_request, SignedHeaders=host;x-amz-date, Signature=5fa00fa31553b73ebf1942676e86291e8372ff2a2260956d9b8aae1d763fbf31"
	if auth != want {
		t.Fatalf("签名不符:\n got %s\nwant %s", auth, want)
	}
}

func TestRoute53(t *testing.T) {
	var value string
	var txtVals []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 Credential=AKID/") {
			w.WriteHeader(403)
			return
		}
		switch {
		case r.Method == "GET" && r.URL.Path == "/2013-04-01/hostedzonesbyname":
			_, _ = w.Write([]byte(`<ListHostedZonesByNameResponse><HostedZones><HostedZone><Id>/hostedzone/ZZ1</Id><Name>example.com.</Name></HostedZone></HostedZones></ListHostedZonesByNameResponse>`))
		case r.Method == "GET" && r.URL.Path == "/2013-04-01/hostedzone/ZZ1/rrset":
			name, typ := r.URL.Query().Get("name"), r.URL.Query().Get("type")
			var vals []string
			if typ == "A" && value != "" {
				vals = []string{value}
			} else if typ == "TXT" {
				vals = txtVals
			}
			if len(vals) == 0 {
				_, _ = w.Write([]byte(`<ListResourceRecordSetsResponse><ResourceRecordSets></ResourceRecordSets></ListResourceRecordSetsResponse>`))
				return
			}
			var sb strings.Builder
			for _, v := range vals {
				sb.WriteString("<ResourceRecord><Value>" + strings.ReplaceAll(v, `"`, "&quot;") + "</Value></ResourceRecord>")
			}
			_, _ = w.Write([]byte(`<ListResourceRecordSetsResponse><ResourceRecordSets><ResourceRecordSet><Name>` + name + `</Name><Type>` + typ + `</Type><TTL>300</TTL><ResourceRecords>` + sb.String() + `</ResourceRecords></ResourceRecordSet></ResourceRecordSets></ListResourceRecordSetsResponse>`))
		case r.Method == "POST" && r.URL.Path == "/2013-04-01/hostedzone/ZZ1/rrset":
			b, _ := io.ReadAll(r.Body)
			var req struct {
				Action string   `xml:"ChangeBatch>Changes>Change>Action"`
				Type   string   `xml:"ChangeBatch>Changes>Change>ResourceRecordSet>Type"`
				Values []string `xml:"ChangeBatch>Changes>Change>ResourceRecordSet>ResourceRecords>ResourceRecord>Value"`
			}
			if err := xmlUnmarshal(b, &req); err != nil {
				t.Errorf("请求体解析失败: %v %s", err, b)
			}
			switch {
			case req.Type == "A":
				value = req.Values[0]
			case req.Action == "DELETE":
				txtVals = nil
			default:
				txtVals = req.Values
			}
			_, _ = w.Write([]byte(`<ChangeResourceRecordSetsResponse/>`))
		default:
			t.Errorf("意外请求 %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()
	p, err := New("route53", Credentials{ID: "AKID", Secret: "SK"}, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	p.(*route53).endpoint = srv.URL
	runUpsertSequence(t, p, Record{Zone: "example.com", Sub: "home", Type: "A"})
	txt := p.(TXTProvider)
	_ = txt.AddTXT(context.Background(), "example.com", "_acme-challenge", "v1")
	_ = txt.AddTXT(context.Background(), "example.com", "_acme-challenge", "v2")
	if len(txtVals) != 2 {
		t.Fatalf("txt = %v", txtVals)
	}
	_ = txt.DeleteTXT(context.Background(), "example.com", "_acme-challenge", "v1")
	if len(txtVals) != 1 || txtVals[0] != `"v2"` {
		t.Fatalf("删除后 = %v", txtVals)
	}
	_ = txt.DeleteTXT(context.Background(), "example.com", "_acme-challenge", "v2")
	if txtVals != nil {
		t.Fatalf("应整体删除: %v", txtVals)
	}
}

func TestOVH(t *testing.T) {
	var rec *ovhRecord
	refreshed := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/time" {
			_, _ = w.Write([]byte("1700000000"))
			return
		}
		if r.Header.Get("X-Ovh-Application") != "app" || !strings.HasPrefix(r.Header.Get("X-Ovh-Signature"), "$1$") || r.Header.Get("X-Ovh-Timestamp") != "1700000000" {
			t.Errorf("签名头不符: %v", r.Header)
		}
		switch {
		case r.Method == "GET" && r.URL.Path == "/domain/zone/example.com/record":
			if rec == nil {
				_, _ = w.Write([]byte(`[]`))
				return
			}
			_, _ = w.Write([]byte(`[1]`))
		case r.Method == "GET" && r.URL.Path == "/domain/zone/example.com/record/1":
			_ = json.NewEncoder(w).Encode(rec)
		case r.Method == "POST" && r.URL.Path == "/domain/zone/example.com/record":
			var body map[string]any
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &body)
			rec = &ovhRecord{ID: 1, FieldType: "A", SubDomain: body["subDomain"].(string), Target: body["target"].(string)}
			_, _ = w.Write([]byte(`{}`))
		case r.Method == "PUT" && r.URL.Path == "/domain/zone/example.com/record/1":
			var body map[string]any
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &body)
			rec.Target = body["target"].(string)
		case r.Method == "POST" && r.URL.Path == "/domain/zone/example.com/refresh":
			refreshed++
		default:
			t.Errorf("意外请求 %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()
	p, err := New("ovh", Credentials{ID: "app", Secret: "sec", Extra: map[string]string{"consumerKey": "ck", "endpoint": srv.URL}}, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	p.(*ovh).nowForSig = func() time.Time { return time.Unix(1700000000, 0) }
	runUpsertSequence(t, p, Record{Zone: "example.com", Sub: "home", Type: "A"})
	if refreshed != 2 {
		t.Fatalf("创建和更新后各应刷新一次，实际 %d", refreshed)
	}
	if _, err := New("ovh", Credentials{ID: "a", Secret: "s"}, nil); err == nil {
		t.Fatal("缺少 Consumer Key 应报错")
	}
}

func TestNewProvidersRegistered(t *testing.T) {
	for _, n := range []string{"dyndns2", "namecheap", "gandi", "ns1", "bunny", "route53", "ovh"} {
		if _, ok := Lookup(n); !ok {
			t.Errorf("缺少服务商 %s", n)
		}
	}
	for _, n := range []string{"gandi", "ns1", "bunny", "route53", "ovh"} {
		if !SupportsTXT(n) {
			t.Errorf("%s 应支持 ACME", n)
		}
	}
	if SupportsTXT("dyndns2") || SupportsTXT("namecheap") {
		t.Error("dyndns2 与 namecheap 不应支持 ACME")
	}
}

func xmlUnmarshal(b []byte, v any) error {
	return xml.Unmarshal(b, v)
}
