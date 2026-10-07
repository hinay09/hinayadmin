// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysPost is the golang structure of table sys_post for DAO operations like Where/Data.
type SysPost struct {
	g.Meta    `orm:"table:sys_post, do:true"`
	Id        any         // 主键ID
	PostCode  any         // 岗位编码(唯一, 如 hr, dept_leader)
	PostName  any         // 岗位名称
	PostKind  any         // 岗位类型:1=普通岗,2=主管岗(部门主管解析依据)
	Sort      any         // 排序
	Status    any         // 状态:1=启用,0=禁用
	Remark    any         // 备注
	CreateId  any         // 创建人ID(ORM自动填充)
	UpdateId  any         // 最后修改人ID(ORM自动填充)
	CreatedAt *gtime.Time // 创建时间
	UpdatedAt *gtime.Time // 更新时间
	DeletedAt *gtime.Time // 删除时间(软删)

}
