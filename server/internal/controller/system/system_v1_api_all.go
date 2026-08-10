package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) ApiAll(ctx context.Context, req *v1.ApiAllReq) (res *v1.ApiAllRes, err error) {
	return service.Api().All(ctx, req)
}
