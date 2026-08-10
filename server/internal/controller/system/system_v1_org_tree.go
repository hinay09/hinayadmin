package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) OrgTree(ctx context.Context, req *v1.OrgTreeReq) (res *v1.OrgTreeRes, err error) {
	return service.Org().Tree(ctx, req)
}
