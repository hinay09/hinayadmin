package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) DictDataList(ctx context.Context, req *v1.DictDataListReq) (res *v1.DictDataListRes, err error) {
	return service.Dict().ListByType(ctx, req)
}
