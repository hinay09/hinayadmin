package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) JobHandlers(ctx context.Context, req *v1.JobHandlersReq) (res *v1.JobHandlersRes, err error) {
	return &v1.JobHandlersRes{Handlers: service.Job().Handlers()}, nil
}
