// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysTenantPackageDao is the data access object for the table sys_tenant_package.
type SysTenantPackageDao struct {
	table    string                  // table is the underlying table name of the DAO.
	group    string                  // group is the database configuration group name of the current DAO.
	columns  SysTenantPackageColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler      // handlers for customized model modification.
}

// SysTenantPackageColumns defines and stores column names for the table sys_tenant_package.
type SysTenantPackageColumns struct {
	Id        string //
	Name      string // 套餐名称(全局唯一, 软删时改写防撞uk)
	MenuIds   string // 应用授权集模板(目录/页面级ID的JSON数组)
	Status    string // 1=启用 0=停用
	Remark    string // 备注
	CreateId  string //
	UpdateId  string //
	CreatedAt string //
	UpdatedAt string //
	DeletedAt string //
}

// sysTenantPackageColumns holds the columns for the table sys_tenant_package.
var sysTenantPackageColumns = SysTenantPackageColumns{
	Id:        "id",
	Name:      "name",
	MenuIds:   "menu_ids",
	Status:    "status",
	Remark:    "remark",
	CreateId:  "create_id",
	UpdateId:  "update_id",
	CreatedAt: "created_at",
	UpdatedAt: "updated_at",
	DeletedAt: "deleted_at",
}

// NewSysTenantPackageDao creates and returns a new DAO object for table data access.
func NewSysTenantPackageDao(handlers ...gdb.ModelHandler) *SysTenantPackageDao {
	return &SysTenantPackageDao{
		group:    "default",
		table:    "sys_tenant_package",
		columns:  sysTenantPackageColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysTenantPackageDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SysTenantPackageDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SysTenantPackageDao) Columns() SysTenantPackageColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SysTenantPackageDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SysTenantPackageDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *SysTenantPackageDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
