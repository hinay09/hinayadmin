// 栏目控制器 (薄透传)。
package cms

import (
	"context"

	v1 "hinay.cn/admin/api/cms/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) CmsCategoryList(ctx context.Context, req *v1.CmsCategoryListReq) (res *v1.CmsCategoryListRes, err error) {
	return service.CmsCategory().List(ctx, req)
}

func (c *ControllerV1) CmsCategoryAll(ctx context.Context, req *v1.CmsCategoryAllReq) (res *v1.CmsCategoryAllRes, err error) {
	return service.CmsCategory().All(ctx, req)
}

func (c *ControllerV1) CmsCategoryCreate(ctx context.Context, req *v1.CmsCategoryCreateReq) (res *v1.CmsCategoryCreateRes, err error) {
	return service.CmsCategory().Create(ctx, req)
}

func (c *ControllerV1) CmsCategoryUpdate(ctx context.Context, req *v1.CmsCategoryUpdateReq) (res *v1.CmsCategoryUpdateRes, err error) {
	return service.CmsCategory().Update(ctx, req)
}

func (c *ControllerV1) CmsCategoryDelete(ctx context.Context, req *v1.CmsCategoryDeleteReq) (res *v1.CmsCategoryDeleteRes, err error) {
	return service.CmsCategory().Delete(ctx, req)
}
