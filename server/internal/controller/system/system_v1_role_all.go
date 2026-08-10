package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) RoleAll(ctx context.Context, req *v1.RoleAllReq) (res *v1.RoleAllRes, err error) {
	return service.Role().All(ctx, req)
}
