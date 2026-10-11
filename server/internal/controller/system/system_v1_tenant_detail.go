package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TenantDetail(ctx context.Context, req *v1.TenantDetailReq) (res *v1.TenantDetailRes, err error) {
	return service.Tenant().Detail(ctx, req)
}
