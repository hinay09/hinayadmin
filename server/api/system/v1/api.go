// Package v1 系统管理-API接口。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/utility/response"
)

// API 资源管理

type ApiListReq struct {
	g.Meta    `path:"/system/apis" method:"get" tags:"API管理" summary:"API列表"`
	GroupName string `json:"groupName" in:"query" dc:"分组名称筛选"`
	Path      string `json:"path"      in:"query" dc:"路径筛选"`
	Page      int    `json:"page"      in:"query" d:"1"  dc:"页码"`
	PageSize  int    `json:"pageSize"  in:"query" d:"10" dc:"页大小"`
}
type ApiListRes response.PageResult
type ApiItem struct {
	Id          int64  `json:"id"`
	Path        string `json:"path"`
	Method      string `json:"method"`
	GroupName   string `json:"groupName"`
	Description string `json:"description"`
}

// ApiAllReq 全量 API（不分页，用于权限分配等场景）。
type ApiAllReq struct {
	g.Meta `path:"/system/apis/all" method:"get" tags:"API管理" summary:"全量API"`
}
type ApiAllRes struct {
	List []ApiItem `json:"list"`
}

type ApiCreateReq struct {
	g.Meta      `path:"/system/apis" method:"post" tags:"API管理" summary:"创建API"`
	Path        string `json:"path" v:"required" dc:"API路径"`
	Method      string `json:"method" v:"required|in:GET,POST,PUT,DELETE" dc:"HTTP方法"`
	GroupName   string `json:"groupName" dc:"分组名称"`
	Description string `json:"description" dc:"接口描述"`
}
type ApiCreateRes struct{}

type ApiUpdateReq struct {
	g.Meta      `path:"/system/apis/{id}" method:"put" tags:"API管理" summary:"更新API"`
	Id          int64  `json:"id" in:"path" v:"required"`
	Path        string `json:"path" v:"required" dc:"API路径"`
	Method      string `json:"method" v:"required|in:GET,POST,PUT,DELETE" dc:"HTTP方法"`
	GroupName   string `json:"groupName" dc:"分组名称"`
	Description string `json:"description" dc:"接口描述"`
}
type ApiUpdateRes struct{}

type ApiDeleteReq struct {
	g.Meta `path:"/system/apis/{id}" method:"delete" tags:"API管理" summary:"删除API"`
	Id     int64 `json:"id" in:"path" v:"required"`
}
type ApiDeleteRes struct{}
