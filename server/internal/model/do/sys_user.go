// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysUser is the golang structure of table sys_user for DAO operations like Where/Data.
type SysUser struct {
	g.Meta    `orm:"table:sys_user, do:true"`
	Id        any         // 用户ID
	Username  any         // 登录账号
	Password  any         // bcrypt 加密密码
	Nickname  any         // 昵称
	Avatar    any         // 头像URL
	Email     any         // 邮箱
	Phone     any         // 手机号
	OrgId     any         // 所属组织ID
	Status    any         // 状态:1=启用,0=禁用
	Remark    any         // 备注
	CreatedAt *gtime.Time // 创建时间
	UpdatedAt *gtime.Time // 更新时间
	DeletedAt *gtime.Time // 删除时间(软删)
}
