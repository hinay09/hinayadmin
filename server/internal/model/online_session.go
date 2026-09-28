// Package model 在线会话模型。
package model

import "github.com/gogf/gf/v2/os/gtime"

// OnlineSession Redis 中存储的在线会话记录 (JSON 序列化后作为 HASH value)。
// Token 仅存于服务端 Redis, 列表 VO 不下发。
type OnlineSession struct {
	Token        string `json:"token"` // 原始 JWT, 强制下线时写入黑名单用
	UserId       uint64 `json:"userId"`
	Username     string `json:"username"`
	Nickname     string `json:"nickname"`
	Ip           string `json:"ip"`
	UserAgent    string `json:"userAgent"`
	LoginAt      int64  `json:"loginAt"`      // 登录时间(秒)
	LastActiveAt int64  `json:"lastActiveAt"` // 最近活跃时间(秒)
}

// OnlineUserItem 在线用户列表 VO (不含 token)。
type OnlineUserItem struct {
	SessionId    string      `json:"sessionId"`
	UserId       uint64      `json:"userId"`
	Username     string      `json:"username"`
	Nickname     string      `json:"nickname"`
	Ip           string      `json:"ip"`
	UserAgent    string      `json:"userAgent"`
	LoginAt      *gtime.Time `json:"loginAt"`
	LastActiveAt *gtime.Time `json:"lastActiveAt"`
}
