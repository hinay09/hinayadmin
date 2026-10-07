// Package flow 流程定义控制器 (薄透传)。
package flow

import (
	"context"

	"hinay.cn/admin/api/flow/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) DefinitionList(ctx context.Context, req *v1.FlowDefinitionListReq) (res *v1.FlowDefinitionListRes, err error) {
	return service.Flow().DefinitionList(ctx, req)
}

func (c *ControllerV1) DefinitionUsable(ctx context.Context, req *v1.FlowDefinitionUsableReq) (res *v1.FlowDefinitionUsableRes, err error) {
	return service.Flow().DefinitionUsable(ctx, req)
}

func (c *ControllerV1) DefinitionDetail(ctx context.Context, req *v1.FlowDefinitionDetailReq) (res *v1.FlowDefinitionDetailRes, err error) {
	return service.Flow().DefinitionDetail(ctx, req)
}

func (c *ControllerV1) DefinitionCreate(ctx context.Context, req *v1.FlowDefinitionCreateReq) (res *v1.FlowDefinitionCreateRes, err error) {
	return service.Flow().DefinitionCreate(ctx, req)
}

func (c *ControllerV1) DefinitionUpdate(ctx context.Context, req *v1.FlowDefinitionUpdateReq) (res *v1.FlowDefinitionUpdateRes, err error) {
	return service.Flow().DefinitionUpdate(ctx, req)
}

func (c *ControllerV1) DefinitionDelete(ctx context.Context, req *v1.FlowDefinitionDeleteReq) (res *v1.FlowDefinitionDeleteRes, err error) {
	return service.Flow().DefinitionDelete(ctx, req)
}

func (c *ControllerV1) DefinitionPublish(ctx context.Context, req *v1.FlowDefinitionPublishReq) (res *v1.FlowDefinitionPublishRes, err error) {
	return service.Flow().DefinitionPublish(ctx, req)
}

func (c *ControllerV1) DefinitionDisable(ctx context.Context, req *v1.FlowDefinitionDisableReq) (res *v1.FlowDefinitionDisableRes, err error) {
	return service.Flow().DefinitionDisable(ctx, req)
}
