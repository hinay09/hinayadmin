package model

import "github.com/gogf/gf/v2/os/gtime"

// SysAuditLog 操作日志实体。
type SysAuditLog struct {
	Id         uint64      `json:"id"         orm:"id"`
	UserId     uint64      `json:"userId"     orm:"user_id"`
	Username   string      `json:"username"   orm:"username"`
	Action     string      `json:"action"     orm:"action"`
	Resource   string      `json:"resource"   orm:"resource"`
	ResourceId string      `json:"resourceId" orm:"resource_id"`
	Detail     string      `json:"detail"     orm:"detail"`
	Ip         string      `json:"ip"         orm:"ip"`
	UserAgent  string      `json:"userAgent"  orm:"user_agent"`
	CreatedAt  *gtime.Time `json:"createdAt"  orm:"created_at"`
}
