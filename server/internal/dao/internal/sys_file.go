// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysFileDao is the data access object for the table sys_file.
type SysFileDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  SysFileColumns     // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// SysFileColumns defines and stores column names for the table sys_file.
type SysFileColumns struct {
	Id           string // ID
	Name         string // 存储文件名
	OriginalName string // 原始文件名
	Path         string // 存储路径
	Url          string // 访问URL
	Size         string // 文件大小(字节)
	MimeType     string // MIME类型
	Extension    string // 文件扩展名
	UserId       string // 上传用户ID
	CreatedAt    string // 创建时间
	DeletedAt    string // 删除时间(软删)
}

// sysFileColumns holds the columns for the table sys_file.
var sysFileColumns = SysFileColumns{
	Id:           "id",
	Name:         "name",
	OriginalName: "original_name",
	Path:         "path",
	Url:          "url",
	Size:         "size",
	MimeType:     "mime_type",
	Extension:    "extension",
	UserId:       "user_id",
	CreatedAt:    "created_at",
	DeletedAt:    "deleted_at",
}

// NewSysFileDao creates and returns a new DAO object for table data access.
func NewSysFileDao(handlers ...gdb.ModelHandler) *SysFileDao {
	return &SysFileDao{
		group:    "default",
		table:    "sys_file",
		columns:  sysFileColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysFileDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SysFileDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SysFileDao) Columns() SysFileColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SysFileDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SysFileDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *SysFileDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
