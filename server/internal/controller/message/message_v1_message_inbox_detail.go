package message

import (
	"context"

	"hinay.cn/admin/api/message/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) MessageInboxDetail(ctx context.Context, req *v1.MessageInboxDetailReq) (res *v1.MessageInboxDetailRes, err error) {
	return service.Message().InboxRead(ctx, req)
}
