package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) ApiList(ctx context.Context, req *v1.ApiListReq) (res *v1.ApiListRes, err error) {
	return service.Api().List(ctx, req)
}
