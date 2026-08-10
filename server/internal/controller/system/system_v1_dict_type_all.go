package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) DictTypeAll(ctx context.Context, req *v1.DictTypeAllReq) (res *v1.DictTypeAllRes, err error) {
	return service.DictType().All(ctx, req)
}
