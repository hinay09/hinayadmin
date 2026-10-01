// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysUserTotpDao is the data access object for the table sys_user_totp.
type SysUserTotpDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  SysUserTotpColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// SysUserTotpColumns defines and stores column names for the table sys_user_totp.
type SysUserTotpColumns struct {
	Id        string // 主键ID
	UserId    string // 用户ID(唯一)
	Secret    string // TOTP 密钥(Base32)
	Enabled   string // 状态:0=待验证(已生成未绑定),1=已启用
	LastStep  string // 最近已消费的时间步(Unix/30, 防验证码重放)
	CreatedAt string // 创建时间
	UpdatedAt string // 更新时间
	DeletedAt string // 删除时间(软删)
}

// sysUserTotpColumns holds the columns for the table sys_user_totp.
var sysUserTotpColumns = SysUserTotpColumns{
	Id:        "id",
	UserId:    "user_id",
	Secret:    "secret",
	Enabled:   "enabled",
	LastStep:  "last_step",
	CreatedAt: "created_at",
	UpdatedAt: "updated_at",
	DeletedAt: "deleted_at",
}

// NewSysUserTotpDao creates and returns a new DAO object for table data access.
func NewSysUserTotpDao(handlers ...gdb.ModelHandler) *SysUserTotpDao {
	return &SysUserTotpDao{
		group:    "default",
		table:    "sys_user_totp",
		columns:  sysUserTotpColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysUserTotpDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SysUserTotpDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SysUserTotpDao) Columns() SysUserTotpColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SysUserTotpDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SysUserTotpDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *SysUserTotpDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
