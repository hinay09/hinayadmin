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
	DataScope int         `json:"dataScope" orm:"data_scope"`
	CreateId  uint64      `json:"createId" orm:"create_id"`
	UpdateId  uint64      `json:"updateId" orm:"update_id"`
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

// OrgScope 组织数据范围 (当前登录用户全部角色的并集)。
type OrgScope struct {
	All    bool     // 全部数据 (admin 或任一角色为全部)
	Self   bool     // 仅本人 (任一角色为仅本人)
	OrgIds []uint64 // 允许访问的组织 ID 集合 (自定义/本部门/及以下 的并集)
}
