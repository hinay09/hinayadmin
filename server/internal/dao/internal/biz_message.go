// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// BizMessageDao is the data access object for the table biz_message.
type BizMessageDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  BizMessageColumns  // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// BizMessageColumns defines and stores column names for the table biz_message.
type BizMessageColumns struct {
	Id          string //
	Type        string // 消息类型:1=系统通知,2=私信
	Title       string // 标题
	Content     string // 内容
	Level       string // 级别:1=普通,2=重要,3=紧急
	SenderId    string // 发送人ID
	TargetScope string // 系统通知范围:1=all,2=role,3=user;私信=0
	ReceiverId  string // 私信接收者ID
	Status      string // 状态:1=已发布,0=草稿
	CreatedAt   string // 创建时间
	UpdatedAt   string // 更新时间
	DeletedAt   string // 删除时间(软删)
}

// bizMessageColumns holds the columns for the table biz_message.
var bizMessageColumns = BizMessageColumns{
	Id:          "id",
	Type:        "type",
	Title:       "title",
	Content:     "content",
	Level:       "level",
	SenderId:    "sender_id",
	TargetScope: "target_scope",
	ReceiverId:  "receiver_id",
	Status:      "status",
	CreatedAt:   "created_at",
	UpdatedAt:   "updated_at",
	DeletedAt:   "deleted_at",
}

// NewBizMessageDao creates and returns a new DAO object for table data access.
func NewBizMessageDao(handlers ...gdb.ModelHandler) *BizMessageDao {
	return &BizMessageDao{
		group:    "default",
		table:    "biz_message",
		columns:  bizMessageColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *BizMessageDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *BizMessageDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *BizMessageDao) Columns() BizMessageColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *BizMessageDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *BizMessageDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *BizMessageDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
