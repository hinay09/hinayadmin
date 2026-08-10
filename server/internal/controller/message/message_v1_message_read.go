package message

import (
	"context"

	"hinay.cn/admin/api/message/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) MessageRead(ctx context.Context, req *v1.MessageReadReq) (res *v1.MessageReadRes, err error) {
	return service.Message().MarkRead(ctx, req)
}
