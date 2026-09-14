package model

import "github.com/gogf/gf/v2/os/gtime"

// SysMenu 菜单实体。
type SysMenu struct {
	Id         uint64      `json:"id"         orm:"id"`
	ParentId   uint64      `json:"parentId"   orm:"parent_id"`
	Name       string      `json:"name"       orm:"name"`
	Type       int         `json:"type"       orm:"type"`
	Path       string      `json:"path"       orm:"path"`
	Component  string      `json:"component"  orm:"component"`
	Icon       string      `json:"icon"       orm:"icon"`
	Permission string      `json:"permission" orm:"permission"`
	Sort       int         `json:"sort"       orm:"sort"`
	Visible    int         `json:"visible"    orm:"visible"`
	Status     int         `json:"status"     orm:"status"`
	CreatedAt  *gtime.Time `json:"createdAt"  orm:"created_at"`
	UpdatedAt  *gtime.Time `json:"updatedAt"  orm:"updated_at"`
}
