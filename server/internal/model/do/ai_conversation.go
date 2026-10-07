// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// AiConversation is the golang structure of table ai_conversation for DAO operations like Where/Data.
type AiConversation struct {
	g.Meta       `orm:"table:ai_conversation, do:true"`
	Id           any         // 主键ID
	SessionId    any         // 会话ID(前端生成并保管)
	UserId       any         // 所属用户ID
	Title        any         // 会话标题(首条用户消息裁剪)
	MessageCount any         // 累计消息条数(user+assistant)
	CreatedAt    *gtime.Time // 创建时间
	UpdatedAt    *gtime.Time // 最近一轮对话时间
	DeletedAt    *gtime.Time // 未使用(保留列对齐代码生成器约定)
}
