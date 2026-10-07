// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysUserPostDao is the data access object for the table sys_user_post.
type SysUserPostDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  SysUserPostColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// SysUserPostColumns defines and stores column names for table sys_user_post.
type SysUserPostColumns struct {
	Id        string // 主键ID
	UserId    string // 用户ID
	PostId    string // 岗位ID
	OrgId     string // 组织ID(0=不限定组织)
	CreatedAt string // 创建时间
	UpdatedAt string // 更新时间

}

// sysUserPostColumns holds the columns for the table sys_user_post.
var sysUserPostColumns = SysUserPostColumns{
	Id:        "id",
	UserId:    "user_id",
	PostId:    "post_id",
	OrgId:     "org_id",
	CreatedAt: "created_at",
	UpdatedAt: "updated_at",
}

// NewSysUserPostDao creates and returns a new DAO object for table data access.
func NewSysUserPostDao(handlers ...gdb.ModelHandler) *SysUserPostDao {
	return &SysUserPostDao{
		group:    "default",
		table:    "sys_user_post",
		columns:  sysUserPostColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysUserPostDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO object.
func (dao *SysUserPostDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO object.
func (dao *SysUserPostDao) Columns() SysUserPostColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO object.
func (dao *SysUserPostDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO object.
// It automatically sets the context for the current operation.
func (dao *SysUserPostDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
func (dao *SysUserPostDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
