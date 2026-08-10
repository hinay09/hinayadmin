// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysDictType is the golang structure for table sys_dict_type.
type SysDictType struct {
	Id        uint64      `json:"id"        orm:"id"         description:"ID"`           // ID
	TypeCode  string      `json:"typeCode"  orm:"type_code"  description:"字典类型编码(唯一)"`   // 字典类型编码(唯一)
	TypeName  string      `json:"typeName"  orm:"type_name"  description:"字典类型名称"`       // 字典类型名称
	Status    int         `json:"status"    orm:"status"     description:"状态:1=启用,0=禁用"` // 状态:1=启用,0=禁用
	Remark    string      `json:"remark"    orm:"remark"     description:"备注"`           // 备注
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:"创建时间"`         // 创建时间
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at" description:"更新时间"`         // 更新时间
	DeletedAt *gtime.Time `json:"deletedAt" orm:"deleted_at" description:"删除时间(软删)"`     // 删除时间(软删)
}
