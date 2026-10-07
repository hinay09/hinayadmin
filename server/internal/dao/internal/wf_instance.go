// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// WfInstanceDao is the data access object for the table wf_instance.
type WfInstanceDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  WfInstanceColumns  // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// WfInstanceColumns defines and stores column names for table wf_instance.
type WfInstanceColumns struct {
	Id             string // 主键ID
	DefinitionId   string // 定义版本行ID
	FlowKey        string // 流程标识(冗余)
	FlowName       string // 流程名称(冗余)
	BizId          string // 业务关联ID(0=审批中心直接发起)
	Title          string // 申请标题
	FormData       string // 提交的表单数据 JSON
	FormConf       string // 表单定义快照(发起时从定义复制)
	FlowConf       string // 节点树快照(发起时从定义复制, 驳回重提沿用)
	CurrentNodeIds string // 当前活跃节点ID(逗号分隔)
	Status         string // 状态:1=运行中,2=已通过,3=已驳回,4=已撤销,5=已终止
	StartUserId    string // 发起人ID
	StartUserName  string // 发起人昵称(冗余)
	FinishedAt     string // 结束时间
	CreateId       string // 创建人ID(ORM自动填充)
	UpdateId       string // 最后修改人ID(ORM自动填充)
	CreatedAt      string // 创建时间
	UpdatedAt      string // 更新时间
	DeletedAt      string // 删除时间(软删)
}

// wfInstanceColumns holds the columns for the table wf_instance.
var wfInstanceColumns = WfInstanceColumns{
	Id:             "id",
	DefinitionId:   "definition_id",
	FlowKey:        "flow_key",
	FlowName:       "flow_name",
	BizId:          "biz_id",
	Title:          "title",
	FormData:       "form_data",
	FormConf:       "form_conf",
	FlowConf:       "flow_conf",
	CurrentNodeIds: "current_node_ids",
	Status:         "status",
	StartUserId:    "start_user_id",
	StartUserName:  "start_user_name",
	FinishedAt:     "finished_at",
	CreateId:       "create_id",
	UpdateId:       "update_id",
	CreatedAt:      "created_at",
	UpdatedAt:      "updated_at",
	DeletedAt:      "deleted_at",
}

// NewWfInstanceDao creates and returns a new DAO object for table data access.
func NewWfInstanceDao(handlers ...gdb.ModelHandler) *WfInstanceDao {
	return &WfInstanceDao{
		group:    "default",
		table:    "wf_instance",
		columns:  wfInstanceColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *WfInstanceDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO object.
func (dao *WfInstanceDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO object.
func (dao *WfInstanceDao) Columns() WfInstanceColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO object.
func (dao *WfInstanceDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO object.
// It automatically sets the context for the current operation.
func (dao *WfInstanceDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
func (dao *WfInstanceDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
