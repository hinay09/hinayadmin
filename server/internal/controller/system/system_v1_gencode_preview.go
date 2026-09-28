package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) GencodePreview(ctx context.Context, req *v1.GencodePreviewReq) (res *v1.GencodePreviewRes, err error) {
	return service.Gencode().Preview(ctx, req)
}
