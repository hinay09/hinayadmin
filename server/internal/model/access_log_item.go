package model

import "github.com/gogf/gf/v2/os/gtime"

// AccessLogItem 接口日志列表 VO。
type AccessLogItem struct {
	Id         uint64      `json:"id"`
	TenantId   uint64      `json:"tenantId"`
	TenantName string      `json:"tenantName" dc:"所属租户名(平台视角混排展示)"`
	UserId     uint64      `json:"userId"`
	Username   string      `json:"username"`
	Method     string      `json:"method"`
	Path       string      `json:"path"`
	Query      string      `json:"query"`
	StatusCode int         `json:"statusCode"`
	Code       int         `json:"code"`
	DurationMs int64       `json:"durationMs"`
	Slow       int         `json:"slow"`
	RequestId  string      `json:"requestId"`
	Ip         string      `json:"ip"`
	UserAgent  string      `json:"userAgent"`
	CreatedAt  *gtime.Time `json:"createdAt"`
}
