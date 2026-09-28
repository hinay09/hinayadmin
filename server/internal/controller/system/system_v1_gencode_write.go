package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) GencodeWrite(ctx context.Context, req *v1.GencodeWriteReq) (res *v1.GencodeWriteRes, err error) {
	return service.Gencode().Write(ctx, req)
}
