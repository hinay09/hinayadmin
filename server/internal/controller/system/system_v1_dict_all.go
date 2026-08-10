package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) DictAll(ctx context.Context, req *v1.DictAllReq) (res *v1.DictAllRes, err error) {
	return service.Dict().All(ctx, req)
}
