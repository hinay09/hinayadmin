package twofactor

import (
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TestNormalizeCode 动态码归一化: 容忍空格, 必须恰好 6 位数字。
func TestNormalizeCode(t *testing.T) {
	ok, err := normalizeCode("123 456")
	if err != nil || ok != "123456" {
		t.Fatalf("normalizeCode(123 456) = %q, %v; want 123456, nil", ok, err)
	}
	if _, err = normalizeCode("12345"); err == nil {
		t.Fatalf("5 位数字应报错")
	}
	if _, err = normalizeCode("12345a"); err == nil {
		t.Fatalf("含字母应报错")
	}
	if _, err = normalizeCode("1234567"); err == nil {
		t.Fatalf("7 位数字应报错")
	}
}

// TestMatchCode 校验命中/防重放/±1 窗口容差。
func TestMatchCode(t *testing.T) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer: "test", AccountName: "u1",
		Period: totpPeriod, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	secret := key.Secret()
	now := time.Now()

	// 当前窗口码: 命中
	code, err := totp.GenerateCode(secret, now)
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}
	step, ok := matchCode(secret, code, now, 0)
	if !ok {
		t.Fatalf("当前窗口验证码应命中")
	}

	// 重放: last_step 已推进后, 同一验证码被拒绝
	if _, ok = matchCode(secret, code, now, step); ok {
		t.Fatalf("同一验证码二次使用应被拒绝(重放)")
	}

	// 时钟偏移: 慢 30s 的客户端代码在 ±1 窗口容差内命中
	slowCode, _ := totp.GenerateCode(secret, now.Add(-time.Duration(totpPeriod)*time.Second))
	if _, ok = matchCode(secret, slowCode, now, 0); !ok {
		t.Fatalf("慢 30s 的验证码应在容差窗口内命中")
	}
	fastCode, _ := totp.GenerateCode(secret, now.Add(time.Duration(totpPeriod)*time.Second))
	if _, ok = matchCode(secret, fastCode, now, 0); !ok {
		t.Fatalf("快 30s 的验证码应在容差窗口内命中")
	}
	// 超出 ±1 窗口 (±60s) 的码拒绝
	farCode, _ := totp.GenerateCode(secret, now.Add(2*time.Duration(totpPeriod)*time.Second))
	if _, ok = matchCode(secret, farCode, now, 0); ok {
		t.Fatalf("快 60s 的验证码应被拒绝")
	}

	// 错误验证码
	if _, ok = matchCode(secret, "000000", now, 0); ok {
		// 000000 恰为当前码的极小概率事件: 重算一个必然不同的码
		code2, _ := totp.GenerateCode(secret, now.Add(90*time.Duration(totpPeriod)*time.Second))
		if _, ok = matchCode(secret, code2, now, 0); ok {
			t.Fatalf("未来窗口外的验证码不应命中")
		}
	}
}
