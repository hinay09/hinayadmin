// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysPostDao is the data access object for the table sys_post.
type SysPostDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  SysPostColumns     // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// SysPostColumns defines and stores column names for table sys_post.
type SysPostColumns struct {
	Id        string // 主键ID
	PostCode  string // 岗位编码(唯一, 如 hr, dept_leader)
	PostName  string // 岗位名称
	PostKind  string // 岗位类型:1=普通岗,2=主管岗(部门主管解析依据)
	Sort      string // 排序
	Status    string // 状态:1=启用,0=禁用
	Remark    string // 备注
	CreateId  string // 创建人ID(ORM自动填充)
	UpdateId  string // 最后修改人ID(ORM自动填充)
	CreatedAt string // 创建时间
	UpdatedAt string // 更新时间
	DeletedAt string // 删除时间(软删)

}

// sysPostColumns holds the columns for the table sys_post.
var sysPostColumns = SysPostColumns{
	Id:        "id",
	PostCode:  "post_code",
	PostName:  "post_name",
	PostKind:  "post_kind",
	Sort:      "sort",
	Status:    "status",
	Remark:    "remark",
	CreateId:  "create_id",
	UpdateId:  "update_id",
	CreatedAt: "created_at",
	UpdatedAt: "updated_at",
	DeletedAt: "deleted_at",
}

// NewSysPostDao creates and returns a new DAO object for table data access.
func NewSysPostDao(handlers ...gdb.ModelHandler) *SysPostDao {
	return &SysPostDao{
		group:    "default",
		table:    "sys_post",
		columns:  sysPostColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysPostDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO object.
func (dao *SysPostDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO object.
func (dao *SysPostDao) Columns() SysPostColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO object.
func (dao *SysPostDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO object.
// It automatically sets the context for the current operation.
func (dao *SysPostDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
func (dao *SysPostDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
