package auth

import (
	"context"

	"hinay.cn/admin/api/auth/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) ForgotReset(ctx context.Context, req *v1.ForgotResetReq) (res *v1.ForgotResetRes, err error) {
	return service.Auth().ForgotReset(ctx, req)
}
