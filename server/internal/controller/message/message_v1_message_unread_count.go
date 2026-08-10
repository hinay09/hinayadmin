package message

import (
	"context"

	"hinay.cn/admin/api/message/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) MessageUnreadCount(ctx context.Context, req *v1.MessageUnreadCountReq) (res *v1.MessageUnreadCountRes, err error) {
	return service.Message().UnreadCount(ctx, req)
}
