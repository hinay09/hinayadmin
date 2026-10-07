// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// AiChatMessage is the golang structure of table ai_chat_message for DAO operations like Where/Data.
type AiChatMessage struct {
	g.Meta         `orm:"table:ai_chat_message, do:true"`
	Id             any         // 主键ID
	ConversationId any         // 会话ID(ai_conversation.id)
	Role           any         // 角色:user/assistant
	Content        any         // 消息正文(思考过程不落库)
	CreatedAt      *gtime.Time // 创建时间
	UpdatedAt      *gtime.Time // 更新时间
	DeletedAt      *gtime.Time // 未使用(保留列对齐代码生成器约定)
}
