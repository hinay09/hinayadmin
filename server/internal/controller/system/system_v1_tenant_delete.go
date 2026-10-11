package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TenantDelete(ctx context.Context, req *v1.TenantDeleteReq) (res *v1.TenantDeleteRes, err error) {
	return service.Tenant().Delete(ctx, req)
}
