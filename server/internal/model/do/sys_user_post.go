// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysUserPost is the golang structure of table sys_user_post for DAO operations like Where/Data.
type SysUserPost struct {
	g.Meta    `orm:"table:sys_user_post, do:true"`
	Id        any         // 主键ID
	UserId    any         // 用户ID
	PostId    any         // 岗位ID
	OrgId     any         // 组织ID(0=不限定组织)
	CreatedAt *gtime.Time // 创建时间
	UpdatedAt *gtime.Time // 更新时间

}
