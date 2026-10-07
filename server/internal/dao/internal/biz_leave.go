// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// BizLeaveDao is the data access object for the table biz_leave.
type BizLeaveDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  BizLeaveColumns    // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// BizLeaveColumns defines and stores column names for the table biz_leave.
type BizLeaveColumns struct {
	Id           string // 主键ID
	LeaveType    string // 请假类型:1=事假,2=病假,3=年假,4=调休,5=其他
	StartDate    string // 开始日期
	EndDate      string // 结束日期
	Days         string // 请假天数(0.5天粒度,申请人填报)
	Reason       string // 请假事由
	FlowStatus   string // 审批状态:0=审批中,1=已通过,2=被退回,3=已撤销,4=已终止(引擎回调写入;未发起时无意义)
	FlowInstance string // 流程实例ID(0=未发起/草稿)
	CreateId     string // 创建人ID(ORM自动填充)
	UpdateId     string // 最后修改人ID(ORM自动填充)
	CreatedAt    string // 创建时间
	UpdatedAt    string // 更新时间
	DeletedAt    string // 删除时间(软删)
}

// bizLeaveColumns holds the columns for the table biz_leave.
var bizLeaveColumns = BizLeaveColumns{
	Id:           "id",
	LeaveType:    "leave_type",
	StartDate:    "start_date",
	EndDate:      "end_date",
	Days:         "days",
	Reason:       "reason",
	FlowStatus:   "flow_status",
	FlowInstance: "flow_instance",
	CreateId:     "create_id",
	UpdateId:     "update_id",
	CreatedAt:    "created_at",
	UpdatedAt:    "updated_at",
	DeletedAt:    "deleted_at",
}

// NewBizLeaveDao creates and returns a new DAO object for table data access.
func NewBizLeaveDao(handlers ...gdb.ModelHandler) *BizLeaveDao {
	return &BizLeaveDao{
		group:    "default",
		table:    "biz_leave",
		columns:  bizLeaveColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *BizLeaveDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *BizLeaveDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *BizLeaveDao) Columns() BizLeaveColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *BizLeaveDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *BizLeaveDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *BizLeaveDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
