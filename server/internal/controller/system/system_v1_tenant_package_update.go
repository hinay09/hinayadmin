package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TenantPackageUpdate(ctx context.Context, req *v1.TenantPackageUpdateReq) (res *v1.TenantPackageUpdateRes, err error) {
	return service.TenantPackage().Update(ctx, req)
}
