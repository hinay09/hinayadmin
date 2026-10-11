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
	ICmsCategory interface {
		// List 栏目分页列表。
		List(ctx context.Context, req *v1.CmsCategoryListReq) (res *v1.CmsCategoryListRes, err error)
		// All 启用栏目全量 (文章表单下拉)。
		All(ctx context.Context, req *v1.CmsCategoryAllReq) (res *v1.CmsCategoryAllRes, err error)
		// Create 新增栏目 (code 租户内唯一)。
		Create(ctx context.Context, req *v1.CmsCategoryCreateReq) (res *v1.CmsCategoryCreateRes, err error)
		// Update 修改栏目 (code 不可改)。
		Update(ctx context.Context, req *v1.CmsCategoryUpdateReq) (res *v1.CmsCategoryUpdateRes, err error)
		// Delete 删除栏目 (栏目下有未删文章时拒绝)。
		Delete(ctx context.Context, req *v1.CmsCategoryDeleteReq) (res *v1.CmsCategoryDeleteRes, err error)
	}
)

var (
	localCmsCategory ICmsCategory
)

func CmsCategory() ICmsCategory {
	if localCmsCategory == nil {
		panic("implement not found for interface ICmsCategory, forgot register?")
	}
	return localCmsCategory
}

func RegisterCmsCategory(i ICmsCategory) {
	localCmsCategory = i
}
