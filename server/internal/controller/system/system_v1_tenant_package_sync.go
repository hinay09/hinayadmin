package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TenantPackageSync(ctx context.Context, req *v1.TenantPackageSyncReq) (res *v1.TenantPackageSyncRes, err error) {
	return service.TenantPackage().Sync(ctx, req)
}
