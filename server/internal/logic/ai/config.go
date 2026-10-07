// Package ai — 配置读取 (sys_config 键值, 管理员在「全局配置」页维护)。
package ai

import (
	"context"
	"strconv"
	"strings"

	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/utility/xerror"
)

// 配置键 (sys_config)。
const (
	cfgKeyAPIKey   = "ai.api_key"       // OpenAI 兼容 API Key (必填才可用)
	cfgKeyBaseURL  = "ai.base_url"      // 接口地址, 默认 OpenAI 官方
	cfgKeyModel    = "ai.model"         // 模型名
	cfgKeyTemp     = "ai.temperature"   // 温度
	cfgKeyPrompt   = "ai.system_prompt" // 系统提示词 (可空)
	defaultBaseURL = "https://api.openai.com/v1"
	defaultModel   = "gpt-4o-mini"
)

// aiConfig 运行时配置快照。
type aiConfig struct {
	apiKey       string
	baseURL      string
	model        string
	systemPrompt string
	temperature  float64
}

// loadAiConfig 读取配置; 未配置 apiKey 视为功能未开通。
func loadAiConfig(ctx context.Context) (*aiConfig, error) {
	c := &aiConfig{
		apiKey:       strings.TrimSpace(cfgOf(ctx, cfgKeyAPIKey, "")),
		baseURL:      strings.TrimSpace(cfgOf(ctx, cfgKeyBaseURL, defaultBaseURL)),
		model:        strings.TrimSpace(cfgOf(ctx, cfgKeyModel, defaultModel)),
		systemPrompt: cfgOf(ctx, cfgKeyPrompt, ""),
		temperature:  0.7,
	}
	if t := strings.TrimSpace(cfgOf(ctx, cfgKeyTemp, "0.7")); t != "" {
		if f, err := strconv.ParseFloat(t, 64); err == nil && f >= 0 && f <= 2 {
			c.temperature = f
		}
	}
	if c.apiKey == "" {
		return nil, xerror.New(xerror.CodeBusinessError,
			"AI 服务未配置: 请管理员在 系统管理-全局配置 中设置 ai.api_key (OpenAI 兼容接口)")
	}
	return c, nil
}

// cfgOf 按 key 取配置值, 空/缺失回落默认值。
func cfgOf(ctx context.Context, key, def string) string {
	v, err := dao.SysConfig.Ctx(ctx).
		Where("config_key", key).Where("deleted_at IS NULL").
		Value("config_value")
	if err != nil || v == nil {
		return def
	}
	s := v.String()
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

// maskKey apiKey 脱敏显示。
func maskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "***"
	}
	return key[:3] + "***" + key[len(key)-4:]
}
