package auth

import (
	"context"

	"hinay.cn/admin/api/auth/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) Register(ctx context.Context, req *v1.RegisterReq) (res *v1.RegisterRes, err error) {
	return service.Auth().Register(ctx, req)
}
