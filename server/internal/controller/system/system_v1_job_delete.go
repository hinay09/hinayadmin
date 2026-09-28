package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) JobDelete(ctx context.Context, req *v1.JobDeleteReq) (res *v1.JobDeleteRes, err error) {
	return service.Job().Delete(ctx, req)
}
