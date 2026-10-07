// Package leave 请假申请控制器 (薄透传)。
package leave

import (
	"context"

	"hinay.cn/admin/api/leave/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) LeaveList(ctx context.Context, req *v1.LeaveListReq) (res *v1.LeaveListRes, err error) {
	return service.Leave().LeaveList(ctx, req)
}

func (c *ControllerV1) LeaveCreate(ctx context.Context, req *v1.LeaveCreateReq) (res *v1.LeaveCreateRes, err error) {
	return service.Leave().LeaveCreate(ctx, req)
}

func (c *ControllerV1) LeaveUpdate(ctx context.Context, req *v1.LeaveUpdateReq) (res *v1.LeaveUpdateRes, err error) {
	return service.Leave().LeaveUpdate(ctx, req)
}

func (c *ControllerV1) LeaveDelete(ctx context.Context, req *v1.LeaveDeleteReq) (res *v1.LeaveDeleteRes, err error) {
	return service.Leave().LeaveDelete(ctx, req)
}

func (c *ControllerV1) LeaveSubmit(ctx context.Context, req *v1.LeaveSubmitReq) (res *v1.LeaveSubmitRes, err error) {
	return service.Leave().LeaveSubmit(ctx, req)
}

func (c *ControllerV1) LeaveCancel(ctx context.Context, req *v1.LeaveCancelReq) (res *v1.LeaveCancelRes, err error) {
	return service.Leave().LeaveCancel(ctx, req)
}
