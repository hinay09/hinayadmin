package message

import (
	"context"

	"hinay.cn/admin/api/message/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) MessageDelete(ctx context.Context, req *v1.MessageDeleteReq) (res *v1.MessageDeleteRes, err error) {
	return service.Message().Delete(ctx, req)
}
