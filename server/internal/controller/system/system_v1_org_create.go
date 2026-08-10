package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) OrgCreate(ctx context.Context, req *v1.OrgCreateReq) (res *v1.OrgCreateRes, err error) {
	return service.Org().Create(ctx, req)
}
