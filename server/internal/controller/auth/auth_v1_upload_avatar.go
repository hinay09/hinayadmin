package auth

import (
	"context"

	v1 "hinay.cn/admin/api/auth/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) UploadAvatar(ctx context.Context, req *v1.UploadAvatarReq) (res *v1.UploadAvatarRes, err error) {
	return service.Auth().UploadAvatar(ctx, req)
}
