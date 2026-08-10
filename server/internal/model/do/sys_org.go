// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysOrg is the golang structure of table sys_org for DAO operations like Where/Data.
type SysOrg struct {
	g.Meta    `orm:"table:sys_org, do:true"`
	Id        any         // 组织ID
	ParentId  any         // 父级ID, 0=顶级
	Name      any         // 组织名称
	Leader    any         // 负责人
	Phone     any         // 联系电话
	Email     any         // 邮箱
	Sort      any         // 排序
	Status    any         // 状态:1=启用,0=禁用
	Remark    any         // 备注
	CreatedAt *gtime.Time // 创建时间
	UpdatedAt *gtime.Time // 更新时间
	DeletedAt *gtime.Time // 删除时间(软删)
}
