package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) ConfigAll(ctx context.Context, req *v1.ConfigAllReq) (res *v1.ConfigAllRes, err error) {
	return service.Config().All(ctx, req)
}
