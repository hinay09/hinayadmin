package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) LoginLogDelete(ctx context.Context, req *v1.LoginLogDeleteReq) (res *v1.LoginLogDeleteRes, err error) {
	return service.LoginLog().Delete(ctx, req)
}
