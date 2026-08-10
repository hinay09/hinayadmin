package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) ApiCreate(ctx context.Context, req *v1.ApiCreateReq) (res *v1.ApiCreateRes, err error) {
	return service.Api().Create(ctx, req)
}
