// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// AiChatMessage is the golang structure for table ai_chat_message.
type AiChatMessage struct {
	Id             uint64      `json:"id"             orm:"id"              description:"主键ID"`                     // 主键ID
	ConversationId uint64      `json:"conversationId" orm:"conversation_id" description:"会话ID(ai_conversation.id)"` // 会话ID(ai_conversation.id)
	Role           string      `json:"role"           orm:"role"            description:"角色:user/assistant"`        // 角色:user/assistant
	Content        string      `json:"content"        orm:"content"         description:"消息正文(思考过程不落库)"`            // 消息正文(思考过程不落库)
	CreatedAt      *gtime.Time `json:"createdAt"      orm:"created_at"      description:"创建时间"`                     // 创建时间
	UpdatedAt      *gtime.Time `json:"updatedAt"      orm:"updated_at"      description:"更新时间"`                     // 更新时间
	DeletedAt      *gtime.Time `json:"deletedAt"      orm:"deleted_at"      description:"未使用(保留列对齐代码生成器约定)"`        // 未使用(保留列对齐代码生成器约定)
}
