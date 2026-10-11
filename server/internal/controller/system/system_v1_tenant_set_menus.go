package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TenantSetMenus(ctx context.Context, req *v1.TenantSetMenusReq) (res *v1.TenantSetMenusRes, err error) {
	return service.Tenant().SetMenus(ctx, req)
}
