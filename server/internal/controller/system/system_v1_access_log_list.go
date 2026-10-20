package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) AccessLogList(ctx context.Context, req *v1.AccessLogListReq) (res *v1.AccessLogListRes, err error) {
	return service.AccessLog().List(ctx, req)
}
