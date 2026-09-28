// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysLoginLog is the golang structure for table sys_login_log.
type SysLoginLog struct {
	Id        uint64      `json:"id"        orm:"id"         description:"ID"`               // ID
	UserId    uint64      `json:"userId"    orm:"user_id"    description:"用户ID(登录用户不存在时为0)"` // 用户ID(登录用户不存在时为0)
	Username  string      `json:"username"  orm:"username"   description:"登录账号"`             // 登录账号
	Status    int         `json:"status"    orm:"status"     description:"结果:1=成功,0=失败"`     // 结果:1=成功,0=失败
	Message   string      `json:"message"   orm:"message"    description:"失败原因(成功时为空)"`      // 失败原因(成功时为空)
	Ip        string      `json:"ip"        orm:"ip"         description:"登录IP"`             // 登录IP
	UserAgent string      `json:"userAgent" orm:"user_agent" description:"User-Agent"`       // User-Agent
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:"创建时间"`             // 创建时间
}
