package auth

import (
	"context"

	"hinay.cn/admin/api/auth/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TenantInfo(ctx context.Context, req *v1.TenantInfoReq) (res *v1.TenantInfoRes, err error) {
	return service.Auth().TenantInfo(ctx, req)
}
