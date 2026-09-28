package model

import "github.com/gogf/gf/v2/os/gtime"

// SysOrg 组织机构实体。
type SysOrg struct {
	Id        uint64      `json:"id"        orm:"id"`
	ParentId  uint64      `json:"parentId"  orm:"parent_id"`
	Name      string      `json:"name"      orm:"name"`
	Leader    string      `json:"leader"    orm:"leader"`
	Phone     string      `json:"phone"     orm:"phone"`
	Email     string      `json:"email"     orm:"email"`
	Sort      int         `json:"sort"      orm:"sort"`
	Status    int         `json:"status"    orm:"status"`
	Remark    string      `json:"remark"    orm:"remark"`
	CreateId  uint64      `json:"createId" orm:"create_id"`
	UpdateId  uint64      `json:"updateId" orm:"update_id"`
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at"`
}
