package message

import (
	"context"

	"hinay.cn/admin/api/message/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) MessageInboxDelete(ctx context.Context, req *v1.MessageInboxDeleteReq) (res *v1.MessageInboxDeleteRes, err error) {
	return service.Message().InboxDelete(ctx, req)
}
