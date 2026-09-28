package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) GencodeColumns(ctx context.Context, req *v1.GencodeColumnsReq) (res *v1.GencodeColumnsRes, err error) {
	return service.Gencode().Columns(ctx, req)
}
