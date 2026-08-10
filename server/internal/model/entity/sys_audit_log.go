// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysAuditLog is the golang structure for table sys_audit_log.
type SysAuditLog struct {
	Id         uint64      `json:"id"         orm:"id"          description:"ID"`                                    // ID
	UserId     uint64      `json:"userId"     orm:"user_id"     description:"用户ID"`                                  // 用户ID
	Username   string      `json:"username"   orm:"username"    description:"用户名"`                                   // 用户名
	Action     string      `json:"action"     orm:"action"      description:"操作类型(create/update/delete/upload/...)"` // 操作类型(create/update/delete/upload/...)
	Resource   string      `json:"resource"   orm:"resource"    description:"操作资源(如user/role/menu/dict/file)"`       // 操作资源(如user/role/menu/dict/file)
	ResourceId string      `json:"resourceId" orm:"resource_id" description:"资源标识"`                                  // 资源标识
	Detail     string      `json:"detail"     orm:"detail"      description:"详情(JSON格式)"`                            // 详情(JSON格式)
	Ip         string      `json:"ip"         orm:"ip"          description:"IP地址"`                                  // IP地址
	UserAgent  string      `json:"userAgent"  orm:"user_agent"  description:"User-Agent"`                            // User-Agent
	CreatedAt  *gtime.Time `json:"createdAt"  orm:"created_at"  description:"创建时间"`                                  // 创建时间
}
