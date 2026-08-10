package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) ApiDelete(ctx context.Context, req *v1.ApiDeleteReq) (res *v1.ApiDeleteRes, err error) {
	return service.Api().Delete(ctx, req)
}
