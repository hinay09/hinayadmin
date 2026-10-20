// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysTenant is the golang structure for table sys_tenant.
type SysTenant struct {
	Id           uint64      `json:"id"           orm:"id"            description:""`                                    //
	Name         string      `json:"name"         orm:"name"          description:"租户名称"`                                // 租户名称
	Code         string      `json:"code"         orm:"code"          description:"租户标识(子域名/登录码, 全局唯一, 创建后不可变)"`         // 租户标识(子域名/登录码, 全局唯一, 创建后不可变)
	Contact      string      `json:"contact"      orm:"contact"       description:"联系人"`                                 // 联系人
	Plan         string      `json:"plan"         orm:"plan"          description:"套餐(展示标签, 预留)"`                        // 套餐(展示标签, 预留)
	PackageId    uint64      `json:"packageId"    orm:"package_id"    description:"套餐ID(sys_tenant_package, 0=自定义/未绑定)"` // 套餐ID(sys_tenant_package, 0=自定义/未绑定)
	QuotaUsers   int         `json:"quotaUsers"   orm:"quota_users"   description:"用户数配额(0=不限, 预留)"`                     // 用户数配额(0=不限, 预留)
	QuotaStorage int64       `json:"quotaStorage" orm:"quota_storage" description:"存储配额字节(0=不限, 预留)"`                    // 存储配额字节(0=不限, 预留)
	ExpireAt     *gtime.Time `json:"expireAt"     orm:"expire_at"     description:"到期时间(NULL=永久)"`                       // 到期时间(NULL=永久)
	Status       int         `json:"status"       orm:"status"        description:"1=启用 0=禁用"`                           // 1=启用 0=禁用
	Remark       string      `json:"remark"       orm:"remark"        description:"备注"`                                  // 备注
	CreateId     uint64      `json:"createId"     orm:"create_id"     description:""`                                    //
	UpdateId     uint64      `json:"updateId"     orm:"update_id"     description:""`                                    //
	CreatedAt    *gtime.Time `json:"createdAt"    orm:"created_at"    description:""`                                    //
	UpdatedAt    *gtime.Time `json:"updatedAt"    orm:"updated_at"    description:""`                                    //
	DeletedAt    *gtime.Time `json:"deletedAt"    orm:"deleted_at"    description:""`                                    //
}
