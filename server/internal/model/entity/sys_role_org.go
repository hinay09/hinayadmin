// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysRoleOrg is the golang structure for table sys_role_org.
type SysRoleOrg struct {
	Id        uint64      `json:"id"        orm:"id"         description:"ID"`               // ID
	RoleId    uint64      `json:"roleId"    orm:"role_id"    description:"角色ID"`             // 角色ID
	OrgId     uint64      `json:"orgId"     orm:"org_id"     description:"组织ID"`             // 组织ID
	CreateId  uint64      `json:"createId"  orm:"create_id"  description:"创建人ID(ORM自动填充)"`   // 创建人ID(ORM自动填充)
	UpdateId  uint64      `json:"updateId"  orm:"update_id"  description:"最后修改人ID(ORM自动填充)"` // 最后修改人ID(ORM自动填充)
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:"创建时间"`             // 创建时间
}
