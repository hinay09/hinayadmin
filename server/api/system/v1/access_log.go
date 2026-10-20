// Package v1 系统管理-接口日志接口。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/utility/response"
)

// AccessLogListReq 接口日志分页列表(全方法访问流水)。
type AccessLogListReq struct {
	g.Meta      `path:"/system/access-logs" tags:"SystemAccessLog" method:"get" summary:"接口日志分页列表"`
	Keyword     string `json:"keyword"     in:"query" dc:"用户名/路径/RequestId模糊"`
	Username    string `json:"username"    in:"query" dc:"用户名精确过滤"`
	Method      string `json:"method"      in:"query" dc:"HTTP方法精确过滤"`
	Slow        int    `json:"slow"        in:"query" dc:"慢接口筛选:1=仅慢接口" v:"in:,0,1"`
	Result      string `json:"result"      in:"query" dc:"结果过滤: success/fail" v:"in:,success,fail"`
	MinDuration int    `json:"minDuration" in:"query" dc:"最小耗时(毫秒)"`
	StartAt     string `json:"startAt"     in:"query" dc:"开始时间"`
	EndAt       string `json:"endAt"       in:"query" dc:"结束时间"`
	Page        int    `json:"page"        in:"query" d:"1"  dc:"页码"`
	PageSize    int    `json:"pageSize"    in:"query" d:"10" dc:"页大小"`
}

// AccessLogListRes 列表响应。
type AccessLogListRes response.PageResult
