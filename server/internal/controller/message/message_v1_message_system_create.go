package message

import (
	"context"

	"hinay.cn/admin/api/message/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) MessageSystemCreate(ctx context.Context, req *v1.MessageSystemCreateReq) (res *v1.MessageSystemCreateRes, err error) {
	return service.Message().SystemCreate(ctx, req)
}
