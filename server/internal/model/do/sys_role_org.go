// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysRoleOrg is the golang structure of table sys_role_org for DAO operations like Where/Data.
type SysRoleOrg struct {
	g.Meta    `orm:"table:sys_role_org, do:true"`
	Id        any         // ID
	RoleId    any         // 角色ID
	OrgId     any         // 组织ID
	CreateId  any         // 创建人ID(ORM自动填充)
	UpdateId  any         // 最后修改人ID(ORM自动填充)
	CreatedAt *gtime.Time // 创建时间
}
