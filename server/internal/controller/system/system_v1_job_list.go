package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) JobList(ctx context.Context, req *v1.JobListReq) (res *v1.JobListRes, err error) {
	return service.Job().List(ctx, req)
}
