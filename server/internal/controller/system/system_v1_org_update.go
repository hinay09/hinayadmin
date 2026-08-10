package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) OrgUpdate(ctx context.Context, req *v1.OrgUpdateReq) (res *v1.OrgUpdateRes, err error) {
	return service.Org().Update(ctx, req)
}
