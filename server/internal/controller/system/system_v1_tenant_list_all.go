package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TenantListAll(ctx context.Context, req *v1.TenantListAllReq) (res *v1.TenantListAllRes, err error) {
	return service.Tenant().ListAll(ctx, req)
}
