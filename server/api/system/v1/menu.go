// Package v1 系统管理-菜单接口。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/internal/model"
)

// MenuListReq 菜单列表(扁平,前端自行构建树或使用 tree 接口)。
type MenuListReq struct {
	g.Meta  `path:"/system/menus" tags:"SystemMenu" method:"get" summary:"菜单列表(扁平)"`
	Keyword string `json:"keyword" in:"query"`
	Status  *int   `json:"status"  in:"query"`
}

// MenuListRes 菜单列表。
type MenuListRes struct {
	List []*model.SysMenu `json:"list"`
}

// MenuTreeReq 树形菜单。
type MenuTreeReq struct {
	g.Meta `path:"/system/menus/tree" tags:"SystemMenu" method:"get" summary:"树形菜单"`
}

// MenuTreeRes 树形响应。
type MenuTreeRes struct {
	Tree []*model.MenuTree `json:"tree"`
}

// MenuDetailReq 详情。
type MenuDetailReq struct {
	g.Meta `path:"/system/menus/{id}" tags:"SystemMenu" method:"get" summary:"菜单详情"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// MenuDetailRes 详情。
type MenuDetailRes struct {
	*model.SysMenu
}

// MenuCreateReq 新增。
type MenuCreateReq struct {
	g.Meta     `path:"/system/menus" tags:"SystemMenu" method:"post" summary:"新增菜单"`
	ParentId   uint64 `json:"parentId"`
	Name       string `json:"name"       v:"required|length:1,32"`
	Type       int    `json:"type"       v:"required|in:1,2,3"`
	Path       string `json:"path"`
	Component  string `json:"component"`
	Icon       string `json:"icon"`
	Permission string `json:"permission"`
	ApiPath    string `json:"apiPath"`
	ApiMethod  string `json:"apiMethod"`
	Sort       int    `json:"sort"`
	Visible    int    `json:"visible"    d:"1"`
	Status     int    `json:"status"     d:"1"`
}

// MenuCreateRes 响应。
type MenuCreateRes struct {
	Id uint64 `json:"id"`
}

// MenuUpdateReq 修改。
type MenuUpdateReq struct {
	g.Meta     `path:"/system/menus/{id}" tags:"SystemMenu" method:"put" summary:"修改菜单"`
	Id         uint64 `json:"id" in:"path" v:"required"`
	ParentId   uint64 `json:"parentId"`
	Name       string `json:"name"       v:"required"`
	Type       int    `json:"type"       v:"required|in:1,2,3"`
	Path       string `json:"path"`
	Component  string `json:"component"`
	Icon       string `json:"icon"`
	Permission string `json:"permission"`
	ApiPath    string `json:"apiPath"`
	ApiMethod  string `json:"apiMethod"`
	Sort       int    `json:"sort"`
	Visible    int    `json:"visible"`
	Status     int    `json:"status"`
}

// MenuUpdateRes 响应。
type MenuUpdateRes struct{}

// MenuDeleteReq 删除。
type MenuDeleteReq struct {
	g.Meta `path:"/system/menus/{id}" tags:"SystemMenu" method:"delete" summary:"删除菜单"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// MenuDeleteRes 响应。
type MenuDeleteRes struct{}
