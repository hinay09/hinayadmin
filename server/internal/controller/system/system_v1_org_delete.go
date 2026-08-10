package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) OrgDelete(ctx context.Context, req *v1.OrgDeleteReq) (res *v1.OrgDeleteRes, err error) {
	return service.Org().Delete(ctx, req)
}
