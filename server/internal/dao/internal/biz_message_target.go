// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// BizMessageTargetDao is the data access object for the table biz_message_target.
type BizMessageTargetDao struct {
	table    string                  // table is the underlying table name of the DAO.
	group    string                  // group is the database configuration group name of the current DAO.
	columns  BizMessageTargetColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler      // handlers for customized model modification.
}

// BizMessageTargetColumns defines and stores column names for the table biz_message_target.
type BizMessageTargetColumns struct {
	Id         string //
	MessageId  string // 消息ID
	TargetType string // 目标类型:2=role,3=user
	TargetId   string // 角色ID或用户ID
}

// bizMessageTargetColumns holds the columns for the table biz_message_target.
var bizMessageTargetColumns = BizMessageTargetColumns{
	Id:         "id",
	MessageId:  "message_id",
	TargetType: "target_type",
	TargetId:   "target_id",
}

// NewBizMessageTargetDao creates and returns a new DAO object for table data access.
func NewBizMessageTargetDao(handlers ...gdb.ModelHandler) *BizMessageTargetDao {
	return &BizMessageTargetDao{
		group:    "default",
		table:    "biz_message_target",
		columns:  bizMessageTargetColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *BizMessageTargetDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *BizMessageTargetDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *BizMessageTargetDao) Columns() BizMessageTargetColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *BizMessageTargetDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *BizMessageTargetDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *BizMessageTargetDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
