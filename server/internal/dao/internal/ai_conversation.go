// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// AiConversationDao is the data access object for the table ai_conversation.
type AiConversationDao struct {
	table    string                // table is the underlying table name of the DAO.
	group    string                // group is the database configuration group name of the current DAO.
	columns  AiConversationColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler    // handlers for customized model modification.
}

// AiConversationColumns defines and stores column names for the table ai_conversation.
type AiConversationColumns struct {
	Id           string // 主键ID
	SessionId    string // 会话ID(前端生成并保管)
	UserId       string // 所属用户ID
	Title        string // 会话标题(首条用户消息裁剪)
	MessageCount string // 累计消息条数(user+assistant)
	CreatedAt    string // 创建时间
	UpdatedAt    string // 最近一轮对话时间
	DeletedAt    string // 未使用(保留列对齐代码生成器约定)
}

// aiConversationColumns holds the columns for the table ai_conversation.
var aiConversationColumns = AiConversationColumns{
	Id:           "id",
	SessionId:    "session_id",
	UserId:       "user_id",
	Title:        "title",
	MessageCount: "message_count",
	CreatedAt:    "created_at",
	UpdatedAt:    "updated_at",
	DeletedAt:    "deleted_at",
}

// NewAiConversationDao creates and returns a new DAO object for table data access.
func NewAiConversationDao(handlers ...gdb.ModelHandler) *AiConversationDao {
	return &AiConversationDao{
		group:    "default",
		table:    "ai_conversation",
		columns:  aiConversationColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *AiConversationDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *AiConversationDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *AiConversationDao) Columns() AiConversationColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *AiConversationDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *AiConversationDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *AiConversationDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
