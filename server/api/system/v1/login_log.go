// Package v1 系统管理-登录日志接口。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/utility/response"
)

// LoginLogListReq 登录日志分页列表。
type LoginLogListReq struct {
	g.Meta   `path:"/system/login-logs" tags:"SystemLoginLog" method:"get" summary:"登录日志分页列表"`
	Username string `json:"username" in:"query" dc:"用户名模糊"`
	Ip       string `json:"ip"       in:"query" dc:"IP 模糊"`
	Status   string `json:"status"   in:"query" dc:"结果过滤: success/fail" v:"in:,success,fail"`
	StartAt  string `json:"startAt"  in:"query" dc:"开始时间"`
	EndAt    string `json:"endAt"    in:"query" dc:"结束时间"`
	Page     int    `json:"page"     in:"query" d:"1"  dc:"页码"`
	PageSize int    `json:"pageSize" in:"query" d:"10" dc:"页大小"`
}

// LoginLogListRes 列表响应。
type LoginLogListRes response.PageResult

// LoginLogDeleteReq 删除登录日志(按 ID)。
type LoginLogDeleteReq struct {
	g.Meta `path:"/system/login-logs/:id" tags:"SystemLoginLog" method:"delete" summary:"删除登录日志"`
	Id     uint64 `v:"min:1" in:"path" dc:"日志ID"`
}

// LoginLogDeleteRes 删除响应。
type LoginLogDeleteRes struct{}
