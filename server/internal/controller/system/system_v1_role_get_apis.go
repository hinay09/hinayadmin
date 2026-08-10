package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) RoleGetApis(ctx context.Context, req *v1.RoleGetApisReq) (res *v1.RoleGetApisRes, err error) {
	return service.Role().GetApis(ctx, req)
}
