package auth

import (
	"context"

	"hinay.cn/admin/api/auth/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) RegisterStatus(ctx context.Context, req *v1.RegisterStatusReq) (res *v1.RegisterStatusRes, err error) {
	return service.Auth().RegisterStatus(ctx, req)
}
