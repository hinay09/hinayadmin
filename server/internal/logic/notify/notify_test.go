// Package notify 纯函数单测 (不依赖 Redis/DB)。
package notify

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestCfgBool(t *testing.T) {
	for in, want := range map[string]bool{
		"true": true, "1": true, " True ": true,
		"false": false, "0": false, "": false, "yes": false,
	} {
		if got := cfgBool(in); got != want {
			t.Fatalf("cfgBool(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestDedupeIds(t *testing.T) {
	got := dedupeIds([]uint64{3, 0, 3, 7, 7, 9})
	if len(got) != 3 || got[0] != 3 || got[1] != 7 || got[2] != 9 {
		t.Fatalf("dedupeIds = %v", got)
	}
}

func TestBusEventJSON(t *testing.T) {
	// 定向
	ev := BusEvent{TenantId: 2, Origin: "abc", UserIds: []uint64{1, 2}, Title: "标题", Level: 2}
	data, err := json.Marshal(&ev)
	if err != nil {
		t.Fatal(err)
	}
	var back BusEvent
	if err = json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if back.TenantId != 2 || back.Origin != "abc" || back.All || len(back.UserIds) != 2 || back.Title != "标题" || back.Level != 2 {
		t.Fatalf("roundtrip mismatch: %+v", back)
	}
	// 全员
	ev2 := BusEvent{All: true, Title: "t"}
	data2, _ := json.Marshal(&ev2)
	if strings.Contains(string(data2), `"userIds"`) {
		t.Fatalf("全员消息不应携带 userIds: %s", data2)
	}
}

func TestBuildMail(t *testing.T) {
	raw := buildMail("from@a.b", "to@c.d", "服务器巡检告警", "CPU 91%\n内存 88%")
	s := string(raw)
	for _, want := range []string{
		"From: from@a.b\r\n",
		"To: to@c.d\r\n",
		"MIME-Version: 1.0\r\n",
		"Content-Type: text/plain; charset=UTF-8\r\n",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("邮件头缺失 %q:\n%s", want, s)
		}
	}
	// Subject 为 RFC 2047 编码 (含 =?UTF-8? 前缀, 原文不出现)
	if !strings.Contains(s, "Subject: =?UTF-8?") || strings.Contains(s, "Subject: 服务器巡检告警") {
		t.Fatalf("Subject 未按 RFC2047 编码:\n%s", s)
	}
	// 正文 base64 解码回原文
	idx := strings.Index(s, "base64\r\n\r\n")
	if idx < 0 {
		t.Fatalf("缺少 base64 正文分隔:\n%s", s)
	}
	b64 := strings.ReplaceAll(strings.TrimSpace(s[idx+len("base64\r\n\r\n"):]), "\r\n", "")
	dec, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("正文不是合法 base64: %v", err)
	}
	if want := "CPU 91%\n内存 88%"; string(dec) != want {
		t.Fatalf("正文解码不符: %q, want %q", dec, want)
	}
}

func TestTruncateBytes(t *testing.T) {
	// ASCII 原样
	if got := truncateBytes("abcdef", 10); got != "abcdef" {
		t.Fatalf("短串应原样: %q", got)
	}
	if got := truncateBytes("abcdef", 3); got != "abc…" {
		t.Fatalf("ASCII 截断: %q", got)
	}
	// 中文 3 字节/字: 限 7 字节应完整保留 2 个字 (不能切出残字节)
	got := truncateBytes("一二三四五", 7)
	if !strings.HasPrefix(got, "一二") || !strings.HasSuffix(got, "…") {
		t.Fatalf("多字节截断应回退到完整字符: %q", got)
	}
}

func TestSignDingTalkAt(t *testing.T) {
	endpoint := signDingTalkAt("https://oapi.dingtalk.com/robot/send?access_token=x", "SEC123", 1700000000000)
	if !strings.HasPrefix(endpoint, "https://oapi.dingtalk.com/robot/send?access_token=x&") {
		t.Fatalf("应原样保留原查询串: %q", endpoint)
	}
	if !strings.Contains(endpoint, "timestamp=1700000000000") || !strings.Contains(endpoint, "&sign=") {
		t.Fatalf("缺少 timestamp/sign: %q", endpoint)
	}
}

func TestEmailEnabled(t *testing.T) {
	e := &emailChannel{}
	if e.Enabled(map[string]string{"notify.email.enable": "true"}) {
		t.Fatal("缺 host 不应启用")
	}
	if !e.Enabled(map[string]string{"notify.email.enable": "true", "notify.email.host": "smtp.x.com"}) {
		t.Fatal("enable+host 应启用")
	}
	if e.Enabled(map[string]string{"notify.email.host": "smtp.x.com"}) {
		t.Fatal("未开启 enable 不应启用")
	}
}

func TestWebhookEnabled(t *testing.T) {
	w := &webhookChannel{kind: "dingtalk"}
	if w.Enabled(map[string]string{"notify.webhook.dingtalk.enable": "true"}) {
		t.Fatal("缺 url 不应启用")
	}
	if !w.Enabled(map[string]string{
		"notify.webhook.dingtalk.enable": "true",
		"notify.webhook.dingtalk.url":    "https://oapi.dingtalk.com/robot/send?access_token=x",
	}) {
		t.Fatal("enable+url 应启用")
	}
	if (&webhookChannel{kind: "wecom"}).Enabled(map[string]string{
		"notify.webhook.dingtalk.enable": "true",
		"notify.webhook.dingtalk.url":    "https://x",
	}) {
		t.Fatal("企微渠道不应读钉钉配置")
	}
}
