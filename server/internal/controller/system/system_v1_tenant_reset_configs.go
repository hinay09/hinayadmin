package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TenantResetConfigs(ctx context.Context, req *v1.TenantResetConfigsReq) (res *v1.TenantResetConfigsRes, err error) {
	return service.Tenant().ResetConfigs(ctx, req)
}
