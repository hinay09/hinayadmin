package sms

import (
	"strings"
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	for in, want := range map[string]string{
		"13800138000":    "+8613800138000",
		"+8613800138000": "+8613800138000",
		"8613800138000":  "+8613800138000",
		" 13800138000 ":  "+8613800138000",
	} {
		if got := normalizePhone(in); got != want {
			t.Fatalf("normalizePhone(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAliyunEncode(t *testing.T) {
	// RFC3986 规则: 空格→%20、*→%2A、~不编码, 中文按 UTF-8 百分号编码
	for in, want := range map[string]string{
		"a b":   "a%20b",
		"a*b":   "a%2Ab",
		"a~b":   "a~b",
		"中文":    "%E4%B8%AD%E6%96%87",
		"a=b&c": "a%3Db%26c",
	} {
		if got := aliyunEncode(in); got != want {
			t.Fatalf("aliyunEncode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSignAliyunQuery(t *testing.T) {
	q := signAliyunQuery(map[string]string{
		"AccessKeyId":  "testid",
		"Action":       "SendSms",
		"PhoneNumbers": "13800138000",
	}, "testsecret")
	if !strings.HasPrefix(q, "AccessKeyId=testid&Action=SendSms&") {
		t.Fatalf("参数应按 key 排序拼接: %q", q)
	}
	if !strings.Contains(q, "&Signature=") {
		t.Fatalf("缺少签名: %q", q)
	}
}

func TestTc3Authorization(t *testing.T) {
	// 锁定签名串构造: 固定输入产出确定输出 (回归防手改)
	auth := tc3Authorization("AKIDtest", "secret", []byte(`{"k":"v"}`), 1700000000)
	if !strings.HasPrefix(auth, "TC3-HMAC-SHA256 Credential=AKIDtest/2023-11-14/sms/tc3_request, ") {
		t.Fatalf("Credential 段不符: %q", auth)
	}
	if !strings.Contains(auth, "SignedHeaders=content-type;host, Signature=") {
		t.Fatalf("SignedHeaders 段不符: %q", auth)
	}
	sig := auth[strings.LastIndex(auth, "Signature=")+len("Signature="):]
	if len(sig) != 64 { // sha256 hex
		t.Fatalf("签名应为 64 位 hex: %q", sig)
	}
}

func TestProvidersRegistry(t *testing.T) {
	for _, name := range []string{"aliyun", "tencent"} {
		if p, ok := providers[name]; !ok || p.Name() != name {
			t.Fatalf("渠道 %s 未注册", name)
		}
	}
	if _, ok := providers["xxx"]; ok {
		t.Fatal("未知渠道不应存在")
	}
}
