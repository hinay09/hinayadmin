package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) JobChangeStatus(ctx context.Context, req *v1.JobChangeStatusReq) (res *v1.JobChangeStatusRes, err error) {
	return service.Job().ChangeStatus(ctx, req)
}
