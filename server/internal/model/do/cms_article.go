// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// CmsArticle is the golang structure of table cms_article for DAO operations like Where/Data.
type CmsArticle struct {
	g.Meta     `orm:"table:cms_article, do:true"`
	Id         any         // 主键ID
	TenantId   any         // 租户ID(ORM按上下文注入)
	CategoryId any         // 栏目ID(0=未分类)
	Title      any         // 文章标题
	Cover      any         // 封面图持久引用(出口经storage.ViewURL重写)
	Summary    any         // 摘要(列表展示)
	Content    any         // 正文HTML(wangeditor; img src存持久引用)
	Tags       any         // 标签(逗号分隔, 仅展示用)
	Status     any         // 状态:0=草稿,1=已发布,2=已下架
	Top        any         // 置顶:1=是,0=否
	PublishAt  *gtime.Time // 发布时间(首次发布写入)
	CreateId   any         // 创建人ID(ORM自动填充)
	UpdateId   any         // 最后修改人ID(ORM自动填充)
	CreatedAt  *gtime.Time // 创建时间
	UpdatedAt  *gtime.Time // 更新时间
	DeletedAt  *gtime.Time // 删除时间(软删)
}
