// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysRoleOrgDao is the data access object for the table sys_role_org.
type SysRoleOrgDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  SysRoleOrgColumns  // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// SysRoleOrgColumns defines and stores column names for the table sys_role_org.
type SysRoleOrgColumns struct {
	Id        string // ID
	RoleId    string // 角色ID
	OrgId     string // 组织ID
	CreateId  string // 创建人ID(ORM自动填充)
	UpdateId  string // 最后修改人ID(ORM自动填充)
	CreatedAt string // 创建时间
}

// sysRoleOrgColumns holds the columns for the table sys_role_org.
var sysRoleOrgColumns = SysRoleOrgColumns{
	Id:        "id",
	RoleId:    "role_id",
	OrgId:     "org_id",
	CreateId:  "create_id",
	UpdateId:  "update_id",
	CreatedAt: "created_at",
}

// NewSysRoleOrgDao creates and returns a new DAO object for table data access.
func NewSysRoleOrgDao(handlers ...gdb.ModelHandler) *SysRoleOrgDao {
	return &SysRoleOrgDao{
		group:    "default",
		table:    "sys_role_org",
		columns:  sysRoleOrgColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysRoleOrgDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SysRoleOrgDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SysRoleOrgDao) Columns() SysRoleOrgColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SysRoleOrgDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SysRoleOrgDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *SysRoleOrgDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
