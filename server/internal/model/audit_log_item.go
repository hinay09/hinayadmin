package model

import "github.com/gogf/gf/v2/os/gtime"

// AuditLogItem 操作日志列表 VO。
type AuditLogItem struct {
	Id         uint64      `json:"id"`
	UserId     uint64      `json:"userId"`
	Username   string      `json:"username"`
	Action     string      `json:"action"`
	Resource   string      `json:"resource"`
	ResourceId string      `json:"resourceId"`
	Method     string      `json:"method"`
	Path       string      `json:"path"`
	StatusCode int         `json:"statusCode"`
	Code       int         `json:"code"`
	Message    string      `json:"message"`
	DurationMs int64       `json:"durationMs"`
	RequestId  string      `json:"requestId"`
	Detail     string      `json:"detail"`
	Ip         string      `json:"ip"`
	UserAgent  string      `json:"userAgent"`
	CreatedAt  *gtime.Time `json:"createdAt"`
}
