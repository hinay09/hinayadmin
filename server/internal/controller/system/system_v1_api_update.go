package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) ApiUpdate(ctx context.Context, req *v1.ApiUpdateReq) (res *v1.ApiUpdateRes, err error) {
	return service.Api().Update(ctx, req)
}
