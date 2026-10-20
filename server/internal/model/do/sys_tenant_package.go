// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysTenantPackage is the golang structure of table sys_tenant_package for DAO operations like Where/Data.
type SysTenantPackage struct {
	g.Meta    `orm:"table:sys_tenant_package, do:true"`
	Id        any         //
	Name      any         // 套餐名称(全局唯一, 软删时改写防撞uk)
	MenuIds   any         // 应用授权集模板(目录/页面级ID的JSON数组)
	Status    any         // 1=启用 0=停用
	Remark    any         // 备注
	CreateId  any         //
	UpdateId  any         //
	CreatedAt *gtime.Time //
	UpdatedAt *gtime.Time //
	DeletedAt *gtime.Time //
}
