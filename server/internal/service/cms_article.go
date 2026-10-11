// ================================================================================
// This code was generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish to be manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	v1 "hinay.cn/admin/api/cms/v1"
)

type (
	ICmsArticle interface {
		// List 文章分页列表 (不含正文)。
		List(ctx context.Context, req *v1.CmsArticleListReq) (res *v1.CmsArticleListRes, err error)
		// Detail 文章详情 (含正文, 图片引用出站重写)。
		Detail(ctx context.Context, req *v1.CmsArticleDetailReq) (res *v1.CmsArticleDetailRes, err error)
		// Create 新增文章 (默认草稿)。
		Create(ctx context.Context, req *v1.CmsArticleCreateReq) (res *v1.CmsArticleCreateRes, err error)
		// Update 修改文章 (状态由发布/下架动作管理)。
		Update(ctx context.Context, req *v1.CmsArticleUpdateReq) (res *v1.CmsArticleUpdateRes, err error)
		// Delete 删除文章 (软删)。
		Delete(ctx context.Context, req *v1.CmsArticleDeleteReq) (res *v1.CmsArticleDeleteRes, err error)
		// Publish 发布文章 (草稿/已下架 → 已发布, 首次发布写 publish_at)。
		Publish(ctx context.Context, req *v1.CmsArticlePublishReq) (res *v1.CmsArticlePublishRes, err error)
		// Offline 下架文章 (已发布 → 已下架)。
		Offline(ctx context.Context, req *v1.CmsArticleOfflineReq) (res *v1.CmsArticleOfflineRes, err error)
	}
)

var (
	localCmsArticle ICmsArticle
)

func CmsArticle() ICmsArticle {
	if localCmsArticle == nil {
		panic("implement not found for interface ICmsArticle, forgot register?")
	}
	return localCmsArticle
}

func RegisterCmsArticle(i ICmsArticle) {
	localCmsArticle = i
}
