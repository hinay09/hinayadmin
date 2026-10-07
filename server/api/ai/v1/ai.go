// Package v1 AI 助手接口契约。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"
)

// AiChatReq AI 对话: 服务端以 SSE 流式返回增量内容、思考过程与工具调用步骤。
//
// 响应流格式 (Content-Type: text/event-stream):
//
//	data: {"content":"正文增量"}\n\n
//	data: {"reasoning":"思考增量"}\n\n      (推理模型, 如 DeepSeek-R1/QwQ)
//	data: {"toolStart":{...AiToolStep}}\n\n (工具开始执行; 含 id/name/args)
//	data: {"toolEnd":{...AiToolStep}}\n\n    (工具执行结束; 含 result 或 error)
//	data: {"usage":{...AiTokenUsage}}\n\n    (本轮成功收尾时发一次, 含 token 用量)
//	data: {"error":"错误原因"}\n\n            (仅出错时)
//	data: [DONE]\n\n
type AiChatReq struct {
	g.Meta `path:"/ai/chat" tags:"AI" method:"post" summary:"AI对话(流式)"`
	// 会话ID (前端生成并保管): 服务端按其持久化历史并维系上下文记忆
	// (MySQL, 上下文取最近 30 条); 留空 = 无记忆单轮
	SessionId string `json:"sessionId" v:"length:0,64#会话ID长度 0-64"`
	// 本轮新消息 (历史无需上送)
	Message string `json:"message" v:"required#消息内容不能为空"`
}

// AiChatRes 流式响应无 JSON 包装。
type AiChatRes struct{}

// AiToolStep 一次工具调用的步骤详情 (SSE toolStart/toolEnd 事件与历史回放共用同一结构)。
type AiToolStep struct {
	Id     string `json:"id"               dc:"工具调用ID"`
	Name   string `json:"name"             dc:"工具名"`
	Args   string `json:"args,omitempty"   dc:"调用参数 (JSON 文本)"`
	Result string `json:"result,omitempty" dc:"执行结果 (JSON 文本)"`
	Error  string `json:"error,omitempty"  dc:"执行错误 (执行失败时)"`
}

// AiTokenUsage 一轮对话的 token 用量 (工具循环各轮累计; 网关未回报时全 0, 前端隐藏不展示)。
type AiTokenUsage struct {
	Prompt     int `json:"prompt"     dc:"输入 token"`
	Completion int `json:"completion" dc:"输出 token"`
	Total      int `json:"total"      dc:"合计 token"`
}

// AiStreamEmit 流式下发五通道 (控制器实现为 SSE 帧; 思考/工具/用量通道可空)。
type AiStreamEmit struct {
	Content   func(delta string) error     // 正文增量
	Reasoning func(delta string) error     // 思考增量
	ToolStart func(step *AiToolStep) error // 工具开始执行 (前端渲染步骤条转圈)
	ToolEnd   func(step *AiToolStep) error // 工具执行结束 (带结果或错误)
	Usage     func(u *AiTokenUsage) error  // 本轮 token 用量 (成功收尾时发一次)
}

// AiHistoryMessage 会话历史消息 (工具步骤行 role=tool, 前端折叠渲染到其后紧邻的 assistant 气泡)。
type AiHistoryMessage struct {
	Role    string        `json:"role"`           // user / assistant / tool
	Content string        `json:"content"`        // 正文; role=tool 时为空
	Tool    *AiToolStep   `json:"tool,omitempty"` // role=tool: 步骤详情
	Usage   *AiTokenUsage `json:"usage,omitempty"` // role=assistant: 本轮 token 用量 (旧数据/网关未回报无)
}

// AiHistoryReq 拉取会话历史 (全量消息, 刷新页面后恢复对话界面; 思考过程不入历史)。
type AiHistoryReq struct {
	g.Meta    `path:"/ai/history" tags:"AI" method:"get" summary:"AI会话历史"`
	SessionId string `json:"sessionId" in:"query" v:"required#会话ID不能为空"`
}

// AiHistoryRes 历史响应。
type AiHistoryRes struct {
	List []AiHistoryMessage `json:"list"`
}

// AiSessionItem 会话列表项 (左侧历史会话栏)。
type AiSessionItem struct {
	SessionId string `json:"sessionId" dc:"会话ID"`
	Title     string `json:"title" dc:"会话标题 (首条用户消息裁剪)"`
	UpdatedAt int64  `json:"updatedAt" dc:"最后活跃时间 (Unix 秒)"`
}

// AiSessionsReq 当前用户的历史会话列表 (按最后活跃倒序)。
type AiSessionsReq struct {
	g.Meta `path:"/ai/sessions" tags:"AI" method:"get" summary:"AI会话列表"`
}

// AiSessionsRes 会话列表响应。
type AiSessionsRes struct {
	List []AiSessionItem `json:"list"`
}

// AiSessionDeleteReq 删除当前用户的一个会话 (校验归属, 会话与消息一并物理删除)。
type AiSessionDeleteReq struct {
	g.Meta    `path:"/ai/sessions" tags:"AI" method:"delete" summary:"AI会话删除"`
	SessionId string `json:"sessionId" in:"query" v:"required#会话ID不能为空"`
}

// AiSessionDeleteRes 删除响应。
type AiSessionDeleteRes struct{}

// AiConfigReq AI 配置状态 (供前端提示是否已配置)。
type AiConfigReq struct {
	g.Meta `path:"/ai/config" tags:"AI" method:"get" summary:"AI配置状态"`
}

// AiConfigRes 配置状态响应 (apiKey 脱敏)。
type AiConfigRes struct {
	Configured bool   `json:"configured"`
	BaseUrl    string `json:"baseUrl"`
	Model      string `json:"model"`
	KeyMasked  string `json:"keyMasked" dc:"脱敏 apiKey, 如 sk-***abcd"`
}
