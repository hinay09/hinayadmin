// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysTenantMenu is the golang structure of table sys_tenant_menu for DAO operations like Where/Data.
type SysTenantMenu struct {
	g.Meta    `orm:"table:sys_tenant_menu, do:true"`
	TenantId  any         //
	MenuId    any         //
	CreatedAt *gtime.Time //
}
