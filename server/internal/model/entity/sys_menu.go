// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysMenu is the golang structure for table sys_menu.
type SysMenu struct {
	Id         uint64      `json:"id"         orm:"id"         description:"菜单ID"`              // 菜单ID
	ParentId   uint64      `json:"parentId"   orm:"parent_id"  description:"父级ID"`              // 父级ID
	Name       string      `json:"name"       orm:"name"       description:"菜单名称"`              // 菜单名称
	Type       int         `json:"type"       orm:"type"       description:"类型:1=目录,2=菜单,3=按钮"` // 类型:1=目录,2=菜单,3=按钮
	Path       string      `json:"path"       orm:"path"       description:"路由路径"`              // 路由路径
	Component  string      `json:"component"  orm:"component"  description:"组件路径"`              // 组件路径
	Icon       string      `json:"icon"       orm:"icon"       description:"图标"`                // 图标
	Permission string      `json:"permission" orm:"permission" description:"权限标识"`              // 权限标识
	Sort       int         `json:"sort"       orm:"sort"       description:"排序"`                // 排序
	Visible    int         `json:"visible"    orm:"visible"    description:"是否显示:1=是,0=否"`      // 是否显示:1=是,0=否
	Status     int         `json:"status"     orm:"status"     description:"状态:1=启用,0=禁用"`      // 状态:1=启用,0=禁用
	CreatedAt  *gtime.Time `json:"createdAt"  orm:"created_at" description:"创建时间"`              // 创建时间
	UpdatedAt  *gtime.Time `json:"updatedAt"  orm:"updated_at" description:"更新时间"`              // 更新时间
	DeletedAt  *gtime.Time `json:"deletedAt"  orm:"deleted_at" description:"删除时间(软删)"`          // 删除时间(软删)
}
