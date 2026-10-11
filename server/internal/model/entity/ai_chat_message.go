// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// AiChatMessage is the golang structure for table ai_chat_message.
type AiChatMessage struct {
	Id               uint64      `json:"id"               orm:"id"                description:"主键ID"`                                          // 主键ID
	TenantId         uint64      `json:"tenantId"         orm:"tenant_id"         description:"租户ID(跟随会话, ORM注入)"`                             // 租户ID(跟随会话, ORM注入)
	ConversationId   uint64      `json:"conversationId"   orm:"conversation_id"   description:"会话ID(ai_conversation.id)"`                      // 会话ID(ai_conversation.id)
	Role             string      `json:"role"             orm:"role"              description:"角色:user/assistant/tool(tool=工具调用步骤,内容为步骤JSON)"` // 角色:user/assistant/tool(tool=工具调用步骤,内容为步骤JSON)
	Content          string      `json:"content"          orm:"content"           description:"消息正文(思考过程不落库)"`                                 // 消息正文(思考过程不落库)
	PromptTokens     uint        `json:"promptTokens"     orm:"prompt_tokens"     description:"输入token用量(仅assistant行,本轮累计)"`                   // 输入token用量(仅assistant行,本轮累计)
	CompletionTokens uint        `json:"completionTokens" orm:"completion_tokens" description:"输出token用量(仅assistant行,本轮累计)"`                   // 输出token用量(仅assistant行,本轮累计)
	TotalTokens      uint        `json:"totalTokens"      orm:"total_tokens"      description:"总token用量(仅assistant行;网关未回报为0)"`                 // 总token用量(仅assistant行;网关未回报为0)
	CreatedAt        *gtime.Time `json:"createdAt"        orm:"created_at"        description:"创建时间"`                                          // 创建时间
	UpdatedAt        *gtime.Time `json:"updatedAt"        orm:"updated_at"        description:"更新时间"`                                          // 更新时间
	DeletedAt        *gtime.Time `json:"deletedAt"        orm:"deleted_at"        description:"未使用(保留列对齐代码生成器约定)"`                             // 未使用(保留列对齐代码生成器约定)
}
