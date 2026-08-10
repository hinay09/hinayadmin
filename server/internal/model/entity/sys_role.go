// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysRole is the golang structure for table sys_role.
type SysRole struct {
	Id        uint64      `json:"id"        orm:"id"         description:"角色ID"`         // 角色ID
	Name      string      `json:"name"      orm:"name"       description:"角色名称"`         // 角色名称
	Code      string      `json:"code"      orm:"code"       description:"角色编码"`         // 角色编码
	Sort      int         `json:"sort"      orm:"sort"       description:"排序"`           // 排序
	Status    int         `json:"status"    orm:"status"     description:"状态:1=启用,0=禁用"` // 状态:1=启用,0=禁用
	Remark    string      `json:"remark"    orm:"remark"     description:"备注"`           // 备注
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:"创建时间"`         // 创建时间
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at" description:"更新时间"`         // 更新时间
	DeletedAt *gtime.Time `json:"deletedAt" orm:"deleted_at" description:"删除时间(软删)"`     // 删除时间(软删)
}
