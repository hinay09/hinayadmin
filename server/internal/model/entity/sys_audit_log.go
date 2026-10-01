// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysAuditLog is the golang structure for table sys_audit_log.
type SysAuditLog struct {
	Id         uint64      `json:"id"         orm:"id"          description:"ID"`                                          // ID
	UserId     uint64      `json:"userId"     orm:"user_id"     description:"用户ID"`                                        // 用户ID
	Username   string      `json:"username"   orm:"username"    description:"用户名"`                                         // 用户名
	Action     string      `json:"action"     orm:"action"      description:"操作类型(create/update/delete/upload/login/...)"` // 操作类型(create/update/delete/upload/login/...)
	Resource   string      `json:"resource"   orm:"resource"    description:"操作资源(如user/role/menu/dict/file)"`             // 操作资源(如user/role/menu/dict/file)
	ResourceId string      `json:"resourceId" orm:"resource_id" description:"资源标识"`                                        // 资源标识
	Method     string      `json:"method"     orm:"method"      description:"HTTP方法"`                                      // HTTP方法
	Path       string      `json:"path"       orm:"path"        description:"请求路径"`                                        // 请求路径
	StatusCode int         `json:"statusCode" orm:"status_code" description:"HTTP状态码"`                                     // HTTP状态码
	Code       int         `json:"code"       orm:"code"        description:"业务码(0=成功)"`                                   // 业务码(0=成功)
	Message    string      `json:"message"    orm:"message"     description:"业务消息/失败原因"`                                   // 业务消息/失败原因
	DurationMs int         `json:"durationMs" orm:"duration_ms" description:"耗时(毫秒)"`                                      // 耗时(毫秒)
	RequestId  string      `json:"requestId"  orm:"request_id"  description:"请求ID(链路追踪)"`                                  // 请求ID(链路追踪)
	Detail     string      `json:"detail"     orm:"detail"      description:"详情(脱敏后的请求体)"`                                 // 详情(脱敏后的请求体)
	Ip         string      `json:"ip"         orm:"ip"          description:"IP地址"`                                        // IP地址
	UserAgent  string      `json:"userAgent"  orm:"user_agent"  description:"User-Agent"`                                  // User-Agent
	CreatedAt  *gtime.Time `json:"createdAt"  orm:"created_at"  description:"创建时间"`                                        // 创建时间
}
