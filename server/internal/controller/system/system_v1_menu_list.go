package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) MenuList(ctx context.Context, req *v1.MenuListReq) (res *v1.MenuListRes, err error) {
	return service.Menu().List(ctx, req)
}
