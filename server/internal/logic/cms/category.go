// Package cms 内容管理 —— 栏目 + 文章 (wangeditor 富文本)。
package cms

import (
	"context"
	"strconv"
	"strings"

	"github.com/gogf/gf/v2/frame/g"

	v1 "hinay.cn/admin/api/cms/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model/entity"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/xerror"
)

func init() {
	service.RegisterCmsCategory(&sCmsCategory{})
	service.RegisterCmsArticle(&sCmsArticle{})
}

// ============================================================
// 栏目
// ============================================================

type sCmsCategory struct{}

// categoryItem 实体转条目。
func categoryItem(r *entity.CmsCategory) *v1.CmsCategoryItem {
	return &v1.CmsCategoryItem{
		Id: r.Id, Name: r.Name, Code: r.Code, Sort: r.Sort, Status: r.Status, Remark: r.Remark,
	}
}

// List 栏目分页列表。
func (s *sCmsCategory) List(ctx context.Context, in *v1.CmsCategoryListReq) (res *v1.CmsCategoryListRes, err error) {
	q := dao.CmsCategory.Ctx(ctx).Where("deleted_at IS NULL")
	if in.Keyword != "" {
		kw := "%" + strings.TrimSpace(in.Keyword) + "%"
		q = q.Where("(name LIKE ? OR code LIKE ?)", kw, kw)
	}
	if in.Status != nil {
		q = q.Where("status", *in.Status)
	}
	total, err := q.Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	var rows []*entity.CmsCategory
	if err = q.Order("sort ASC, id ASC").Page(in.Page, in.PageSize).Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	list := make([]*v1.CmsCategoryItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, categoryItem(r))
	}
	page := response.Page(list, int64(total), in.Page, in.PageSize)
	return (*v1.CmsCategoryListRes)(&page), nil
}

// All 启用栏目全量 (文章表单下拉)。
// 收窄到数据租户: 平台超管未切换时取平台租户栏目, 同 posts/all。
func (s *sCmsCategory) All(ctx context.Context, in *v1.CmsCategoryAllReq) (res *v1.CmsCategoryAllRes, err error) {
	var rows []*entity.CmsCategory
	if err = dao.CmsCategory.Ctx(contextx.ScopedToDataTenant(ctx)).
		Where("status", 1).Where("deleted_at IS NULL").
		Order("sort ASC, id ASC").Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	list := make([]*v1.CmsCategoryItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, categoryItem(r))
	}
	return &v1.CmsCategoryAllRes{List: list}, nil
}

// Create 新增栏目 (code 租户内唯一 —— uk 唯一键兜底, 预检给友好报错)。
func (s *sCmsCategory) Create(ctx context.Context, in *v1.CmsCategoryCreateReq) (res *v1.CmsCategoryCreateRes, err error) {
	if s.codeExists(ctx, in.Code, 0) {
		return nil, xerror.New(xerror.CodeParamInvalid, "栏目编码已存在")
	}
	id, err := dao.CmsCategory.Ctx(ctx).Data(g.Map{
		"name":   strings.TrimSpace(in.Name),
		"code":   in.Code,
		"sort":   in.Sort,
		"status": in.Status,
		"remark": in.Remark,
	}).InsertAndGetId()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "新增栏目失败")
	}
	return &v1.CmsCategoryCreateRes{Id: uint64(id)}, nil
}

// Update 修改栏目 (code 不可改: 唯一键即语义)。
func (s *sCmsCategory) Update(ctx context.Context, in *v1.CmsCategoryUpdateReq) (res *v1.CmsCategoryUpdateRes, err error) {
	if _, err = dao.CmsCategory.Ctx(ctx).Where("id", in.Id).Where("deleted_at IS NULL").
		Data(g.Map{
			"name":   strings.TrimSpace(in.Name),
			"sort":   in.Sort,
			"status": in.Status,
			"remark": in.Remark,
		}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "修改栏目失败")
	}
	return &v1.CmsCategoryUpdateRes{}, nil
}

// Delete 删除栏目: 栏目下还有未删文章时拒绝 (强制运营先移走文章, 避免文章悬空)。
func (s *sCmsCategory) Delete(ctx context.Context, in *v1.CmsCategoryDeleteReq) (res *v1.CmsCategoryDeleteRes, err error) {
	cnt, err := dao.CmsArticle.Ctx(ctx).Where("category_id", in.Id).Where("deleted_at IS NULL").Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if cnt > 0 {
		return nil, xerror.New(xerror.CodeParamInvalid, "栏目下还有 "+strconv.Itoa(cnt)+" 篇文章, 请先移除或转移后再删除")
	}
	if _, err = dao.CmsCategory.Ctx(ctx).Where("id", in.Id).Delete(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "删除栏目失败")
	}
	return &v1.CmsCategoryDeleteRes{}, nil
}

// codeExists 编码唯一性预检 (排除自身); 租户内唯一由注入器保证查询边界。
func (s *sCmsCategory) codeExists(ctx context.Context, code string, excludeId uint64) bool {
	if code == "" {
		return false
	}
	cnt, err := dao.CmsCategory.Ctx(ctx).Where("code", code).
		Where("deleted_at IS NULL").WhereNot("id", excludeId).Count()
	return err == nil && cnt > 0
}
