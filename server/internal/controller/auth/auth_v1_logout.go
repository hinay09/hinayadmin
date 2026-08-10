package auth

import (
	"context"

	"hinay.cn/admin/internal/service"

	"hinay.cn/admin/api/auth/v1"
)

func (c *ControllerV1) Logout(ctx context.Context, req *v1.LogoutReq) (res *v1.LogoutRes, err error) {
	return service.Auth().Logout(ctx, req)
}
