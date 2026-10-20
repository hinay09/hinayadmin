// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysTenantPackage is the golang structure for table sys_tenant_package.
type SysTenantPackage struct {
	Id        uint64      `json:"id"        orm:"id"         description:""`                         //
	Name      string      `json:"name"      orm:"name"       description:"套餐名称(全局唯一, 软删时改写防撞uk)"`    // 套餐名称(全局唯一, 软删时改写防撞uk)
	MenuIds   string      `json:"menuIds"   orm:"menu_ids"   description:"应用授权集模板(目录/页面级ID的JSON数组)"` // 应用授权集模板(目录/页面级ID的JSON数组)
	Status    int         `json:"status"    orm:"status"     description:"1=启用 0=停用"`                // 1=启用 0=停用
	Remark    string      `json:"remark"    orm:"remark"     description:"备注"`                       // 备注
	CreateId  uint64      `json:"createId"  orm:"create_id"  description:""`                         //
	UpdateId  uint64      `json:"updateId"  orm:"update_id"  description:""`                         //
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:""`                         //
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at" description:""`                         //
	DeletedAt *gtime.Time `json:"deletedAt" orm:"deleted_at" description:""`                         //
}
