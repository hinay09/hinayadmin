package model

import "github.com/gogf/gf/v2/os/gtime"

// SysUser 数据库实体。
type SysUser struct {
	Id        uint64      `json:"id"        orm:"id"`
	Username  string      `json:"username"  orm:"username"`
	Password  string      `json:"-"         orm:"password"`
	Nickname  string      `json:"nickname"  orm:"nickname"`
	Avatar    string      `json:"avatar"    orm:"avatar"`
	Email     string      `json:"email"     orm:"email"`
	Phone     string      `json:"phone"     orm:"phone"`
	OrgId     uint64      `json:"orgId"     orm:"org_id"`
	Status    int         `json:"status"    orm:"status"`
	Remark    string      `json:"remark"    orm:"remark"`
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at"`
}
