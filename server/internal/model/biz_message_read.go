package model

import "github.com/gogf/gf/v2/os/gtime"

// BizMessageRead 消息已读关系。
type BizMessageRead struct {
	Id        uint64      `json:"id"        orm:"id"`
	MessageId uint64      `json:"messageId" orm:"message_id"`
	UserId    uint64      `json:"userId"    orm:"user_id"`
	ReadAt    *gtime.Time `json:"readAt"    orm:"read_at"`
	Hidden    int         `json:"hidden"    orm:"hidden"`
}
