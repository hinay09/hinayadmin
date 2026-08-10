// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysOrgDao is the data access object for the table sys_org.
type SysOrgDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  SysOrgColumns      // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// SysOrgColumns defines and stores column names for the table sys_org.
type SysOrgColumns struct {
	Id        string // 组织ID
	ParentId  string // 父级ID, 0=顶级
	Name      string // 组织名称
	Leader    string // 负责人
	Phone     string // 联系电话
	Email     string // 邮箱
	Sort      string // 排序
	Status    string // 状态:1=启用,0=禁用
	Remark    string // 备注
	CreatedAt string // 创建时间
	UpdatedAt string // 更新时间
	DeletedAt string // 删除时间(软删)
}

// sysOrgColumns holds the columns for the table sys_org.
var sysOrgColumns = SysOrgColumns{
	Id:        "id",
	ParentId:  "parent_id",
	Name:      "name",
	Leader:    "leader",
	Phone:     "phone",
	Email:     "email",
	Sort:      "sort",
	Status:    "status",
	Remark:    "remark",
	CreatedAt: "created_at",
	UpdatedAt: "updated_at",
	DeletedAt: "deleted_at",
}

// NewSysOrgDao creates and returns a new DAO object for table data access.
func NewSysOrgDao(handlers ...gdb.ModelHandler) *SysOrgDao {
	return &SysOrgDao{
		group:    "default",
		table:    "sys_org",
		columns:  sysOrgColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysOrgDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SysOrgDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SysOrgDao) Columns() SysOrgColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SysOrgDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SysOrgDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *SysOrgDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
