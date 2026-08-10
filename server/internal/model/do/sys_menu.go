// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysMenu is the golang structure of table sys_menu for DAO operations like Where/Data.
type SysMenu struct {
	g.Meta     `orm:"table:sys_menu, do:true"`
	Id         any         // 菜单ID
	ParentId   any         // 父级ID
	Name       any         // 菜单名称
	Type       any         // 类型:1=目录,2=菜单,3=按钮
	Path       any         // 路由路径
	Component  any         // 组件路径
	Icon       any         // 图标
	Permission any         // 权限标识
	Sort       any         // 排序
	Visible    any         // 是否显示:1=是,0=否
	Status     any         // 状态:1=启用,0=禁用
	CreatedAt  *gtime.Time // 创建时间
	UpdatedAt  *gtime.Time // 更新时间
	DeletedAt  *gtime.Time // 删除时间(软删)
}
