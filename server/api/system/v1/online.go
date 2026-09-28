// Package v1 系统管理-在线用户接口。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/utility/response"
)

// OnlineListReq 在线用户分页列表。
type OnlineListReq struct {
	g.Meta   `path:"/system/online" tags:"SystemOnline" method:"get" summary:"在线用户分页列表"`
	Username string `json:"username" in:"query" dc:"用户名模糊过滤"`
	Page     int    `json:"page"     in:"query" d:"1"  dc:"页码"`
	PageSize int    `json:"pageSize" in:"query" d:"10" dc:"页大小"`
}

// OnlineListRes 列表响应。
type OnlineListRes response.PageResult

// OnlineKickReq 强制下线指定会话。
type OnlineKickReq struct {
	g.Meta `path:"/system/online/:id" tags:"SystemOnline" method:"delete" summary:"强制下线"`
	Id     string `v:"required#缺少会话标识" in:"path" dc:"会话ID"`
}

// OnlineKickRes 强制下线响应。
type OnlineKickRes struct{}
