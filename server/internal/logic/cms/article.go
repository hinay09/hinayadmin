// article.go 文章业务逻辑 —— 状态机 (草稿/已发布/已下架) 与富文本正文的
// 持久引用策略在这里收口。
package cms

import (
	"context"
	"regexp"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	v1 "hinay.cn/admin/api/cms/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model/entity"
	"hinay.cn/admin/internal/storage"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/sanitizer"
	"hinay.cn/admin/utility/xerror"
)

type sCmsArticle struct{}

// articleListFields 列表/详情瘦身字段 (不含 content, 列表不出正文)。
const articleListFields = "id, category_id, title, cover, summary, tags, status, top, publish_at, created_at"

// List 文章分页列表 (不含正文; 置顶优先, 再按发布时间/ID 倒序)。
func (s *sCmsArticle) List(ctx context.Context, in *v1.CmsArticleListReq) (res *v1.CmsArticleListRes, err error) {
	q := dao.CmsArticle.Ctx(ctx).Where("deleted_at IS NULL")
	if in.Keyword != "" {
		q = q.Where("title LIKE ?", "%"+strings.TrimSpace(in.Keyword)+"%")
	}
	if in.CategoryId != nil {
		q = q.Where("category_id", *in.CategoryId)
	}
	if in.Status != nil {
		q = q.Where("status", *in.Status)
	}
	total, err := q.Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	var rows []*entity.CmsArticle
	if err = q.Fields(articleListFields).
		Order("top DESC, publish_at DESC, id DESC").Page(in.Page, in.PageSize).Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	names := categoryNames(ctx, rows...)
	list := make([]*v1.CmsArticleItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, articleItem(ctx, r, names[r.CategoryId]))
	}
	page := response.Page(list, int64(total), in.Page, in.PageSize)
	return (*v1.CmsArticleListRes)(&page), nil
}

// Detail 文章详情 (含正文): 图片引用出站重写 (s3:// → 预签名), 封面同。
func (s *sCmsArticle) Detail(ctx context.Context, in *v1.CmsArticleDetailReq) (res *v1.CmsArticleDetailRes, err error) {
	a, err := s.loadOne(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	return &v1.CmsArticleDetailRes{Detail: &v1.CmsArticleDetail{
		CmsArticleItem: *articleItem(ctx, a, ""),
		Content:        viewContentImgs(ctx, a.Content),
		CoverRef:       a.Cover,
	}}, nil
}

// Create 新增文章 (正文 XSS 白名单过滤 + 图片引用归一后入库; status=1 时直接发布)。
func (s *sCmsArticle) Create(ctx context.Context, in *v1.CmsArticleCreateReq) (res *v1.CmsArticleCreateRes, err error) {
	if err = s.checkCategory(ctx, in.CategoryId); err != nil {
		return nil, err
	}
	data := g.Map{
		"category_id": in.CategoryId,
		"title":       sanitizer.PlainText(strings.TrimSpace(in.Title)),
		"cover":       sanitizer.PlainText(in.Cover),
		"summary":     sanitizer.PlainText(in.Summary),
		"content":     normalizeContentImgs(sanitizer.RichText(in.Content)),
		"tags":        sanitizer.PlainText(strings.TrimSpace(in.Tags)),
		"status":      in.Status,
		"top":         in.Top,
	}
	if in.Status == v1.ArticleStatusOnline {
		data["publish_at"] = gtime.Now()
	}
	id, err := dao.CmsArticle.Ctx(ctx).Data(data).InsertAndGetId()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "新增文章失败")
	}
	return &v1.CmsArticleCreateRes{Id: uint64(id)}, nil
}

// Update 修改文章 (正文 XSS 白名单过滤 + 图片引用归一后入库; 状态由发布/下架动作管理)。
func (s *sCmsArticle) Update(ctx context.Context, in *v1.CmsArticleUpdateReq) (res *v1.CmsArticleUpdateRes, err error) {
	if err = s.checkCategory(ctx, in.CategoryId); err != nil {
		return nil, err
	}
	if _, err = dao.CmsArticle.Ctx(ctx).Where("id", in.Id).Where("deleted_at IS NULL").
		Data(g.Map{
			"category_id": in.CategoryId,
			"title":       sanitizer.PlainText(strings.TrimSpace(in.Title)),
			"cover":       sanitizer.PlainText(in.Cover),
			"summary":     sanitizer.PlainText(in.Summary),
			"content":     normalizeContentImgs(sanitizer.RichText(in.Content)),
			"tags":        sanitizer.PlainText(strings.TrimSpace(in.Tags)),
			"top":         in.Top,
		}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "修改文章失败")
	}
	return &v1.CmsArticleUpdateRes{}, nil
}

// Delete 删除文章 (软删)。
func (s *sCmsArticle) Delete(ctx context.Context, in *v1.CmsArticleDeleteReq) (res *v1.CmsArticleDeleteRes, err error) {
	if _, err = dao.CmsArticle.Ctx(ctx).Where("id", in.Id).Delete(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "删除文章失败")
	}
	return &v1.CmsArticleDeleteRes{}, nil
}

// Publish 发布文章: 草稿/已下架 → 已发布; 已发布幂等成功 (编辑页「保存并发布」
// 对已发布文章重入不再报错), 首次发布写 publish_at (重发保留原时间)。
func (s *sCmsArticle) Publish(ctx context.Context, in *v1.CmsArticlePublishReq) (res *v1.CmsArticlePublishRes, err error) {
	a, err := s.loadOne(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	if a.Status == v1.ArticleStatusOnline && a.PublishAt != nil {
		return &v1.CmsArticlePublishRes{}, nil // 已发布: 无变化幂等成功
	}
	data := g.Map{"status": v1.ArticleStatusOnline}
	if a.PublishAt == nil {
		data["publish_at"] = gtime.Now()
	}
	if _, err = dao.CmsArticle.Ctx(ctx).Where("id", a.Id).Data(data).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "发布失败")
	}
	return &v1.CmsArticlePublishRes{}, nil
}

// Offline 下架文章: 已发布 → 已下架 (内容保留, 可再发布)。
func (s *sCmsArticle) Offline(ctx context.Context, in *v1.CmsArticleOfflineReq) (res *v1.CmsArticleOfflineRes, err error) {
	a, err := s.loadOne(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	if a.Status != v1.ArticleStatusOnline {
		return nil, xerror.New(xerror.CodeParamInvalid, "仅已发布的文章可下架")
	}
	if _, err = dao.CmsArticle.Ctx(ctx).Where("id", a.Id).
		Data(g.Map{"status": v1.ArticleStatusOffline}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "下架失败")
	}
	return &v1.CmsArticleOfflineRes{}, nil
}

// ============================================================
// 内部工具
// ============================================================

// loadOne 按 ID 取未删除文章。
func (s *sCmsArticle) loadOne(ctx context.Context, id uint64) (*entity.CmsArticle, error) {
	var a *entity.CmsArticle
	if err := dao.CmsArticle.Ctx(ctx).Where("id", id).Where("deleted_at IS NULL").Scan(&a); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if a == nil {
		return nil, xerror.New(xerror.CodeNotFound, "文章不存在")
	}
	return a, nil
}

// checkCategory 栏目存在性校验 (0=未分类放行; 查询边界由租户注入器保证)。
func (s *sCmsArticle) checkCategory(ctx context.Context, categoryId uint64) error {
	if categoryId == 0 {
		return nil
	}
	cnt, err := dao.CmsCategory.Ctx(ctx).Where("id", categoryId).Where("deleted_at IS NULL").Count()
	if err != nil {
		return xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if cnt == 0 {
		return xerror.New(xerror.CodeParamInvalid, "栏目不存在或已被删除")
	}
	return nil
}

// articleItem 实体转条目 (封面持久引用出站重写, 同 avatar 模式)。
func articleItem(ctx context.Context, r *entity.CmsArticle, categoryName string) *v1.CmsArticleItem {
	return &v1.CmsArticleItem{
		Id: r.Id, CategoryId: r.CategoryId, CategoryName: categoryName,
		Title: r.Title, Cover: storage.ViewURL(ctx, r.Cover), Summary: r.Summary,
		Tags: r.Tags, Status: r.Status, Top: r.Top,
		PublishAt: r.PublishAt, CreatedAt: r.CreatedAt,
	}
}

// categoryNames 批量取栏目名 (列表展示; 查不到回退空串)。
func categoryNames(ctx context.Context, rows ...*entity.CmsArticle) map[uint64]string {
	out := map[uint64]string{}
	ids := make([]uint64, 0, len(rows))
	for _, r := range rows {
		if r.CategoryId > 0 {
			ids = append(ids, r.CategoryId)
		}
	}
	if len(ids) == 0 {
		return out
	}
	var cats []*entity.CmsCategory
	if err := dao.CmsCategory.Ctx(ctx).WhereIn("id", ids).Scan(&cats); err != nil {
		return out
	}
	for _, c := range cats {
		out[c.Id] = c.Name
	}
	return out
}

// ============================================================
// 富文本正文图片: 持久引用归一 (入库) / 出站重写 (回显)
// ============================================================

var (
	// mediaTagRe 匹配 <img ...> / <video ...> 开标签 (wangeditor 输出不换行, 非贪婪到 >)。
	mediaTagRe = regexp.MustCompile(`<(?:img|video)\b[^>]*>`)
	// mediaSrcRe / mediaDataRefRe 在单个标签内取属性值 (只认双引号形式, wangeditor 产物如此)。
	mediaSrcRe     = regexp.MustCompile(`\ssrc="([^"]*)"`)
	mediaDataRefRe = regexp.MustCompile(`\sdata-ref="([^"]*)"`)
)

// normalizeContentImgs 入库归一: 编辑器插图带 data-ref (上传响应 ref 持久引用),
// src 可能是当时展示用的预签名 URL (会过期) —— 统一重置为持久引用。
// 无 data-ref 的媒体 (外链图片等) 原样保留。
// 注意上游已过 sanitizer.RichText: s3:// 等非白名单 scheme 的 src 会被剥掉
// (data-ref 是自定义属性不受 scheme 校验, 保留), 这里从 data-ref 补回 src。
func normalizeContentImgs(content string) string {
	if !strings.Contains(content, "data-ref=") {
		return content
	}
	return mediaTagRe.ReplaceAllStringFunc(content, func(tag string) string {
		m := mediaDataRefRe.FindStringSubmatch(tag)
		if m == nil || m[1] == "" {
			return tag
		}
		ref := m[1]
		sm := mediaSrcRe.FindStringSubmatch(tag)
		if sm != nil && sm[1] == ref {
			return tag // 已是持久引用
		}
		if sm == nil {
			// 无 src (被过滤剥掉或本就缺失): 在开标签末尾补回持久引用。
			t := strings.TrimSuffix(strings.TrimSuffix(tag, ">"), "/")
			return t + ` src="` + ref + `">`
		}
		return mediaSrcRe.ReplaceAllLiteralString(tag, ` src="`+ref+`"`)
	})
}

// viewContentImgs 出站重写: src 为持久引用时经 storage.ViewURL 换展示 URL
// (本地存储 /upload/.. 原样, s3:// 换预签名)。
func viewContentImgs(ctx context.Context, content string) string {
	if !strings.Contains(content, "s3://") {
		return content
	}
	return mediaTagRe.ReplaceAllStringFunc(content, func(tag string) string {
		m := mediaSrcRe.FindStringSubmatch(tag)
		if m == nil || !strings.HasPrefix(m[1], "s3://") {
			return tag
		}
		return mediaSrcRe.ReplaceAllLiteralString(tag, ` src="`+storage.ViewURL(ctx, m[1])+`"`)
	})
}
