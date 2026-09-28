package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) UserExport(ctx context.Context, req *v1.UserExportReq) (res *v1.UserExportRes, err error) {
	return service.User().Export(ctx, req)
}
