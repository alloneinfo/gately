package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wangwanjie/gately/internal/i18n"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/wangwanjie/gately/internal/events"
	"github.com/wangwanjie/gately/internal/webhook"
)

// 渠道类型
const (
	KindWebhook  = "webhook"
	KindTelegram = "telegram"
	KindBark     = "bark"
	KindWeCom    = "wecom"
	KindDingTalk = "dingtalk"
	KindFeishu   = "feishu"
	KindSMTP     = "smtp"
)

// Message 是投递给渠道的内容。
type Message struct {
	Event events.Event
	// Title 与 Text 已经过格式化，可直接展示
	Title string
	Text  string
}

var levelLabel = map[string]string{
	events.LevelInfo:  "信息",
	events.LevelWarn:  "警告",
	events.LevelError: "错误",
}

// NewMessage 把事件格式化为人可读的消息。
func NewMessage(e events.Event) Message {
	// 通知内容按服务端语言翻译（事件文案由程序生成，用户自定义的内容不在翻译表中，会原样保留）
	lang := i18n.ServerLang()
	tr := func(s string) string { return i18n.Tr(lang, s) }
	e.Title = tr(e.Title)
	e.Message = tr(e.Message)
	title := e.Title
	if title == "" {
		title = e.Type
	}
	label := tr(levelLabel[e.Level])
	if label == "" {
		label = e.Level
	}
	var b strings.Builder
	if e.Message != "" {
		b.WriteString(e.Message)
		b.WriteString("\n")
	}
	if e.Source != "" {
		b.WriteString(tr("来源：") + e.Source + "\n")
	}
	b.WriteString(tr("级别：") + label + "\n")
	b.WriteString(tr("时间：") + e.Time.Format("2006-01-02 15:04:05"))
	return Message{Event: e, Title: "[Gately] " + title, Text: b.String()}
}

// vars 返回 Webhook 模板变量。
func (m Message) vars() map[string]string {
	v := map[string]string{
		"title":   m.Title,
		"message": m.Event.Message,
		"text":    m.Text,
		"type":    m.Event.Type,
		"source":  m.Event.Source,
		"level":   m.Event.Level,
		"time":    m.Event.Time.Format(time.RFC3339),
	}
	for k, val := range m.Event.Fields {
		if _, exists := v[k]; !exists {
			v[k] = val
		}
	}
	return v
}

// httpClient 供各渠道共用；测试可替换。
var httpClient = &http.Client{Timeout: 20 * time.Second}

// send 按渠道类型投递消息。
func send(ctx context.Context, ch Channel, msg Message) error {
	switch ch.Kind {
	case KindWebhook:
		cfg := ch.Webhook
		cfg.Enable = true
		_, err := webhook.Send(ctx, cfg, msg.vars())
		return err
	case KindTelegram:
		return sendTelegram(ctx, ch, msg)
	case KindBark:
		return sendBark(ctx, ch, msg)
	case KindWeCom:
		return postJSON(ctx, ch.URL, map[string]any{
			"msgtype": "text",
			"text":    map[string]string{"content": msg.Title + "\n" + msg.Text},
		}, "errcode", 0)
	case KindDingTalk:
		return sendDingTalk(ctx, ch, msg)
	case KindFeishu:
		return sendFeishu(ctx, ch, msg)
	case KindSMTP:
		return sendSMTP(ctx, ch, msg)
	}
	return fmt.Errorf("不支持的渠道类型 %q", ch.Kind)
}

// postJSON 发送 JSON；okField 非空时要求响应 JSON 中该字段等于 okValue（数字），兼容企业微信、钉钉等“HTTP 200 但业务失败”的返回。
func postJSON(ctx context.Context, target string, body any, okField string, okValue float64) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, shorten(string(raw)))
	}
	if okField == "" {
		return nil
	}
	var parsed map[string]any
	if json.Unmarshal(raw, &parsed) != nil {
		return nil
	}
	v, ok := parsed[okField]
	if !ok {
		return nil
	}
	if n, isNum := v.(float64); isNum && n != okValue {
		return fmt.Errorf("渠道返回错误: %s", shorten(string(raw)))
	}
	return nil
}

func shorten(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) > 200 {
		return string(r[:200]) + "..."
	}
	return s
}

func sendTelegram(ctx context.Context, ch Channel, msg Message) error {
	base := strings.TrimRight(ch.URL, "/")
	if base == "" {
		base = "https://api.telegram.org"
	}
	target := base + "/bot" + ch.Secret + "/sendMessage"
	return postJSON(ctx, target, map[string]any{
		"chat_id": ch.Target,
		"text":    msg.Title + "\n" + msg.Text,
	}, "", 0)
}

func sendBark(ctx context.Context, ch Channel, msg Message) error {
	base := strings.TrimRight(ch.URL, "/")
	if base == "" {
		base = "https://api.day.app"
	}
	target := base + "/" + url.PathEscape(ch.Secret)
	return postJSON(ctx, target, map[string]any{
		"title": msg.Title,
		"body":  msg.Text,
		"group": "Gately",
	}, "code", 200)
}

func sendDingTalk(ctx context.Context, ch Channel, msg Message) error {
	target := ch.URL
	if ch.Secret != "" {
		ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
		mac := hmac.New(sha256.New, []byte(ch.Secret))
		mac.Write([]byte(ts + "\n" + ch.Secret))
		sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		sep := "?"
		if strings.Contains(target, "?") {
			sep = "&"
		}
		target += sep + "timestamp=" + ts + "&sign=" + url.QueryEscape(sign)
	}
	return postJSON(ctx, target, map[string]any{
		"msgtype": "text",
		"text":    map[string]string{"content": msg.Title + "\n" + msg.Text},
	}, "errcode", 0)
}

func sendFeishu(ctx context.Context, ch Channel, msg Message) error {
	body := map[string]any{
		"msg_type": "text",
		"content":  map[string]string{"text": msg.Title + "\n" + msg.Text},
	}
	if ch.Secret != "" {
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		mac := hmac.New(sha256.New, []byte(ts+"\n"+ch.Secret))
		body["timestamp"] = ts
		body["sign"] = base64.StdEncoding.EncodeToString(mac.Sum(nil))
	}
	return postJSON(ctx, ch.URL, body, "code", 0)
}

// SMTP 加密方式
const (
	SecurityNone     = "none"
	SecurityStartTLS = "starttls"
	SecurityTLS      = "tls"
)

func sendSMTP(ctx context.Context, ch Channel, msg Message) error {
	addr := net.JoinHostPort(ch.Host, strconv.Itoa(ch.Port))
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	tlsCfg := &tls.Config{ServerName: ch.Host}
	var conn net.Conn
	var err error
	if ch.Security == SecurityTLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsCfg)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return err
	}
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	c, err := smtp.NewClient(conn, ch.Host)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()
	if ch.Security == SecurityStartTLS {
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return errors.New("服务器不支持 STARTTLS")
		}
		if err := c.StartTLS(tlsCfg); err != nil {
			return err
		}
	}
	if ch.User != "" {
		if ok, _ := c.Extension("AUTH"); ok {
			if err := c.Auth(smtp.PlainAuth("", ch.User, ch.Secret, ch.Host)); err != nil {
				return err
			}
		}
	}
	from := ch.From
	if from == "" {
		from = ch.User
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	var rcpts []string
	for _, t := range strings.Split(ch.Target, ",") {
		if t = strings.TrimSpace(t); t != "" {
			rcpts = append(rcpts, t)
			if err := c.Rcpt(t); err != nil {
				return err
			}
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteString("From: " + from + "\r\n")
	buf.WriteString("To: " + strings.Join(rcpts, ", ") + "\r\n")
	buf.WriteString("Subject: =?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(msg.Title)) + "?=\r\n")
	buf.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n")
	enc := base64.StdEncoding.EncodeToString([]byte(msg.Text))
	for len(enc) > 76 {
		buf.WriteString(enc[:76] + "\r\n")
		enc = enc[76:]
	}
	buf.WriteString(enc + "\r\n")
	if _, err := w.Write(buf.Bytes()); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
