// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysAuditLogDao is the data access object for the table sys_audit_log.
type SysAuditLogDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  SysAuditLogColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// SysAuditLogColumns defines and stores column names for the table sys_audit_log.
type SysAuditLogColumns struct {
	Id         string // ID
	UserId     string // 用户ID
	Username   string // 用户名
	Action     string // 操作类型(create/update/delete/upload/...)
	Resource   string // 操作资源(如user/role/menu/dict/file)
	ResourceId string // 资源标识
	Detail     string // 详情(JSON格式)
	Ip         string // IP地址
	UserAgent  string // User-Agent
	CreatedAt  string // 创建时间
}

// sysAuditLogColumns holds the columns for the table sys_audit_log.
var sysAuditLogColumns = SysAuditLogColumns{
	Id:         "id",
	UserId:     "user_id",
	Username:   "username",
	Action:     "action",
	Resource:   "resource",
	ResourceId: "resource_id",
	Detail:     "detail",
	Ip:         "ip",
	UserAgent:  "user_agent",
	CreatedAt:  "created_at",
}

// NewSysAuditLogDao creates and returns a new DAO object for table data access.
func NewSysAuditLogDao(handlers ...gdb.ModelHandler) *SysAuditLogDao {
	return &SysAuditLogDao{
		group:    "default",
		table:    "sys_audit_log",
		columns:  sysAuditLogColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysAuditLogDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SysAuditLogDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SysAuditLogDao) Columns() SysAuditLogColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SysAuditLogDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SysAuditLogDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *SysAuditLogDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
