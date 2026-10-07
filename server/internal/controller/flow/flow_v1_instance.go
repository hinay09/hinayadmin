// Package flow 流程实例控制器 (薄透传)。
package flow

import (
	"context"

	"hinay.cn/admin/api/flow/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) InstanceStart(ctx context.Context, req *v1.FlowInstanceStartReq) (res *v1.FlowInstanceStartRes, err error) {
	return service.Flow().InstanceStart(ctx, req)
}

func (c *ControllerV1) InstanceList(ctx context.Context, req *v1.FlowInstanceListReq) (res *v1.FlowInstanceListRes, err error) {
	return service.Flow().InstanceList(ctx, req)
}

func (c *ControllerV1) InstanceDetail(ctx context.Context, req *v1.FlowInstanceDetailReq) (res *v1.FlowInstanceDetailRes, err error) {
	return service.Flow().InstanceDetail(ctx, req)
}

func (c *ControllerV1) InstanceCancel(ctx context.Context, req *v1.FlowInstanceCancelReq) (res *v1.FlowInstanceCancelRes, err error) {
	return service.Flow().InstanceCancel(ctx, req)
}

func (c *ControllerV1) InstanceWithdraw(ctx context.Context, req *v1.FlowInstanceWithdrawReq) (res *v1.FlowInstanceWithdrawRes, err error) {
	return service.Flow().InstanceWithdraw(ctx, req)
}

func (c *ControllerV1) InstanceResubmit(ctx context.Context, req *v1.FlowInstanceResubmitReq) (res *v1.FlowInstanceResubmitRes, err error) {
	return service.Flow().InstanceResubmit(ctx, req)
}

func (c *ControllerV1) InstanceTerminate(ctx context.Context, req *v1.FlowInstanceTerminateReq) (res *v1.FlowInstanceTerminateRes, err error) {
	return service.Flow().InstanceTerminate(ctx, req)
}

func (c *ControllerV1) InstanceUrge(ctx context.Context, req *v1.FlowInstanceUrgeReq) (res *v1.FlowInstanceUrgeRes, err error) {
	return service.Flow().InstanceUrge(ctx, req)
}
