// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysLoginLogDao is the data access object for the table sys_login_log.
type SysLoginLogDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  SysLoginLogColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// SysLoginLogColumns defines and stores column names for the table sys_login_log.
type SysLoginLogColumns struct {
	Id        string // ID
	UserId    string // 用户ID(登录用户不存在时为0)
	Username  string // 登录账号
	Status    string // 结果:1=成功,0=失败
	Message   string // 失败原因(成功时为空)
	Ip        string // 登录IP
	UserAgent string // User-Agent
	CreatedAt string // 创建时间
}

// sysLoginLogColumns holds the columns for the table sys_login_log.
var sysLoginLogColumns = SysLoginLogColumns{
	Id:        "id",
	UserId:    "user_id",
	Username:  "username",
	Status:    "status",
	Message:   "message",
	Ip:        "ip",
	UserAgent: "user_agent",
	CreatedAt: "created_at",
}

// NewSysLoginLogDao creates and returns a new DAO object for table data access.
func NewSysLoginLogDao(handlers ...gdb.ModelHandler) *SysLoginLogDao {
	return &SysLoginLogDao{
		group:    "default",
		table:    "sys_login_log",
		columns:  sysLoginLogColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysLoginLogDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SysLoginLogDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SysLoginLogDao) Columns() SysLoginLogColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SysLoginLogDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SysLoginLogDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *SysLoginLogDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
