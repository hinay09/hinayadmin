// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysUserTotp is the golang structure for table sys_user_totp.
type SysUserTotp struct {
	Id        uint64      `json:"id"        orm:"id"         description:"主键ID"`                       // 主键ID
	UserId    uint64      `json:"userId"    orm:"user_id"    description:"用户ID(唯一)"`                   // 用户ID(唯一)
	Secret    string      `json:"secret"    orm:"secret"     description:"TOTP 密钥(Base32)"`            // TOTP 密钥(Base32)
	Enabled   int         `json:"enabled"   orm:"enabled"    description:"状态:0=待验证(已生成未绑定),1=已启用"`     // 状态:0=待验证(已生成未绑定),1=已启用
	LastStep  int64       `json:"lastStep"  orm:"last_step"  description:"最近已消费的时间步(Unix/30, 防验证码重放)"` // 最近已消费的时间步(Unix/30, 防验证码重放)
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:"创建时间"`                       // 创建时间
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at" description:"更新时间"`                       // 更新时间
	DeletedAt *gtime.Time `json:"deletedAt" orm:"deleted_at" description:"删除时间(软删)"`                   // 删除时间(软删)
}
