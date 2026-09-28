package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) JobUpdate(ctx context.Context, req *v1.JobUpdateReq) (res *v1.JobUpdateRes, err error) {
	return service.Job().Update(ctx, req)
}
