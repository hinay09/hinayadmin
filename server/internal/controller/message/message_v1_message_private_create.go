package message

import (
	"context"

	"hinay.cn/admin/api/message/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) MessagePrivateCreate(ctx context.Context, req *v1.MessagePrivateCreateReq) (res *v1.MessagePrivateCreateRes, err error) {
	return service.Message().PrivateCreate(ctx, req)
}
