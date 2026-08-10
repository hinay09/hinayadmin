// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// BizMessageRead is the golang structure of table biz_message_read for DAO operations like Where/Data.
type BizMessageRead struct {
	g.Meta    `orm:"table:biz_message_read, do:true"`
	Id        any         //
	MessageId any         // 消息ID
	UserId    any         // 用户ID
	ReadAt    *gtime.Time // 阅读时间
	Hidden    any         // 是否在收件箱视角隐藏(个人删除):1=是
}
