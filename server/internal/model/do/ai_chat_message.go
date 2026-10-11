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
	g.Meta           `orm:"table:ai_chat_message, do:true"`
	Id               any         // 主键ID
	TenantId         any         // 租户ID(跟随会话, ORM注入)
	ConversationId   any         // 会话ID(ai_conversation.id)
	Role             any         // 角色:user/assistant/tool(tool=工具调用步骤,内容为步骤JSON)
	Content          any         // 消息正文(思考过程不落库)
	PromptTokens     any         // 输入token用量(仅assistant行,本轮累计)
	CompletionTokens any         // 输出token用量(仅assistant行,本轮累计)
	TotalTokens      any         // 总token用量(仅assistant行;网关未回报为0)
	CreatedAt        *gtime.Time // 创建时间
	UpdatedAt        *gtime.Time // 更新时间
	DeletedAt        *gtime.Time // 未使用(保留列对齐代码生成器约定)
}
