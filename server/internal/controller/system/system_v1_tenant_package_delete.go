package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TenantPackageDelete(ctx context.Context, req *v1.TenantPackageDeleteReq) (res *v1.TenantPackageDeleteRes, err error) {
	return service.TenantPackage().Delete(ctx, req)
}
