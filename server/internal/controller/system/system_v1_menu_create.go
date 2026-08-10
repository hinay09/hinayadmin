package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) MenuCreate(ctx context.Context, req *v1.MenuCreateReq) (res *v1.MenuCreateRes, err error) {
	return service.Menu().Create(ctx, req)
}
