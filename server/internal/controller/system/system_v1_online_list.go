package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) OnlineList(ctx context.Context, req *v1.OnlineListReq) (res *v1.OnlineListRes, err error) {
	return service.Online().List(ctx, req)
}
