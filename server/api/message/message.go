// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package message

import (
	"context"

	"hinay.cn/admin/api/message/v1"
)

type IMessageV1 interface {
	MessageList(ctx context.Context, req *v1.MessageListReq) (res *v1.MessageListRes, err error)
	MessageInbox(ctx context.Context, req *v1.MessageInboxReq) (res *v1.MessageInboxRes, err error)
	MessageDetail(ctx context.Context, req *v1.MessageDetailReq) (res *v1.MessageDetailRes, err error)
	MessageInboxDetail(ctx context.Context, req *v1.MessageInboxDetailReq) (res *v1.MessageInboxDetailRes, err error)
	MessageSystemCreate(ctx context.Context, req *v1.MessageSystemCreateReq) (res *v1.MessageSystemCreateRes, err error)
	MessagePrivateCreate(ctx context.Context, req *v1.MessagePrivateCreateReq) (res *v1.MessagePrivateCreateRes, err error)
	MessageDelete(ctx context.Context, req *v1.MessageDeleteReq) (res *v1.MessageDeleteRes, err error)
	MessageInboxDelete(ctx context.Context, req *v1.MessageInboxDeleteReq) (res *v1.MessageInboxDeleteRes, err error)
	MessageRead(ctx context.Context, req *v1.MessageReadReq) (res *v1.MessageReadRes, err error)
	MessageReadAll(ctx context.Context, req *v1.MessageReadAllReq) (res *v1.MessageReadAllRes, err error)
	MessageUnreadCount(ctx context.Context, req *v1.MessageUnreadCountReq) (res *v1.MessageUnreadCountRes, err error)
}
