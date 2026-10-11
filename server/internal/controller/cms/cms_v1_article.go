// 文章控制器 (薄透传)。
package cms

import (
	"context"

	v1 "hinay.cn/admin/api/cms/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) CmsArticleList(ctx context.Context, req *v1.CmsArticleListReq) (res *v1.CmsArticleListRes, err error) {
	return service.CmsArticle().List(ctx, req)
}

func (c *ControllerV1) CmsArticleDetail(ctx context.Context, req *v1.CmsArticleDetailReq) (res *v1.CmsArticleDetailRes, err error) {
	return service.CmsArticle().Detail(ctx, req)
}

func (c *ControllerV1) CmsArticleCreate(ctx context.Context, req *v1.CmsArticleCreateReq) (res *v1.CmsArticleCreateRes, err error) {
	return service.CmsArticle().Create(ctx, req)
}

func (c *ControllerV1) CmsArticleUpdate(ctx context.Context, req *v1.CmsArticleUpdateReq) (res *v1.CmsArticleUpdateRes, err error) {
	return service.CmsArticle().Update(ctx, req)
}

func (c *ControllerV1) CmsArticleDelete(ctx context.Context, req *v1.CmsArticleDeleteReq) (res *v1.CmsArticleDeleteRes, err error) {
	return service.CmsArticle().Delete(ctx, req)
}

func (c *ControllerV1) CmsArticlePublish(ctx context.Context, req *v1.CmsArticlePublishReq) (res *v1.CmsArticlePublishRes, err error) {
	return service.CmsArticle().Publish(ctx, req)
}

func (c *ControllerV1) CmsArticleOffline(ctx context.Context, req *v1.CmsArticleOfflineReq) (res *v1.CmsArticleOfflineRes, err error) {
	return service.CmsArticle().Offline(ctx, req)
}
