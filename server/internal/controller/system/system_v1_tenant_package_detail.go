package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TenantPackageDetail(ctx context.Context, req *v1.TenantPackageDetailReq) (res *v1.TenantPackageDetailRes, err error) {
	return service.TenantPackage().Detail(ctx, req)
}
