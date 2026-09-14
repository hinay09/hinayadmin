package model

import "github.com/gogf/gf/v2/os/gtime"

// SysRole 角色实体。
type SysRole struct {
	Id        uint64      `json:"id"        orm:"id"`
	Name      string      `json:"name"      orm:"name"`
	Code      string      `json:"code"      orm:"code"`
	Sort      int         `json:"sort"      orm:"sort"`
	Status    int         `json:"status"    orm:"status"`
	Remark    string      `json:"remark"    orm:"remark"`
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at"`
}
