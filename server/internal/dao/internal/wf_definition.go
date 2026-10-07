// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// WfDefinitionDao is the data access object for the table wf_definition.
type WfDefinitionDao struct {
	table    string              // table is the underlying table name of the DAO.
	group    string              // group is the database configuration group name of the current DAO.
	columns  WfDefinitionColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler  // handlers for customized model modification.
}

// WfDefinitionColumns defines and stores column names for table wf_definition.
type WfDefinitionColumns struct {
	Id        string // 主键ID
	FlowKey   string // 流程标识(同标识共用一组版本, 如 leave)
	Name      string // 流程名称
	FormConf  string // 表单字段定义 JSON [{key,label,type,options,required}]
	FlowConf  string // 节点树定义 JSON {id,type,name,child,...}
	Version   string // 版本:0=草稿,>=1=已发布版本号
	Status    string // 状态:0=草稿,1=已发布,2=已停用
	Remark    string // 备注
	CreateId  string // 创建人ID(ORM自动填充)
	UpdateId  string // 最后修改人ID(ORM自动填充)
	CreatedAt string // 创建时间
	UpdatedAt string // 更新时间
	DeletedAt string // 删除时间(软删)
}

// wfDefinitionColumns holds the columns for the table wf_definition.
var wfDefinitionColumns = WfDefinitionColumns{
	Id:        "id",
	FlowKey:   "flow_key",
	Name:      "name",
	FormConf:  "form_conf",
	FlowConf:  "flow_conf",
	Version:   "version",
	Status:    "status",
	Remark:    "remark",
	CreateId:  "create_id",
	UpdateId:  "update_id",
	CreatedAt: "created_at",
	UpdatedAt: "updated_at",
	DeletedAt: "deleted_at",
}

// NewWfDefinitionDao creates and returns a new DAO object for table data access.
func NewWfDefinitionDao(handlers ...gdb.ModelHandler) *WfDefinitionDao {
	return &WfDefinitionDao{
		group:    "default",
		table:    "wf_definition",
		columns:  wfDefinitionColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *WfDefinitionDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO object.
func (dao *WfDefinitionDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO object.
func (dao *WfDefinitionDao) Columns() WfDefinitionColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO object.
func (dao *WfDefinitionDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO object.
// It automatically sets the context for the current operation.
func (dao *WfDefinitionDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
// It rolls back the transaction and returns the error if function f returns a non-nil error.
// It commits the transaction and returns nil if function f returns a nil error.
//
// Note: Do not commit or roll back the transaction in function f,
// as it is automatically handled by this function.
func (dao *WfDefinitionDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
