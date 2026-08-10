// Package v1 系统管理-角色接口。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/utility/response"
)

// RoleListReq 角色分页。
type RoleListReq struct {
	g.Meta   `path:"/system/roles" tags:"SystemRole" method:"get" summary:"角色分页列表"`
	Keyword  string `json:"keyword"  in:"query" dc:"名称/编码模糊"`
	Status   *int   `json:"status"   in:"query"`
	Page     int    `json:"page"     in:"query" d:"1"`
	PageSize int    `json:"pageSize" in:"query" d:"10"`
}

// RoleListRes 列表响应。
type RoleListRes response.PageResult

// RoleAllReq 全量角色。
type RoleAllReq struct {
	g.Meta `path:"/system/roles/all" tags:"SystemRole" method:"get" summary:"全量角色"`
}

// RoleAllRes 全量。
type RoleAllRes struct {
	List any `json:"list"`
}

// RoleDetailReq 详情。
type RoleDetailReq struct {
	g.Meta `path:"/system/roles/{id}" tags:"SystemRole" method:"get" summary:"角色详情"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// RoleDetailRes 详情响应。
type RoleDetailRes struct {
	Role    any      `json:"role"`
	MenuIds []uint64 `json:"menuIds"`
}

// RoleCreateReq 新增。
type RoleCreateReq struct {
	g.Meta `path:"/system/roles" tags:"SystemRole" method:"post" summary:"新增角色"`
	Name   string `json:"name"   v:"required|length:2,32"`
	Code   string `json:"code"   v:"required|length:2,32"`
	Sort   int    `json:"sort"`
	Status int    `json:"status" d:"1"`
	Remark string `json:"remark"`
}

// RoleCreateRes 新增响应。
type RoleCreateRes struct {
	Id uint64 `json:"id"`
}

// RoleUpdateReq 修改。
type RoleUpdateReq struct {
	g.Meta `path:"/system/roles/{id}" tags:"SystemRole" method:"put" summary:"修改角色"`
	Id     uint64 `json:"id" in:"path" v:"required"`
	Name   string `json:"name"   v:"required|length:2,32"`
	Sort   int    `json:"sort"`
	Status int    `json:"status"`
	Remark string `json:"remark"`
}

// RoleUpdateRes 修改响应。
type RoleUpdateRes struct{}

// RoleDeleteReq 删除。
type RoleDeleteReq struct {
	g.Meta `path:"/system/roles/{id}" tags:"SystemRole" method:"delete" summary:"删除角色"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// RoleDeleteRes 删除响应。
type RoleDeleteRes struct{}

// RoleAssignMenusReq 绑定菜单。
type RoleAssignMenusReq struct {
	g.Meta  `path:"/system/roles/{id}/menus" tags:"SystemRole" method:"put" summary:"角色分配菜单"`
	Id      uint64   `json:"id"      in:"path" v:"required"`
	MenuIds []uint64 `json:"menuIds"`
}

// RoleAssignMenusRes 响应。
type RoleAssignMenusRes struct{}

// RoleGetMenusReq 获取角色已分配的菜单 ID 列表。
type RoleGetMenusReq struct {
	g.Meta `path:"/system/roles/{id}/menus" tags:"SystemRole" method:"get" summary:"获取角色菜单"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// RoleGetMenusRes 响应。
type RoleGetMenusRes struct {
	MenuIds []uint64 `json:"menuIds"`
}

// 角色分配 API 权限
type RoleAssignApisReq struct {
	g.Meta `path:"/system/roles/{id}/apis" method:"put" tags:"角色管理" summary:"分配角色API权限"`
	Id     int64         `json:"id" in:"path" v:"required"`
	Apis   []RoleApiItem `json:"apis" dc:"API权限列表"`
}
type RoleApiItem struct {
	Path   string `json:"path" v:"required"`
	Method string `json:"method" v:"required"`
}
type RoleAssignApisRes struct{}

// 获取角色已分配的 API
type RoleGetApisReq struct {
	g.Meta `path:"/system/roles/{id}/apis" method:"get" tags:"角色管理" summary:"获取角色API权限"`
	Id     int64 `json:"id" in:"path" v:"required"`
}
type RoleGetApisRes struct {
	List []RoleApiItem `json:"list"`
}
