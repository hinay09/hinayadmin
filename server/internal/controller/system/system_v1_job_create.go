package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) JobCreate(ctx context.Context, req *v1.JobCreateReq) (res *v1.JobCreateRes, err error) {
	return service.Job().Create(ctx, req)
}
