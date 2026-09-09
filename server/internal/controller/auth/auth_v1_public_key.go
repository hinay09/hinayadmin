package auth

import (
	"context"

	"hinay.cn/admin/api/auth/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) PublicKey(ctx context.Context, req *v1.PublicKeyReq) (res *v1.PublicKeyRes, err error) {
	return service.Auth().PublicKey(ctx, req)
}
