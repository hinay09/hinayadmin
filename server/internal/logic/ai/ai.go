// Package ai AI 助手 (OpenAI 兼容接口, 基于 langchaingo)。
package ai

import (
	"hinay.cn/admin/internal/service"
)

// sAi AI 助手服务实现 (具体能力分布在同包各文件: chat.go 对话 / tools.go 工具 / session.go 会话持久化)。
type sAi struct{}

func init() {
	service.RegisterAi(&sAi{})
}
