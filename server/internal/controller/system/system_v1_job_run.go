package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) JobRun(ctx context.Context, req *v1.JobRunReq) (res *v1.JobRunRes, err error) {
	return service.Job().Run(ctx, req)
}
