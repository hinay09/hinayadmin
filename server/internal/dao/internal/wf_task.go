// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// WfTaskDao is the data access object for the table wf_task.
type WfTaskDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  WfTaskColumns      // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// WfTaskColumns defines and stores column names for table wf_task.
type WfTaskColumns struct {
	Id           string // 主键ID
	InstanceId   string // 实例ID
	NodeId       string // 节点ID(树内唯一)
	NodeName     string // 节点名称(冗余)
	NodeType     string // 节点类型:1=审批,2=抄送
	SignType     string // 签核方式:1=或签,2=会签
	AssigneeId   string // 处理人ID
	AssigneeName string // 处理人昵称(冗余)
	Status       string // 状态:1=待办,2=已同意,3=已驳回,4=已转出,5=已作废
	Comment      string // 审批意见
	ReceiveTime  string // 到达时间
	ActedAt      string // 处理时间
	CreatedAt    string // 创建时间
	UpdatedAt    string // 更新时间
}

// wfTaskColumns holds the columns for the table wf_task.
var wfTaskColumns = WfTaskColumns{
	Id:           "id",
	InstanceId:   "instance_id",
	NodeId:       "node_id",
	NodeName:     "node_name",
	NodeType:     "node_type",
	SignType:     "sign_type",
	AssigneeId:   "assignee_id",
	AssigneeName: "assignee_name",
	Status:       "status",
	Comment:      "comment",
	ReceiveTime:  "receive_time",
	ActedAt:      "acted_at",
	CreatedAt:    "created_at",
	UpdatedAt:    "updated_at",
}

// NewWfTaskDao creates and returns a new DAO object for table data access.
func NewWfTaskDao(handlers ...gdb.ModelHandler) *WfTaskDao {
	return &WfTaskDao{
		group:    "default",
		table:    "wf_task",
		columns:  wfTaskColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *WfTaskDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO object.
func (dao *WfTaskDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO object.
func (dao *WfTaskDao) Columns() WfTaskColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO object.
func (dao *WfTaskDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO object.
// It automatically sets the context for the current operation.
func (dao *WfTaskDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
func (dao *WfTaskDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
