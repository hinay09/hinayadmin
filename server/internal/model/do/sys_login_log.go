// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysLoginLog is the golang structure of table sys_login_log for DAO operations like Where/Data.
type SysLoginLog struct {
	g.Meta    `orm:"table:sys_login_log, do:true"`
	Id        any         // ID
	UserId    any         // 用户ID(登录用户不存在时为0)
	Username  any         // 登录账号
	Status    any         // 结果:1=成功,0=失败
	Message   any         // 失败原因(成功时为空)
	Ip        any         // 登录IP
	UserAgent any         // User-Agent
	CreatedAt *gtime.Time // 创建时间
}
