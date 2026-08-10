// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// BizMessage is the golang structure of table biz_message for DAO operations like Where/Data.
type BizMessage struct {
	g.Meta      `orm:"table:biz_message, do:true"`
	Id          any         //
	Type        any         // 消息类型:1=系统通知,2=私信
	Title       any         // 标题
	Content     any         // 内容
	Level       any         // 级别:1=普通,2=重要,3=紧急
	SenderId    any         // 发送人ID
	TargetScope any         // 系统通知范围:1=all,2=role,3=user;私信=0
	ReceiverId  any         // 私信接收者ID
	Status      any         // 状态:1=已发布,0=草稿
	CreatedAt   *gtime.Time // 创建时间
	UpdatedAt   *gtime.Time // 更新时间
	DeletedAt   *gtime.Time // 删除时间(软删)
}
