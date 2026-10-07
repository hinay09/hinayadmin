// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish to manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	v1 "hinay.cn/admin/api/ai/v1"
)

type (
	IAi interface {
		// ChatStream AI 对话: 逐段回调增量内容与思考过程 (emit 由控制器实现为 SSE 写出)。
		ChatStream(ctx context.Context, in *v1.AiChatReq, emit *v1.AiStreamEmit) error
		// AiConfig 配置状态 (是否已配置/模型/脱敏 key)。
		AiConfig(ctx context.Context, in *v1.AiConfigReq) (res *v1.AiConfigRes, err error)
		// AiHistory 会话历史 (恢复对话界面)。
		AiHistory(ctx context.Context, in *v1.AiHistoryReq) (res *v1.AiHistoryRes, err error)
		// AiSessions 当前用户的历史会话列表 (左侧历史会话栏)。
		AiSessions(ctx context.Context, in *v1.AiSessionsReq) (res *v1.AiSessionsRes, err error)
		// AiSessionDelete 删除当前用户的一个会话 (校验归属)。
		AiSessionDelete(ctx context.Context, in *v1.AiSessionDeleteReq) (res *v1.AiSessionDeleteRes, err error)
	}
)

var (
	localAi IAi
)

func Ai() IAi {
	if localAi == nil {
		panic("implement not found for interface IAi, forgot register?")
	}
	return localAi
}

func RegisterAi(i IAi) {
	localAi = i
}
