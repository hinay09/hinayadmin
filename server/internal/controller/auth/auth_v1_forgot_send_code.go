package auth

import (
	"context"

	"hinay.cn/admin/api/auth/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) ForgotSendCode(ctx context.Context, req *v1.ForgotSendCodeReq) (res *v1.ForgotSendCodeRes, err error) {
	return service.Auth().ForgotSendCode(ctx, req)
}
