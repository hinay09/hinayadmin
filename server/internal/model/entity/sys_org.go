// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysOrg is the golang structure for table sys_org.
type SysOrg struct {
	Id        uint64      `json:"id"        orm:"id"         description:"组织ID"`         // 组织ID
	ParentId  uint64      `json:"parentId"  orm:"parent_id"  description:"父级ID, 0=顶级"`   // 父级ID, 0=顶级
	Name      string      `json:"name"      orm:"name"       description:"组织名称"`         // 组织名称
	Leader    string      `json:"leader"    orm:"leader"     description:"负责人"`          // 负责人
	Phone     string      `json:"phone"     orm:"phone"      description:"联系电话"`         // 联系电话
	Email     string      `json:"email"     orm:"email"      description:"邮箱"`           // 邮箱
	Sort      int         `json:"sort"      orm:"sort"       description:"排序"`           // 排序
	Status    int         `json:"status"    orm:"status"     description:"状态:1=启用,0=禁用"` // 状态:1=启用,0=禁用
	Remark    string      `json:"remark"    orm:"remark"     description:"备注"`           // 备注
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:"创建时间"`         // 创建时间
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at" description:"更新时间"`         // 更新时间
	DeletedAt *gtime.Time `json:"deletedAt" orm:"deleted_at" description:"删除时间(软删)"`     // 删除时间(软删)
}
