package message

import (
	"context"

	"hinay.cn/admin/api/message/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) MessageDetail(ctx context.Context, req *v1.MessageDetailReq) (res *v1.MessageDetailRes, err error) {
	return service.Message().Detail(ctx, req)
}
