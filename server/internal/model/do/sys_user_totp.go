// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysUserTotp is the golang structure of table sys_user_totp for DAO operations like Where/Data.
type SysUserTotp struct {
	g.Meta    `orm:"table:sys_user_totp, do:true"`
	Id        any         // 主键ID
	UserId    any         // 用户ID(唯一)
	Secret    any         // TOTP 密钥(Base32)
	Enabled   any         // 状态:0=待验证(已生成未绑定),1=已启用
	LastStep  any         // 最近已消费的时间步(Unix/30, 防验证码重放)
	CreatedAt *gtime.Time // 创建时间
	UpdatedAt *gtime.Time // 更新时间
	DeletedAt *gtime.Time // 删除时间(软删)
}
