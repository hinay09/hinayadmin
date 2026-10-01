package auth

import (
	"context"

	"hinay.cn/admin/api/auth/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TotpLogin(ctx context.Context, req *v1.TotpLoginReq) (res *v1.TotpLoginRes, err error) {
	return service.Auth().TotpLogin(ctx, req)
}
