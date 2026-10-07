// Package flow 设计器选项控制器 (薄透传)。
package flow

import (
	"context"

	"hinay.cn/admin/api/flow/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) DesignerOptions(ctx context.Context, req *v1.FlowDesignerOptionsReq) (res *v1.FlowDesignerOptionsRes, err error) {
	return service.Flow().DesignerOptions(ctx, req)
}
