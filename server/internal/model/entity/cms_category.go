// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// CmsCategory is the golang structure for table cms_category.
type CmsCategory struct {
	Id        uint64      `json:"id"        orm:"id"         description:"主键ID"`             // 主键ID
	TenantId  uint64      `json:"tenantId"  orm:"tenant_id"  description:"租户ID(ORM按上下文注入)"`  // 租户ID(ORM按上下文注入)
	Name      string      `json:"name"      orm:"name"       description:"栏目名称"`             // 栏目名称
	Code      string      `json:"code"      orm:"code"       description:"栏目编码(租户内唯一)"`      // 栏目编码(租户内唯一)
	Sort      int         `json:"sort"      orm:"sort"       description:"排序(小在前)"`          // 排序(小在前)
	Status    int         `json:"status"    orm:"status"     description:"状态:1=启用,0=禁用"`     // 状态:1=启用,0=禁用
	Remark    string      `json:"remark"    orm:"remark"     description:"备注"`               // 备注
	CreateId  uint64      `json:"createId"  orm:"create_id"  description:"创建人ID(ORM自动填充)"`   // 创建人ID(ORM自动填充)
	UpdateId  uint64      `json:"updateId"  orm:"update_id"  description:"最后修改人ID(ORM自动填充)"` // 最后修改人ID(ORM自动填充)
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:"创建时间"`             // 创建时间
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at" description:"更新时间"`             // 更新时间
	DeletedAt *gtime.Time `json:"deletedAt" orm:"deleted_at" description:"删除时间(软删)"`         // 删除时间(软删)
}
