package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TenantPackageList(ctx context.Context, req *v1.TenantPackageListReq) (res *v1.TenantPackageListRes, err error) {
	return service.TenantPackage().List(ctx, req)
}
