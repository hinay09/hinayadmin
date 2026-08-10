// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	v1 "hinay.cn/admin/api/message/v1"
)

type (
	IMessage interface {
		// List 管理员视角的消息分页列表。
		List(ctx context.Context, in *v1.MessageListReq) (res *v1.MessageListRes, err error)
		// Detail 管理员视角详情。
		Detail(ctx context.Context, in *v1.MessageDetailReq) (res *v1.MessageDetailRes, err error)
		// SystemCreate 发布系统通知。事务: 写主表 + 批量写定向目标。
		SystemCreate(ctx context.Context, in *v1.MessageSystemCreateReq) (res *v1.MessageSystemCreateRes, err error)
		// PrivateCreate 发送私信。
		PrivateCreate(ctx context.Context, in *v1.MessagePrivateCreateReq) (res *v1.MessagePrivateCreateRes, err error)
		// Delete 管理员软删除消息 (同时清理 target/read 关系)。
		Delete(ctx context.Context, in *v1.MessageDeleteReq) (res *v1.MessageDeleteRes, err error)
		// Inbox 我的收件箱分页。
		Inbox(ctx context.Context, in *v1.MessageInboxReq) (res *v1.MessageInboxRes, err error)
		// InboxRead 阅读详情, 自动标记已读。
		InboxRead(ctx context.Context, in *v1.MessageInboxDetailReq) (res *v1.MessageInboxDetailRes, err error)
		// MarkRead 仅标记已读, 不返回详情。
		MarkRead(ctx context.Context, in *v1.MessageReadReq) (res *v1.MessageReadRes, err error)
		// MarkReadAll 全部标记为已读 (返回受影响条数)。
		MarkReadAll(ctx context.Context, in *v1.MessageReadAllReq) (res *v1.MessageReadAllRes, err error)
		// InboxDelete 个人删除 (从我的收件箱视角隐藏)。
		InboxDelete(ctx context.Context, in *v1.MessageInboxDeleteReq) (res *v1.MessageInboxDeleteRes, err error)
		// UnreadCount 未读消息数量 (总数 / 系统通知 / 私信)。
		UnreadCount(ctx context.Context, in *v1.MessageUnreadCountReq) (res *v1.MessageUnreadCountRes, err error)
	}
)

var (
	localMessage IMessage
)

func Message() IMessage {
	if localMessage == nil {
		panic("implement not found for interface IMessage, forgot register?")
	}
	return localMessage
}

func RegisterMessage(i IMessage) {
	localMessage = i
}
