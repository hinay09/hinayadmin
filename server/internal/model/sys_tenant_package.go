package model

import "github.com/gogf/gf/v2/os/gtime"

// SysTenantPackage 租户套餐 (应用授权集模板, 平台表, 管理入口仅平台超管)。
type SysTenantPackage struct {
	Id        uint64      `json:"id"        orm:"id"`
	Name      string      `json:"name"      orm:"name"`
	MenuIds   string      `json:"menuIds"   orm:"menu_ids"`
	Status    int         `json:"status"    orm:"status"`
	Remark    string      `json:"remark"    orm:"remark"`
	CreateId  uint64      `json:"createId"  orm:"create_id"`
	UpdateId  uint64      `json:"updateId"  orm:"update_id"`
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at"`
	DeletedAt *gtime.Time `json:"deletedAt" orm:"deleted_at"`
}
