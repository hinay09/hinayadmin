package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TenantPackageListAll(ctx context.Context, req *v1.TenantPackageListAllReq) (res *v1.TenantPackageListAllRes, err error) {
	return service.TenantPackage().ListAll(ctx, req)
}
