// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysTenantMenu is the golang structure for table sys_tenant_menu.
type SysTenantMenu struct {
	TenantId  uint64      `json:"tenantId"  orm:"tenant_id"  description:""` //
	MenuId    uint64      `json:"menuId"    orm:"menu_id"    description:""` //
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:""` //
}
