// Package flow 审批任务控制器 (薄透传)。
package flow

import (
	"context"

	"hinay.cn/admin/api/flow/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) TaskApprove(ctx context.Context, req *v1.FlowTaskApproveReq) (res *v1.FlowTaskApproveRes, err error) {
	return service.Flow().TaskApprove(ctx, req)
}

func (c *ControllerV1) TaskReject(ctx context.Context, req *v1.FlowTaskRejectReq) (res *v1.FlowTaskRejectRes, err error) {
	return service.Flow().TaskReject(ctx, req)
}

func (c *ControllerV1) TaskTransfer(ctx context.Context, req *v1.FlowTaskTransferReq) (res *v1.FlowTaskTransferRes, err error) {
	return service.Flow().TaskTransfer(ctx, req)
}

func (c *ControllerV1) TaskAppend(ctx context.Context, req *v1.FlowTaskAppendReq) (res *v1.FlowTaskAppendRes, err error) {
	return service.Flow().TaskAppend(ctx, req)
}

func (c *ControllerV1) TaskReduce(ctx context.Context, req *v1.FlowTaskReduceReq) (res *v1.FlowTaskReduceRes, err error) {
	return service.Flow().TaskReduce(ctx, req)
}

func (c *ControllerV1) TaskRead(ctx context.Context, req *v1.FlowTaskReadReq) (res *v1.FlowTaskReadRes, err error) {
	return service.Flow().TaskRead(ctx, req)
}

func (c *ControllerV1) TaskCount(ctx context.Context, req *v1.FlowTaskCountReq) (res *v1.FlowTaskCountRes, err error) {
	return service.Flow().TaskCount(ctx, req)
}
