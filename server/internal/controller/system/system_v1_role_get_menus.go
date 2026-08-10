package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) RoleGetMenus(ctx context.Context, req *v1.RoleGetMenusReq) (res *v1.RoleGetMenusRes, err error) {
	return service.Role().GetMenus(ctx, req)
}
