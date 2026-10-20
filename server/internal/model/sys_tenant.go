package model

import "github.com/gogf/gf/v2/os/gtime"

// SysTenant 租户实体 (平台表, 管理入口仅平台超管)。
type SysTenant struct {
	Id           uint64      `json:"id"           orm:"id"`
	Name         string      `json:"name"         orm:"name"`
	Code         string      `json:"code"         orm:"code"`
	Contact      string      `json:"contact"      orm:"contact"`
	Plan         string      `json:"plan"         orm:"plan"`
	PackageId    uint64      `json:"packageId"    orm:"package_id"`
	QuotaUsers   int         `json:"quotaUsers"   orm:"quota_users"`
	QuotaStorage int64       `json:"quotaStorage" orm:"quota_storage"`
	ExpireAt     *gtime.Time `json:"expireAt"     orm:"expire_at"`
	Status       int         `json:"status"       orm:"status"`
	Remark       string      `json:"remark"       orm:"remark"`
	CreateId     uint64      `json:"createId"     orm:"create_id"`
	UpdateId     uint64      `json:"updateId"     orm:"update_id"`
	CreatedAt    *gtime.Time `json:"createdAt"    orm:"created_at"`
	UpdatedAt    *gtime.Time `json:"updatedAt"    orm:"updated_at"`
	DeletedAt    *gtime.Time `json:"deletedAt"    orm:"deleted_at"`
}
