package model

import "github.com/gogf/gf/v2/os/gtime"

// MessageItem 消息列表/详情 VO (面向接口)。
type MessageItem struct {
	Id              uint64      `json:"id"`
	Type            int         `json:"type"`
	Title           string      `json:"title"`
	Content         string      `json:"content"`
	Level           int         `json:"level"`
	Status          int         `json:"status"`
	SenderId        uint64      `json:"senderId"`
	SenderName      string      `json:"senderName"`
	TargetScope     int         `json:"targetScope"`
	ReceiverId      uint64      `json:"receiverId"`
	ReceiverName    string      `json:"receiverName"`
	TargetRoleIds   []uint64    `json:"targetRoleIds"`
	TargetUserIds   []uint64    `json:"targetUserIds"`
	TargetRoleNames []string    `json:"targetRoleNames"`
	IsRead          bool        `json:"isRead"`
	CreatedAt       *gtime.Time `json:"createdAt"`
	UpdatedAt       *gtime.Time `json:"updatedAt"`
}
