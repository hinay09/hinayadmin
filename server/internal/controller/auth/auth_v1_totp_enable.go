package auth

import (
	"context"

	"hinay.cn/admin/api/auth/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TotpEnable(ctx context.Context, req *v1.TotpEnableReq) (res *v1.TotpEnableRes, err error) {
	return service.TwoFactor().Enable(ctx, req)
}
