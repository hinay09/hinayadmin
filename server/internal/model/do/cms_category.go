// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// CmsCategory is the golang structure of table cms_category for DAO operations like Where/Data.
type CmsCategory struct {
	g.Meta    `orm:"table:cms_category, do:true"`
	Id        any         // 主键ID
	TenantId  any         // 租户ID(ORM按上下文注入)
	Name      any         // 栏目名称
	Code      any         // 栏目编码(租户内唯一)
	Sort      any         // 排序(小在前)
	Status    any         // 状态:1=启用,0=禁用
	Remark    any         // 备注
	CreateId  any         // 创建人ID(ORM自动填充)
	UpdateId  any         // 最后修改人ID(ORM自动填充)
	CreatedAt *gtime.Time // 创建时间
	UpdatedAt *gtime.Time // 更新时间
	DeletedAt *gtime.Time // 删除时间(软删)
}
