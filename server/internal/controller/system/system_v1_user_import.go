package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) UserImport(ctx context.Context, req *v1.UserImportReq) (res *v1.UserImportRes, err error) {
	return service.User().Import(ctx, req)
}
