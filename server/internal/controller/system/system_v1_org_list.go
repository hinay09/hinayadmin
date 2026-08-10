package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) OrgList(ctx context.Context, req *v1.OrgListReq) (res *v1.OrgListRes, err error) {
	return service.Org().List(ctx, req)
}
