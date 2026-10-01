// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysJobDao is the data access object for the table sys_job.
type SysJobDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  SysJobColumns      // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// SysJobColumns defines and stores column names for the table sys_job.
type SysJobColumns struct {
	Id        string // 任务ID
	Name      string // 任务名称
	Handler   string // 处理器名称(需已注册)
	CronExpr  string // cron表达式(6位: 秒 分 时 日 月 周)
	Params    string // 任务参数(JSON, 可空)
	Status    string // 状态:1=启动,0=暂停
	Remark    string // 备注
	CreateId  string // 创建人ID(ORM自动填充)
	UpdateId  string // 最后修改人ID(ORM自动填充)
	CreatedAt string // 创建时间
	UpdatedAt string // 更新时间
	DeletedAt string // 删除时间(软删)
}

// sysJobColumns holds the columns for the table sys_job.
var sysJobColumns = SysJobColumns{
	Id:        "id",
	Name:      "name",
	Handler:   "handler",
	CronExpr:  "cron_expr",
	Params:    "params",
	Status:    "status",
	Remark:    "remark",
	CreateId:  "create_id",
	UpdateId:  "update_id",
	CreatedAt: "created_at",
	UpdatedAt: "updated_at",
	DeletedAt: "deleted_at",
}

// NewSysJobDao creates and returns a new DAO object for table data access.
func NewSysJobDao(handlers ...gdb.ModelHandler) *SysJobDao {
	return &SysJobDao{
		group:    "default",
		table:    "sys_job",
		columns:  sysJobColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysJobDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SysJobDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SysJobDao) Columns() SysJobColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SysJobDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SysJobDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *SysJobDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
