package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) GencodeTables(ctx context.Context, req *v1.GencodeTablesReq) (res *v1.GencodeTablesRes, err error) {
	return service.Gencode().Tables(ctx, req)
}
