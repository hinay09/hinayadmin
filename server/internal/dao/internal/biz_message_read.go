// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// BizMessageReadDao is the data access object for the table biz_message_read.
type BizMessageReadDao struct {
	table    string                // table is the underlying table name of the DAO.
	group    string                // group is the database configuration group name of the current DAO.
	columns  BizMessageReadColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler    // handlers for customized model modification.
}

// BizMessageReadColumns defines and stores column names for the table biz_message_read.
type BizMessageReadColumns struct {
	Id        string //
	MessageId string // 消息ID
	UserId    string // 用户ID
	ReadAt    string // 阅读时间
	Hidden    string // 是否在收件箱视角隐藏(个人删除):1=是
}

// bizMessageReadColumns holds the columns for the table biz_message_read.
var bizMessageReadColumns = BizMessageReadColumns{
	Id:        "id",
	MessageId: "message_id",
	UserId:    "user_id",
	ReadAt:    "read_at",
	Hidden:    "hidden",
}

// NewBizMessageReadDao creates and returns a new DAO object for table data access.
func NewBizMessageReadDao(handlers ...gdb.ModelHandler) *BizMessageReadDao {
	return &BizMessageReadDao{
		group:    "default",
		table:    "biz_message_read",
		columns:  bizMessageReadColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *BizMessageReadDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *BizMessageReadDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *BizMessageReadDao) Columns() BizMessageReadColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *BizMessageReadDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *BizMessageReadDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *BizMessageReadDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
