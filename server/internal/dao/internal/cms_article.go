// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// CmsArticleDao is the data access object for the table cms_article.
type CmsArticleDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  CmsArticleColumns  // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// CmsArticleColumns defines and stores column names for the table cms_article.
type CmsArticleColumns struct {
	Id         string // 主键ID
	TenantId   string // 租户ID(ORM按上下文注入)
	CategoryId string // 栏目ID(0=未分类)
	Title      string // 文章标题
	Cover      string // 封面图持久引用(出口经storage.ViewURL重写)
	Summary    string // 摘要(列表展示)
	Content    string // 正文HTML(wangeditor; img src存持久引用)
	Tags       string // 标签(逗号分隔, 仅展示用)
	Status     string // 状态:0=草稿,1=已发布,2=已下架
	Top        string // 置顶:1=是,0=否
	PublishAt  string // 发布时间(首次发布写入)
	CreateId   string // 创建人ID(ORM自动填充)
	UpdateId   string // 最后修改人ID(ORM自动填充)
	CreatedAt  string // 创建时间
	UpdatedAt  string // 更新时间
	DeletedAt  string // 删除时间(软删)
}

// cmsArticleColumns holds the columns for the table cms_article.
var cmsArticleColumns = CmsArticleColumns{
	Id:         "id",
	TenantId:   "tenant_id",
	CategoryId: "category_id",
	Title:      "title",
	Cover:      "cover",
	Summary:    "summary",
	Content:    "content",
	Tags:       "tags",
	Status:     "status",
	Top:        "top",
	PublishAt:  "publish_at",
	CreateId:   "create_id",
	UpdateId:   "update_id",
	CreatedAt:  "created_at",
	UpdatedAt:  "updated_at",
	DeletedAt:  "deleted_at",
}

// NewCmsArticleDao creates and returns a new DAO object for table data access.
func NewCmsArticleDao(handlers ...gdb.ModelHandler) *CmsArticleDao {
	return &CmsArticleDao{
		group:    "default",
		table:    "cms_article",
		columns:  cmsArticleColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *CmsArticleDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *CmsArticleDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *CmsArticleDao) Columns() CmsArticleColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *CmsArticleDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *CmsArticleDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
// It rolls back the transaction and returns the error if function f returns a non-nil error.
// It commits the transaction and returns nil if function f returns nil.
//
// Note: Do not commit or roll back the transaction in function f,
// as it is automatically handled by this function.
func (dao *CmsArticleDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
