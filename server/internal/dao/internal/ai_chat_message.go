// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// AiChatMessageDao is the data access object for the table ai_chat_message.
type AiChatMessageDao struct {
	table    string               // table is the underlying table name of the DAO.
	group    string               // group is the database configuration group name of the current DAO.
	columns  AiChatMessageColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler   // handlers for customized model modification.
}

// AiChatMessageColumns defines and stores column names for the table ai_chat_message.
type AiChatMessageColumns struct {
	Id             string // 主键ID
	ConversationId string // 会话ID(ai_conversation.id)
	Role           string // 角色:user/assistant
	Content        string // 消息正文(思考过程不落库)
	CreatedAt      string // 创建时间
	UpdatedAt      string // 更新时间
	DeletedAt      string // 未使用(保留列对齐代码生成器约定)
}

// aiChatMessageColumns holds the columns for the table ai_chat_message.
var aiChatMessageColumns = AiChatMessageColumns{
	Id:             "id",
	ConversationId: "conversation_id",
	Role:           "role",
	Content:        "content",
	CreatedAt:      "created_at",
	UpdatedAt:      "updated_at",
	DeletedAt:      "deleted_at",
}

// NewAiChatMessageDao creates and returns a new DAO object for table data access.
func NewAiChatMessageDao(handlers ...gdb.ModelHandler) *AiChatMessageDao {
	return &AiChatMessageDao{
		group:    "default",
		table:    "ai_chat_message",
		columns:  aiChatMessageColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *AiChatMessageDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *AiChatMessageDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *AiChatMessageDao) Columns() AiChatMessageColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *AiChatMessageDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *AiChatMessageDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *AiChatMessageDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
