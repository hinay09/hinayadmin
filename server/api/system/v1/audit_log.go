// Package v1 系统管理-操作日志接口。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/utility/response"
)

// AuditLogListReq 操作日志分页列表。
type AuditLogListReq struct {
	g.Meta   `path:"/system/audit-logs" tags:"SystemAuditLog" method:"get" summary:"操作日志分页列表"`
	Keyword  string `json:"keyword"  in:"query" dc:"用户名/操作/资源模糊"`
	Action   string `json:"action"   in:"query" dc:"操作类型精确过滤"`
	StartAt  string `json:"startAt" in:"query" dc:"开始时间"`
	EndAt    string `json:"endAt"   in:"query" dc:"结束时间"`
	Page     int    `json:"page"     in:"query" d:"1"  dc:"页码"`
	PageSize int    `json:"pageSize" in:"query" d:"10" dc:"页大小"`
}

// AuditLogListRes 列表响应。
type AuditLogListRes response.PageResult
