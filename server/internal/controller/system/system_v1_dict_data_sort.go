package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) DictDataSort(ctx context.Context, req *v1.DictDataSortReq) (res *v1.DictDataSortRes, err error) {
	return service.Dict().Sort(ctx, req)
}
