// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// BizMessageTarget is the golang structure of table biz_message_target for DAO operations like Where/Data.
type BizMessageTarget struct {
	g.Meta     `orm:"table:biz_message_target, do:true"`
	Id         any //
	MessageId  any // 消息ID
	TargetType any // 目标类型:2=role,3=user
	TargetId   any // 角色ID或用户ID
}
