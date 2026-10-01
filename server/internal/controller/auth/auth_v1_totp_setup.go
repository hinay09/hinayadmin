package auth

import (
	"context"

	"hinay.cn/admin/api/auth/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TotpSetup(ctx context.Context, req *v1.TotpSetupReq) (res *v1.TotpSetupRes, err error) {
	return service.TwoFactor().Setup(ctx, req)
}
