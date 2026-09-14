package model

import "github.com/gogf/gf/v2/os/gtime"

// 消息通知: 类型与范围枚举。
const (
	MessageTypeSystem  = 1 // 系统通知
	MessageTypePrivate = 2 // 私信通知

	MessageScopeNone = 0 // 私信专用
	MessageScopeAll  = 1 // 全员
	MessageScopeRole = 2 // 指定角色
	MessageScopeUser = 3 // 指定用户

	MessageTargetTypeRole = 2
	MessageTargetTypeUser = 3
)

// BizMessage 消息通知主表实体。
type BizMessage struct {
	Id          uint64      `json:"id"          orm:"id"`
	Type        int         `json:"type"        orm:"type"`
	Title       string      `json:"title"       orm:"title"`
	Content     string      `json:"content"     orm:"content"`
	Level       int         `json:"level"       orm:"level"`
	SenderId    uint64      `json:"senderId"    orm:"sender_id"`
	TargetScope int         `json:"targetScope" orm:"target_scope"`
	ReceiverId  uint64      `json:"receiverId"  orm:"receiver_id"`
	Status      int         `json:"status"      orm:"status"`
	CreatedAt   *gtime.Time `json:"createdAt"   orm:"created_at"`
	UpdatedAt   *gtime.Time `json:"updatedAt"   orm:"updated_at"`
}
