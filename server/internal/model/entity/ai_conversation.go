// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// AiConversation is the golang structure for table ai_conversation.
type AiConversation struct {
	Id           uint64      `json:"id"           orm:"id"            description:"主键ID"`                   // 主键ID
	SessionId    string      `json:"sessionId"    orm:"session_id"    description:"会话ID(前端生成并保管)"`          // 会话ID(前端生成并保管)
	UserId       uint64      `json:"userId"       orm:"user_id"       description:"所属用户ID"`                 // 所属用户ID
	Title        string      `json:"title"        orm:"title"         description:"会话标题(首条用户消息裁剪)"`         // 会话标题(首条用户消息裁剪)
	MessageCount uint        `json:"messageCount" orm:"message_count" description:"累计消息条数(user+assistant)"` // 累计消息条数(user+assistant)
	CreatedAt    *gtime.Time `json:"createdAt"    orm:"created_at"    description:"创建时间"`                   // 创建时间
	UpdatedAt    *gtime.Time `json:"updatedAt"    orm:"updated_at"    description:"最近一轮对话时间"`               // 最近一轮对话时间
	DeletedAt    *gtime.Time `json:"deletedAt"    orm:"deleted_at"    description:"未使用(保留列对齐代码生成器约定)"`      // 未使用(保留列对齐代码生成器约定)
}
