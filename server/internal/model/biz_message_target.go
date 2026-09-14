package model

// BizMessageTarget 系统通知定向目标。
type BizMessageTarget struct {
	Id         uint64 `json:"id"         orm:"id"`
	MessageId  uint64 `json:"messageId"  orm:"message_id"`
	TargetType int    `json:"targetType" orm:"target_type"`
	TargetId   uint64 `json:"targetId"   orm:"target_id"`
}
