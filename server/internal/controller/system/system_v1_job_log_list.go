package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) JobLogList(ctx context.Context, req *v1.JobLogListReq) (res *v1.JobLogListRes, err error) {
	return service.Job().LogList(ctx, req)
}
