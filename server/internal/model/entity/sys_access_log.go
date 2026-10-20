// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysAccessLog is the golang structure for table sys_access_log.
type SysAccessLog struct {
	Id         uint64      `json:"id"         orm:"id"          description:"ID"`                               // ID
	TenantId   uint64      `json:"tenantId"   orm:"tenant_id"   description:"租户ID(记录归属, 平台可查全量)"`               // 租户ID(记录归属, 平台可查全量)
	UserId     uint64      `json:"userId"     orm:"user_id"     description:"用户ID(匿名请求为0)"`                     // 用户ID(匿名请求为0)
	Username   string      `json:"username"   orm:"username"    description:"用户名(匿名请求为空)"`                      // 用户名(匿名请求为空)
	Method     string      `json:"method"     orm:"method"      description:"HTTP方法"`                           // HTTP方法
	Path       string      `json:"path"       orm:"path"        description:"请求路径"`                             // 请求路径
	Query      string      `json:"query"      orm:"query"       description:"URL查询串(GET筛选条件)"`                  // URL查询串(GET筛选条件)
	StatusCode int         `json:"statusCode" orm:"status_code" description:"HTTP状态码"`                          // HTTP状态码
	Code       int         `json:"code"       orm:"code"        description:"业务码(0=成功; 二进制/超大响应不解析记0)"`         // 业务码(0=成功; 二进制/超大响应不解析记0)
	DurationMs int         `json:"durationMs" orm:"duration_ms" description:"耗时(毫秒)"`                           // 耗时(毫秒)
	Slow       int         `json:"slow"       orm:"slow"        description:"慢接口标记:1=耗时超过阈值(accessLog.slowMs)"` // 慢接口标记:1=耗时超过阈值(accessLog.slowMs)
	RequestId  string      `json:"requestId"  orm:"request_id"  description:"请求ID(链路追踪)"`                       // 请求ID(链路追踪)
	Ip         string      `json:"ip"         orm:"ip"          description:"IP地址"`                             // IP地址
	UserAgent  string      `json:"userAgent"  orm:"user_agent"  description:"User-Agent"`                       // User-Agent
	CreatedAt  *gtime.Time `json:"createdAt"  orm:"created_at"  description:"创建时间"`                             // 创建时间
}
