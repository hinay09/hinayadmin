package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) GencodeDownload(ctx context.Context, req *v1.GencodeDownloadReq) (res *v1.GencodeDownloadRes, err error) {
	return service.Gencode().Download(ctx, req)
}
