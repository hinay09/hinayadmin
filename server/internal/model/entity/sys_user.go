// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysUser is the golang structure for table sys_user.
type SysUser struct {
	Id        uint64      `json:"id"        orm:"id"         description:"用户ID"`         // 用户ID
	Username  string      `json:"username"  orm:"username"   description:"登录账号"`         // 登录账号
	Password  string      `json:"password"  orm:"password"   description:"bcrypt 加密密码"`  // bcrypt 加密密码
	Nickname  string      `json:"nickname"  orm:"nickname"   description:"昵称"`           // 昵称
	Avatar    string      `json:"avatar"    orm:"avatar"     description:"头像URL"`        // 头像URL
	Email     string      `json:"email"     orm:"email"      description:"邮箱"`           // 邮箱
	Phone     string      `json:"phone"     orm:"phone"      description:"手机号"`          // 手机号
	OrgId     uint64      `json:"orgId"     orm:"org_id"     description:"所属组织ID"`       // 所属组织ID
	Status    int         `json:"status"    orm:"status"     description:"状态:1=启用,0=禁用"` // 状态:1=启用,0=禁用
	Remark    string      `json:"remark"    orm:"remark"     description:"备注"`           // 备注
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:"创建时间"`         // 创建时间
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at" description:"更新时间"`         // 更新时间
	DeletedAt *gtime.Time `json:"deletedAt" orm:"deleted_at" description:"删除时间(软删)"`     // 删除时间(软删)
}
