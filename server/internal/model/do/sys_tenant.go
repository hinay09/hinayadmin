// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysTenant is the golang structure of table sys_tenant for DAO operations like Where/Data.
type SysTenant struct {
	g.Meta       `orm:"table:sys_tenant, do:true"`
	Id           any         //
	Name         any         // 租户名称
	Code         any         // 租户标识(子域名/登录码, 全局唯一, 创建后不可变)
	Contact      any         // 联系人
	Plan         any         // 套餐(展示标签, 预留)
	QuotaUsers   any         // 用户数配额(0=不限, 预留)
	QuotaStorage any         // 存储配额字节(0=不限, 预留)
	ExpireAt     *gtime.Time // 到期时间(NULL=永久)
	Status       any         // 1=启用 0=禁用
	Remark       any         // 备注
	CreateId     any         //
	UpdateId     any         //
	CreatedAt    *gtime.Time //
	UpdatedAt    *gtime.Time //
	DeletedAt    *gtime.Time //
}
