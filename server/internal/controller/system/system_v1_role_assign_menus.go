package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) RoleAssignMenus(ctx context.Context, req *v1.RoleAssignMenusReq) (res *v1.RoleAssignMenusRes, err error) {
	return service.Role().AssignMenus(ctx, req)
}
