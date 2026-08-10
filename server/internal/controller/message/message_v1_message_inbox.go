package message

import (
	"context"

	"hinay.cn/admin/api/message/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) MessageInbox(ctx context.Context, req *v1.MessageInboxReq) (res *v1.MessageInboxRes, err error) {
	return service.Message().Inbox(ctx, req)
}
