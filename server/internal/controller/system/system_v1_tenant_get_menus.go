package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TenantGetMenus(ctx context.Context, req *v1.TenantGetMenusReq) (res *v1.TenantGetMenusRes, err error) {
	return service.Tenant().GetMenus(ctx, req)
}
