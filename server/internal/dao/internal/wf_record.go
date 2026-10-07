// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// WfRecordDao is the data access object for the table wf_record.
type WfRecordDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  WfRecordColumns    // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// WfRecordColumns defines and stores column names for table wf_record.
type WfRecordColumns struct {
	Id           string // 主键ID
	InstanceId   string // 实例ID
	TaskId       string // 关联任务ID(无则为0)
	NodeId       string // 节点ID
	NodeName     string // 节点名称
	Action       string // 动作:submit/approve/reject/cancel/cc/finish
	OperatorId   string // 操作人ID(0=系统)
	OperatorName string // 操作人昵称(0=系统)
	Comment      string // 备注/意见
	CreatedAt    string // 创建时间
	UpdatedAt    string // 更新时间
}

// wfRecordColumns holds the columns for the table wf_record.
var wfRecordColumns = WfRecordColumns{
	Id:           "id",
	InstanceId:   "instance_id",
	TaskId:       "task_id",
	NodeId:       "node_id",
	NodeName:     "node_name",
	Action:       "action",
	OperatorId:   "operator_id",
	OperatorName: "operator_name",
	Comment:      "comment",
	CreatedAt:    "created_at",
	UpdatedAt:    "updated_at",
}

// NewWfRecordDao creates and returns a new DAO object for table data access.
func NewWfRecordDao(handlers ...gdb.ModelHandler) *WfRecordDao {
	return &WfRecordDao{
		group:    "default",
		table:    "wf_record",
		columns:  wfRecordColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *WfRecordDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO object.
func (dao *WfRecordDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO object.
func (dao *WfRecordDao) Columns() WfRecordColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO object.
func (dao *WfRecordDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO object.
// It automatically sets the context for the current operation.
func (dao *WfRecordDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
func (dao *WfRecordDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
