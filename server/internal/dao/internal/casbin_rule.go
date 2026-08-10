// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// CasbinRuleDao is the data access object for the table casbin_rule.
type CasbinRuleDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  CasbinRuleColumns  // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// CasbinRuleColumns defines and stores column names for the table casbin_rule.
type CasbinRuleColumns struct {
	Id    string // 主键
	Ptype string // 策略类型: p / g / g2 ...
	V0    string // p:sub(role) | g:user
	V1    string // p:obj(path) | g:role
	V2    string // p:act(method) | g:domain
	V3    string // 预留字段
	V4    string // 预留字段
	V5    string // 预留字段
}

// casbinRuleColumns holds the columns for the table casbin_rule.
var casbinRuleColumns = CasbinRuleColumns{
	Id:    "id",
	Ptype: "ptype",
	V0:    "v0",
	V1:    "v1",
	V2:    "v2",
	V3:    "v3",
	V4:    "v4",
	V5:    "v5",
}

// NewCasbinRuleDao creates and returns a new DAO object for table data access.
func NewCasbinRuleDao(handlers ...gdb.ModelHandler) *CasbinRuleDao {
	return &CasbinRuleDao{
		group:    "default",
		table:    "casbin_rule",
		columns:  casbinRuleColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *CasbinRuleDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *CasbinRuleDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *CasbinRuleDao) Columns() CasbinRuleColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *CasbinRuleDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *CasbinRuleDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *CasbinRuleDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
