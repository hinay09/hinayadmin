// Package sms 短信发送服务 (验证码场景): 阿里云/腾讯云双实现, 标准库零依赖。
//
// 选择与参数在全局配置页 (sys_config 平台键 sms.*):
//
//	sms.provider            aliyun / tencent (空或其他=未启用, 发送直接报"短信服务未配置")
//	sms.sign_name           短信签名
//	sms.template_id         模板 CODE/ID (模板内容须含验证码变量)
//	sms.aliyun.access_key_id / sms.aliyun.access_key_secret   RAM 密钥
//	sms.tencent.secret_id   / sms.tencent.secret_key / sms.tencent.sdk_app_id  TC3 密钥与应用ID
//
// 接口只服务验证码场景: 模板参数约定为单个变量 "code" (阿里云 TemplateParam
// {"code":"123456"} / 腾讯云 TemplateParamList ["123456"]), 手机号自动补 +86 前缀。
// 配置读取走 configx 平台视角 (基础设施键, 租户不可覆盖)。
package sms

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"hinay.cn/admin/internal/logic/configx"
)

// readRandom 可注入的随机源 (测试替换用), 默认 crypto/rand。
var readRandom = rand.Read

// Provider 短信渠道实现。
type Provider interface {
	// Name 渠道标识 (aliyun / tencent)。
	Name() string
	// Send 发送验证码短信; cfg 为 sms.* 平台配置快照。
	Send(ctx context.Context, cfg map[string]string, phone, code string) error
}

// providers 已注册渠道。
var providers = map[string]Provider{
	"aliyun":  &aliyunProvider{},
	"tencent": &tencentProvider{},
}

// Send 按 sms.provider 配置发送验证码短信。
// 未配置/未识别的渠道返回错误 (调用方按业务语义决定吞掉或上抛)。
func Send(ctx context.Context, phone, code string) error {
	cfg := configx.PlatformPrefix(ctx, "sms.")
	name := strings.TrimSpace(cfg["sms.provider"])
	p, ok := providers[name]
	if !ok {
		return fmt.Errorf("短信服务未配置 (sms.provider=%q)", name)
	}
	return p.Send(ctx, cfg, phone, code)
}

// normalizePhone 大陆手机号统一 +86 E.164 格式 (腾讯云要求; 阿里云两者皆可)。
func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if strings.HasPrefix(phone, "+") {
		return phone
	}
	if strings.HasPrefix(phone, "86") && len(phone) == 13 {
		return "+" + phone
	}
	return "+86" + phone
}

// randNonce 请求唯一随机串 (防重放)。
func randNonce() string {
	b := make([]byte, 12)
	if _, err := readRandom(b); err != nil {
		return fmt.Sprintf("n%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
