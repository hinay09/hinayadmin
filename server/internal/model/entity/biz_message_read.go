// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// BizMessageRead is the golang structure for table biz_message_read.
type BizMessageRead struct {
	Id        uint64      `json:"id"        orm:"id"         description:""`                     //
	MessageId uint64      `json:"messageId" orm:"message_id" description:"消息ID"`                 // 消息ID
	UserId    uint64      `json:"userId"    orm:"user_id"    description:"用户ID"`                 // 用户ID
	ReadAt    *gtime.Time `json:"readAt"    orm:"read_at"    description:"阅读时间"`                 // 阅读时间
	Hidden    int         `json:"hidden"    orm:"hidden"     description:"是否在收件箱视角隐藏(个人删除):1=是"` // 是否在收件箱视角隐藏(个人删除):1=是
}
