// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// CmsArticle is the golang structure for table cms_article.
type CmsArticle struct {
	Id         uint64      `json:"id"         orm:"id"          description:"主键ID"`                             // 主键ID
	TenantId   uint64      `json:"tenantId"   orm:"tenant_id"   description:"租户ID(ORM按上下文注入)"`                  // 租户ID(ORM按上下文注入)
	CategoryId uint64      `json:"categoryId" orm:"category_id" description:"栏目ID(0=未分类)"`                      // 栏目ID(0=未分类)
	Title      string      `json:"title"      orm:"title"       description:"文章标题"`                             // 文章标题
	Cover      string      `json:"cover"      orm:"cover"       description:"封面图持久引用(出口经storage.ViewURL重写)"`    // 封面图持久引用(出口经storage.ViewURL重写)
	Summary    string      `json:"summary"    orm:"summary"     description:"摘要(列表展示)"`                         // 摘要(列表展示)
	Content    string      `json:"content"    orm:"content"     description:"正文HTML(wangeditor; img src存持久引用)"` // 正文HTML(wangeditor; img src存持久引用)
	Tags       string      `json:"tags"       orm:"tags"        description:"标签(逗号分隔, 仅展示用)"`                   // 标签(逗号分隔, 仅展示用)
	Status     int         `json:"status"     orm:"status"      description:"状态:0=草稿,1=已发布,2=已下架"`              // 状态:0=草稿,1=已发布,2=已下架
	Top        int         `json:"top"        orm:"top"         description:"置顶:1=是,0=否"`                       // 置顶:1=是,0=否
	PublishAt  *gtime.Time `json:"publishAt"  orm:"publish_at"  description:"发布时间(首次发布写入)"`                     // 发布时间(首次发布写入)
	CreateId   uint64      `json:"createId"   orm:"create_id"   description:"创建人ID(ORM自动填充)"`                   // 创建人ID(ORM自动填充)
	UpdateId   uint64      `json:"updateId"   orm:"update_id"   description:"最后修改人ID(ORM自动填充)"`                 // 最后修改人ID(ORM自动填充)
	CreatedAt  *gtime.Time `json:"createdAt"  orm:"created_at"  description:"创建时间"`                             // 创建时间
	UpdatedAt  *gtime.Time `json:"updatedAt"  orm:"updated_at"  description:"更新时间"`                             // 更新时间
	DeletedAt  *gtime.Time `json:"deletedAt"  orm:"deleted_at"  description:"删除时间(软删)"`                         // 删除时间(软删)
}
