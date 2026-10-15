package dnsprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// txtStore 是模拟服务商侧的 TXT 记录存储。
type txtStore struct {
	mu   sync.Mutex
	next int
	recs map[string]txtRec // id -> 记录
}

type txtRec struct{ name, value string }

func newTXTStore() *txtStore { return &txtStore{recs: map[string]txtRec{}} }

func (s *txtStore) add(name, value string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	id := fmt.Sprint(s.next)
	s.recs[id] = txtRec{name, value}
	return id
}

func (s *txtStore) del(id string) {
	s.mu.Lock()
	delete(s.recs, id)
	s.mu.Unlock()
}

func (s *txtStore) find(name string) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]string{}
	for id, r := range s.recs {
		if r.name == name {
			out[id] = r.value
		}
	}
	return out
}

func (s *txtStore) values(name string) []string {
	var out []string
	for _, v := range s.find(name) {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// runTXT 添加两个值、删除一个，验证剩余值正确。
func runTXT(t *testing.T, p Provider, store *txtStore, name string) {
	t.Helper()
	tp, err := AsTXT(p)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, v := range []string{"v1", "v2"} {
		if err := tp.AddTXT(ctx, "example.com", "_acme-challenge", v); err != nil {
			t.Fatalf("AddTXT %s: %v", v, err)
		}
	}
	if got := store.values(name); strings.Join(got, ",") != "v1,v2" {
		t.Fatalf("添加后应有两个值: %v", got)
	}
	if err := tp.DeleteTXT(ctx, "example.com", "_acme-challenge", "v1"); err != nil {
		t.Fatal(err)
	}
	if got := store.values(name); strings.Join(got, ",") != "v2" {
		t.Fatalf("删除后应只剩 v2: %v", got)
	}
	// 删除不存在的值不报错
	if err := tp.DeleteTXT(ctx, "example.com", "_acme-challenge", "nope"); err != nil {
		t.Fatal(err)
	}
}

func TestTXTAliDNS(t *testing.T) {
	st := newTXTStore()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch q.Get("Action") {
		case "AddDomainRecord":
			st.add(q.Get("RR"), q.Get("Value"))
			_, _ = w.Write([]byte(`{}`))
		case "DescribeSubDomainRecords":
			var recs []map[string]string
			for id, v := range st.find("_acme-challenge") {
				recs = append(recs, map[string]string{"RecordId": id, "Value": v, "Type": "TXT"})
			}
			b, _ := json.Marshal(map[string]any{"DomainRecords": map[string]any{"Record": recs}})
			_, _ = w.Write(b)
		case "DeleteDomainRecord":
			st.del(q.Get("RecordId"))
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()
	p, _ := New("alidns", Credentials{ID: "a", Secret: "b"}, srv.Client())
	p.(*aliDNS).endpoint = srv.URL + "/"
	runTXT(t, p, st, "_acme-challenge")
}

func TestTXTCloudflare(t *testing.T) {
	st := newTXTStore()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/zones":
			_, _ = w.Write([]byte(`{"success":true,"result":[{"id":"z","name":"example.com"}]}`))
		case r.Method == "POST":
			var rec cfRecord
			_ = json.NewDecoder(r.Body).Decode(&rec)
			st.add(rec.Name, rec.Content)
			_, _ = w.Write([]byte(`{"success":true,"result":{}}`))
		case r.Method == "GET":
			var recs []cfRecord
			// 模拟 Cloudflare 返回带引号的 TXT 内容
			for id, v := range st.find(r.URL.Query().Get("name")) {
				recs = append(recs, cfRecord{ID: id, Type: "TXT", Content: `"` + v + `"`})
			}
			b, _ := json.Marshal(map[string]any{"success": true, "result": recs})
			_, _ = w.Write(b)
		case r.Method == "DELETE":
			st.del(r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:])
			_, _ = w.Write([]byte(`{"success":true,"result":{}}`))
		}
	}))
	defer srv.Close()
	p, _ := New("cloudflare", Credentials{Secret: "t"}, srv.Client())
	p.(*cloudflare).endpoint = srv.URL
	runTXT(t, p, st, "_acme-challenge.example.com")
}

func TestTXTTencentCloud(t *testing.T) {
	st := newTXTStore()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p map[string]any
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &p)
		switch r.Header.Get("X-TC-Action") {
		case "CreateRecord":
			st.add(p["SubDomain"].(string), p["Value"].(string))
			_, _ = w.Write([]byte(`{"Response":{}}`))
		case "DescribeRecordList":
			var recs []map[string]any
			for id, v := range st.find(p["Subdomain"].(string)) {
				var n int
				fmt.Sscan(id, &n)
				recs = append(recs, map[string]any{"RecordId": n, "Value": v, "Type": "TXT"})
			}
			b, _ := json.Marshal(map[string]any{"Response": map[string]any{"RecordList": recs}})
			_, _ = w.Write(b)
		case "DeleteRecord":
			st.del(fmt.Sprint(p["RecordId"]))
			_, _ = w.Write([]byte(`{"Response":{}}`))
		}
	}))
	defer srv.Close()
	p, _ := New("tencentcloud", Credentials{ID: "a", Secret: "b"}, srv.Client())
	p.(*tencentCloud).endpoint = srv.URL
	p.(*tencentCloud).now = func() time.Time { return time.Unix(1700000000, 0) }
	runTXT(t, p, st, "_acme-challenge")
}

func TestTXTDNSPod(t *testing.T) {
	st := newTXTStore()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch r.URL.Path {
		case "/Record.Create":
			st.add(r.Form.Get("sub_domain"), r.Form.Get("value"))
			_, _ = w.Write([]byte(`{"status":{"code":"1"}}`))
		case "/Record.List":
			var recs []map[string]string
			for id, v := range st.find(r.Form.Get("sub_domain")) {
				recs = append(recs, map[string]string{"id": id, "value": v, "type": "TXT"})
			}
			b, _ := json.Marshal(map[string]any{"status": map[string]string{"code": "1"}, "records": recs})
			_, _ = w.Write(b)
		case "/Record.Remove":
			st.del(r.Form.Get("record_id"))
			_, _ = w.Write([]byte(`{"status":{"code":"1"}}`))
		}
	}))
	defer srv.Close()
	p, _ := New("dnspod", Credentials{ID: "1", Secret: "t"}, srv.Client())
	p.(*dnspod).endpoint = srv.URL
	runTXT(t, p, st, "_acme-challenge")
}

func TestTXTHuaweiCloud(t *testing.T) {
	// 华为云把同名 TXT 合并在一个记录集里，这里用 sets 模拟
	var mu sync.Mutex
	sets := map[string][]string{}
	st := newTXTStore()
	sync2store := func() {
		st.mu.Lock()
		st.recs = map[string]txtRec{}
		st.mu.Unlock()
		for _, v := range sets["rs1"] {
			st.add("_acme-challenge.example.com.", strings.Trim(v, `"`))
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.URL.Path == "/v2/zones":
			_, _ = w.Write([]byte(`{"zones":[{"id":"z","name":"example.com."}]}`))
		case r.Method == "GET":
			if recs, ok := sets["rs1"]; ok {
				b, _ := json.Marshal(map[string]any{"recordsets": []hwRecordset{{ID: "rs1", Name: "_acme-challenge.example.com.", Type: "TXT", Records: recs}}})
				_, _ = w.Write(b)
				return
			}
			_, _ = w.Write([]byte(`{"recordsets":[]}`))
		case r.Method == "POST" || r.Method == "PUT":
			var body hwRecordset
			_ = json.NewDecoder(r.Body).Decode(&body)
			for _, v := range body.Records {
				if !strings.HasPrefix(v, `"`) {
					t.Errorf("华为云 TXT 值应带引号: %s", v)
				}
			}
			sets["rs1"] = body.Records
			sync2store()
			_, _ = w.Write([]byte(`{}`))
		case r.Method == "DELETE":
			delete(sets, "rs1")
			sync2store()
		}
	}))
	defer srv.Close()
	p, _ := New("huaweicloud", Credentials{ID: "a", Secret: "b"}, srv.Client())
	p.(*huaweiCloud).endpoint = srv.URL
	runTXT(t, p, st, "_acme-challenge.example.com.")
}

func TestTXTGoDaddyPorkbunDynv6(t *testing.T) {
	// GoDaddy：按名称整体替换
	st := newTXTStore()
	gd := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			var recs []gdRecord
			for _, v := range st.values("_acme-challenge") {
				recs = append(recs, gdRecord{Data: v, TTL: 600})
			}
			b, _ := json.Marshal(recs)
			_, _ = w.Write(b)
		case "PUT":
			var recs []gdRecord
			_ = json.NewDecoder(r.Body).Decode(&recs)
			st.mu.Lock()
			st.recs = map[string]txtRec{}
			st.mu.Unlock()
			for _, rec := range recs {
				st.add("_acme-challenge", rec.Data)
			}
		case "DELETE":
			st.mu.Lock()
			st.recs = map[string]txtRec{}
			st.mu.Unlock()
		}
	}))
	defer gd.Close()
	p, _ := New("godaddy", Credentials{ID: "k", Secret: "s"}, gd.Client())
	p.(*godaddy).endpoint = gd.URL
	runTXT(t, p, st, "_acme-challenge")

	// Porkbun
	pst := newTXTStore()
	pb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case r.URL.Path == "/dns/create/example.com":
			pst.add(body["name"].(string), body["content"].(string))
			_, _ = w.Write([]byte(`{"status":"SUCCESS"}`))
		case strings.HasPrefix(r.URL.Path, "/dns/retrieveByNameType/"):
			var recs []map[string]string
			for id, v := range pst.find("_acme-challenge") {
				recs = append(recs, map[string]string{"id": id, "content": v})
			}
			b, _ := json.Marshal(map[string]any{"status": "SUCCESS", "records": recs})
			_, _ = w.Write(b)
		case strings.HasPrefix(r.URL.Path, "/dns/delete/example.com/"):
			pst.del(r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:])
			_, _ = w.Write([]byte(`{"status":"SUCCESS"}`))
		}
	}))
	defer pb.Close()
	pp, _ := New("porkbun", Credentials{ID: "k", Secret: "s"}, pb.Client())
	pp.(*porkbun).endpoint = pb.URL
	runTXT(t, pp, pst, "_acme-challenge")

	// dynv6
	dst := newTXTStore()
	dv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/zones/by-name/example.com":
			_, _ = w.Write([]byte(`{"id":1}`))
		case r.Method == "POST":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			dst.add(body["name"], body["data"])
		case r.Method == "GET":
			var recs []dynv6Record
			for id, v := range dst.find("_acme-challenge") {
				var n int64
				fmt.Sscan(id, &n)
				recs = append(recs, dynv6Record{ID: n, Name: "_acme-challenge", Type: "TXT", Data: v})
			}
			b, _ := json.Marshal(recs)
			_, _ = w.Write(b)
		case r.Method == "DELETE":
			dst.del(r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:])
		}
	}))
	defer dv.Close()
	dp, _ := New("dynv6", Credentials{Secret: "t"}, dv.Client())
	dp.(*dynv6).endpoint = dv.URL
	runTXT(t, dp, dst, "_acme-challenge")

	if _, err := AsTXT(&duckdns{}); err == nil {
		t.Fatal("DuckDNS 不支持 TXT，应报错")
	}
}
