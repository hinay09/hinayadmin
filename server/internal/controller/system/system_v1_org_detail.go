package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) OrgDetail(ctx context.Context, req *v1.OrgDetailReq) (res *v1.OrgDetailRes, err error) {
	return service.Org().Detail(ctx, req)
}
