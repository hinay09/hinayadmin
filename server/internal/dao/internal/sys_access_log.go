// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysAccessLogDao is the data access object for the table sys_access_log.
type SysAccessLogDao struct {
	table    string              // table is the underlying table name of the DAO.
	group    string              // group is the database configuration group name of the current DAO.
	columns  SysAccessLogColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler  // handlers for customized model modification.
}

// SysAccessLogColumns defines and stores column names for the table sys_access_log.
type SysAccessLogColumns struct {
	Id         string // ID
	TenantId   string // 租户ID(记录归属, 平台可查全量)
	UserId     string // 用户ID(匿名请求为0)
	Username   string // 用户名(匿名请求为空)
	Method     string // HTTP方法
	Path       string // 请求路径
	Query      string // URL查询串(GET筛选条件)
	StatusCode string // HTTP状态码
	Code       string // 业务码(0=成功; 二进制/超大响应不解析记0)
	DurationMs string // 耗时(毫秒)
	Slow       string // 慢接口标记:1=耗时超过阈值(accessLog.slowMs)
	RequestId  string // 请求ID(链路追踪)
	Ip         string // IP地址
	UserAgent  string // User-Agent
	CreatedAt  string // 创建时间
}

// sysAccessLogColumns holds the columns for the table sys_access_log.
var sysAccessLogColumns = SysAccessLogColumns{
	Id:         "id",
	TenantId:   "tenant_id",
	UserId:     "user_id",
	Username:   "username",
	Method:     "method",
	Path:       "path",
	Query:      "query",
	StatusCode: "status_code",
	Code:       "code",
	DurationMs: "duration_ms",
	Slow:       "slow",
	RequestId:  "request_id",
	Ip:         "ip",
	UserAgent:  "user_agent",
	CreatedAt:  "created_at",
}

// NewSysAccessLogDao creates and returns a new DAO object for table data access.
func NewSysAccessLogDao(handlers ...gdb.ModelHandler) *SysAccessLogDao {
	return &SysAccessLogDao{
		group:    "default",
		table:    "sys_access_log",
		columns:  sysAccessLogColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysAccessLogDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SysAccessLogDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SysAccessLogDao) Columns() SysAccessLogColumns {
	return dao.columns
}

// Group returns the configuration group name of the current DAO.
func (dao *SysAccessLogDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SysAccessLogDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
// It rolls back the transaction and returns the error if the function f returns a non-nil error.
// It commits the transaction and returns nil if the function f returns nil.
//
// Note: Do not commit or roll back the transaction in function f,
// as it is automatically handled by this function.
func (dao *SysAccessLogDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
