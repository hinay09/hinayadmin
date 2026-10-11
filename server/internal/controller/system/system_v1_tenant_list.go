package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TenantList(ctx context.Context, req *v1.TenantListReq) (res *v1.TenantListRes, err error) {
	return service.Tenant().List(ctx, req)
}
