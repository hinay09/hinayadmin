package auth

import (
	"context"

	"hinay.cn/admin/api/auth/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TotpDisable(ctx context.Context, req *v1.TotpDisableReq) (res *v1.TotpDisableRes, err error) {
	return service.TwoFactor().Disable(ctx, req)
}
