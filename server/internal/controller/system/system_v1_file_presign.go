package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

// FilePresign 获取预签名上传地址。
func (c *ControllerV1) FilePresign(ctx context.Context, req *v1.FilePresignReq) (res *v1.FilePresignRes, err error) {
	return service.File().Presign(ctx, req)
}

// FilePresignConfirm 预签名上传确认。
func (c *ControllerV1) FilePresignConfirm(ctx context.Context, req *v1.FilePresignConfirmReq) (res *v1.FilePresignConfirmRes, err error) {
	return service.File().PresignConfirm(ctx, req)
}
