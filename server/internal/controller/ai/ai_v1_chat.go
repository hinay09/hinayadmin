// Package ai AI 助手控制器 — 对话走 SSE 流式直写。
package ai

import (
	"context"
	"encoding/json"

	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/api/ai/v1"
	"hinay.cn/admin/internal/service"
)

// AiChat AI 对话: SSE 流式返回增量内容、思考过程与工具调用步骤。
//
// 流帧格式: data: {"content"|"reasoning"|"toolStart"|"toolEnd"|"error":...} / data: [DONE]。
// 校验失败(参数/鉴权)在进入本方法前已被标准协议拦截为 JSON 错误;
// 进入流后的问题 (未配置/上游失败) 以 error 帧下发, 前端提示后保持会话可继续。
func (c *ControllerV1) AiChat(ctx context.Context, req *v1.AiChatReq) (res *v1.AiChatRes, err error) {
	r := g.RequestFromCtx(ctx)
	r.Response.Header().Set("Content-Type", "text/event-stream")
	r.Response.Header().Set("Cache-Control", "no-cache")
	r.Response.Header().Set("X-Accel-Buffering", "no")

	write := func(payload any) {
		data, _ := json.Marshal(payload)
		r.Response.Write("data: " + string(data) + "\n\n")
		r.Response.Flush()
	}

	if serr := service.Ai().ChatStream(ctx, req, &v1.AiStreamEmit{
		Content: func(delta string) error {
			write(map[string]string{"content": delta})
			return nil
		},
		Reasoning: func(delta string) error {
			write(map[string]string{"reasoning": delta})
			return nil
		},
		ToolStart: func(step *v1.AiToolStep) error {
			write(map[string]*v1.AiToolStep{"toolStart": step})
			return nil
		},
		ToolEnd: func(step *v1.AiToolStep) error {
			write(map[string]*v1.AiToolStep{"toolEnd": step})
			return nil
		},
		Usage: func(u *v1.AiTokenUsage) error {
			write(map[string]*v1.AiTokenUsage{"usage": u})
			return nil
		},
	}); serr != nil {
		write(map[string]string{"error": serr.Error()})
	}

	r.Response.Write("data: [DONE]\n\n")
	r.Response.Flush()
	return nil, nil
}

// AiConfig AI 配置状态。
func (c *ControllerV1) AiConfig(ctx context.Context, req *v1.AiConfigReq) (res *v1.AiConfigRes, err error) {
	return service.Ai().AiConfig(ctx, req)
}

// AiHistory 会话历史。
func (c *ControllerV1) AiHistory(ctx context.Context, req *v1.AiHistoryReq) (res *v1.AiHistoryRes, err error) {
	return service.Ai().AiHistory(ctx, req)
}

// AiSessions 当前用户的历史会话列表。
func (c *ControllerV1) AiSessions(ctx context.Context, req *v1.AiSessionsReq) (res *v1.AiSessionsRes, err error) {
	return service.Ai().AiSessions(ctx, req)
}

// AiSessionDelete 删除当前用户的一个会话。
func (c *ControllerV1) AiSessionDelete(ctx context.Context, req *v1.AiSessionDeleteReq) (res *v1.AiSessionDeleteRes, err error) {
	return service.Ai().AiSessionDelete(ctx, req)
}
