// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysTenantDao is the data access object for the table sys_tenant.
type SysTenantDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  SysTenantColumns   // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// SysTenantColumns defines and stores column names for the table sys_tenant.
type SysTenantColumns struct {
	Id           string //
	Name         string // 租户名称
	Code         string // 租户标识(子域名/登录码, 全局唯一, 创建后不可变)
	Contact      string // 联系人
	Plan         string // 套餐(展示标签, 预留)
	PackageId    string // 套餐ID(sys_tenant_package, 0=自定义/未绑定)
	QuotaUsers   string // 用户数配额(0=不限, 预留)
	QuotaStorage string // 存储配额字节(0=不限, 预留)
	ExpireAt     string // 到期时间(NULL=永久)
	Status       string // 1=启用 0=禁用
	Remark       string // 备注
	CreateId     string //
	UpdateId     string //
	CreatedAt    string //
	UpdatedAt    string //
	DeletedAt    string //
}

// sysTenantColumns holds the columns for the table sys_tenant.
var sysTenantColumns = SysTenantColumns{
	Id:           "id",
	Name:         "name",
	Code:         "code",
	Contact:      "contact",
	Plan:         "plan",
	PackageId:    "package_id",
	QuotaUsers:   "quota_users",
	QuotaStorage: "quota_storage",
	ExpireAt:     "expire_at",
	Status:       "status",
	Remark:       "remark",
	CreateId:     "create_id",
	UpdateId:     "update_id",
	CreatedAt:    "created_at",
	UpdatedAt:    "updated_at",
	DeletedAt:    "deleted_at",
}

// NewSysTenantDao creates and returns a new DAO object for table data access.
func NewSysTenantDao(handlers ...gdb.ModelHandler) *SysTenantDao {
	return &SysTenantDao{
		group:    "default",
		table:    "sys_tenant",
		columns:  sysTenantColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysTenantDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SysTenantDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SysTenantDao) Columns() SysTenantColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SysTenantDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SysTenantDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *SysTenantDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
