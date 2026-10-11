// Package v1 内容管理 CMS 接口契约 —— 栏目 + 文章 (wangeditor 富文本)。
//
// 多租户: cms_category / cms_article 均为 A 类隔离表, 租户条件由驱动层注入器
// (ormfill/tenantorm.go) 管理, 业务代码不感知 tenant_id。
//
// 富文本图片持久化: 编辑器插图带 data-ref (上传响应 ref 持久引用) 属性,
// 入库前 normalizeContentImgs 把 img src 归一为持久引用; 出站 (详情) 时
// viewContentImgs 把 s3:// 引用经 storage.ViewURL 换成预签名 URL ——
// 同 sys_user.avatar "库内存引用、出口重写", 切换存储不腐蚀历史内容。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	"hinay.cn/admin/utility/response"
)

// ============================================================
// 栏目
// ============================================================

// CmsCategoryItem 栏目条目。
type CmsCategoryItem struct {
	Id     uint64 `json:"id"`
	Name   string `json:"name"   dc:"栏目名称"`
	Code   string `json:"code"   dc:"栏目编码(租户内唯一)"`
	Sort   int    `json:"sort"   dc:"排序(小在前)"`
	Status int    `json:"status" dc:"状态:1=启用,0=禁用"`
	Remark string `json:"remark"`
}

// CmsCategoryListReq 栏目分页列表。
type CmsCategoryListReq struct {
	g.Meta   `path:"/cms/categories" tags:"Cms" method:"get" summary:"栏目分页列表"`
	Keyword  string `json:"keyword"  in:"query" dc:"名称/编码模糊"`
	Status   *int   `json:"status"   in:"query" dc:"状态"`
	Page     int    `json:"page"     in:"query" d:"1"  dc:"页码"`
	PageSize int    `json:"pageSize" in:"query" d:"10" dc:"页大小"`
}

// CmsCategoryListRes 列表响应。
type CmsCategoryListRes response.PageResult

// CmsCategoryAllReq 启用栏目全量 (文章表单下拉)。
type CmsCategoryAllReq struct {
	g.Meta `path:"/cms/categories/all" tags:"Cms" method:"get" summary:"启用栏目全量"`
}

// CmsCategoryAllRes 全量响应。
type CmsCategoryAllRes struct {
	List []*CmsCategoryItem `json:"list"`
}

// CmsCategoryCreateReq 新增栏目。
type CmsCategoryCreateReq struct {
	g.Meta `path:"/cms/categories" tags:"Cms" method:"post" summary:"新增栏目"`
	Name   string `json:"name"   v:"required|length:1,64#请输入栏目名称|栏目名称长度 1-64"`
	Code   string `json:"code"   v:"required|regex:^[a-zA-Z0-9_-]{1,64}$#请输入栏目编码|编码为 1-64 位字母/数字/下划线/连字符"`
	Sort   int    `json:"sort"`
	Status int    `json:"status" v:"in:0,1#状态仅能为 0/1" d:"1"`
	Remark string `json:"remark" v:"max-length:200#备注最长 200 字"`
}

// CmsCategoryCreateRes 新增响应。
type CmsCategoryCreateRes struct {
	Id uint64 `json:"id"`
}

// CmsCategoryUpdateReq 修改栏目 (code 不可改 —— 唯一键即语义)。
type CmsCategoryUpdateReq struct {
	g.Meta `path:"/cms/categories/{id}" tags:"Cms" method:"put" summary:"修改栏目"`
	Id     uint64 `json:"id"     in:"path" v:"required"`
	Name   string `json:"name"   v:"required|length:1,64#请输入栏目名称|栏目名称长度 1-64"`
	Sort   int    `json:"sort"`
	Status int    `json:"status" v:"in:0,1#状态仅能为 0/1"`
	Remark string `json:"remark" v:"max-length:200#备注最长 200 字"`
}

// CmsCategoryUpdateRes 修改响应。
type CmsCategoryUpdateRes struct{}

// CmsCategoryDeleteReq 删除栏目 (栏目下有未删文章时拒绝)。
type CmsCategoryDeleteReq struct {
	g.Meta `path:"/cms/categories/{id}" tags:"Cms" method:"delete" summary:"删除栏目"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// CmsCategoryDeleteRes 删除响应。
type CmsCategoryDeleteRes struct{}

// ============================================================
// 文章
// ============================================================

// 文章状态编码。
const (
	ArticleStatusDraft   = 0 // 草稿
	ArticleStatusOnline  = 1 // 已发布
	ArticleStatusOffline = 2 // 已下架
)

// CmsArticleItem 文章列表条目 (不含正文 content —— 列表瘦身)。
type CmsArticleItem struct {
	Id           uint64      `json:"id"`
	CategoryId   uint64      `json:"categoryId"`
	CategoryName string      `json:"categoryName" dc:"栏目名(批量取回; 未分类为空)"`
	Title        string      `json:"title"`
	Cover        string      `json:"cover" dc:"封面(出口已重写为可展示URL)"`
	Summary      string      `json:"summary"`
	Tags         string      `json:"tags"`
	Status       int         `json:"status" dc:"状态:0=草稿,1=已发布,2=已下架"`
	Top          int         `json:"top"    dc:"置顶:1=是,0=否"`
	PublishAt    *gtime.Time `json:"publishAt"`
	CreatedAt    *gtime.Time `json:"createdAt"`
}

// CmsArticleDetail 文章详情 (含正文, 图片引用出站重写后可直接回显编辑器)。
type CmsArticleDetail struct {
	CmsArticleItem
	Content string `json:"content" dc:"正文HTML"`
	// CoverRef 封面持久引用 (表单回存用; cover 字段是出站展示URL, 预签名会过期不可回存)
	CoverRef string `json:"coverRef"`
}

// CmsArticleListReq 文章分页列表。
type CmsArticleListReq struct {
	g.Meta     `path:"/cms/articles" tags:"Cms" method:"get" summary:"文章分页列表"`
	Keyword    string  `json:"keyword"    in:"query" dc:"标题模糊"`
	CategoryId *uint64 `json:"categoryId" in:"query" dc:"栏目ID"`
	Status     *int    `json:"status"     in:"query" dc:"状态:0=草稿,1=已发布,2=已下架"`
	Page       int     `json:"page"       in:"query" d:"1"  dc:"页码"`
	PageSize   int     `json:"pageSize"   in:"query" d:"10" dc:"页大小"`
}

// CmsArticleListRes 列表响应。
type CmsArticleListRes response.PageResult

// CmsArticleCreateReq 新增文章 (默认草稿; status 可直接给 1 发布)。
type CmsArticleCreateReq struct {
	g.Meta     `path:"/cms/articles" tags:"Cms" method:"post" summary:"新增文章"`
	CategoryId uint64 `json:"categoryId" dc:"栏目ID, 0=未分类"`
	Title      string `json:"title"      v:"required|length:1,200#请输入文章标题|标题长度 1-200"`
	Cover      string `json:"cover"      v:"max-length:500#封面引用最长 500 字"`
	Summary    string `json:"summary"    v:"max-length:500#摘要最长 500 字"`
	Content    string `json:"content"`
	Tags       string `json:"tags"       v:"max-length:200#标签最长 200 字"`
	Status     int    `json:"status"     v:"in:0,1#新增状态仅能为 草稿(0)/发布(1)"`
	Top        int    `json:"top"        v:"in:0,1#置顶仅能为 0/1"`
}

// CmsArticleCreateRes 新增响应。
type CmsArticleCreateRes struct {
	Id uint64 `json:"id"`
}

// CmsArticleUpdateReq 修改文章 (status 不在此改 —— 发布/下架是独立动作)。
type CmsArticleUpdateReq struct {
	g.Meta     `path:"/cms/articles/{id}" tags:"Cms" method:"put" summary:"修改文章"`
	Id         uint64 `json:"id"         in:"path" v:"required"`
	CategoryId uint64 `json:"categoryId" dc:"栏目ID, 0=未分类"`
	Title      string `json:"title"      v:"required|length:1,200#请输入文章标题|标题长度 1-200"`
	Cover      string `json:"cover"      v:"max-length:500#封面引用最长 500 字"`
	Summary    string `json:"summary"    v:"max-length:500#摘要最长 500 字"`
	Content    string `json:"content"`
	Tags       string `json:"tags"       v:"max-length:200#标签最长 200 字"`
	Top        int    `json:"top"        v:"in:0,1#置顶仅能为 0/1"`
}

// CmsArticleUpdateRes 修改响应。
type CmsArticleUpdateRes struct{}

// CmsArticleDetailReq 文章详情 (含正文)。
type CmsArticleDetailReq struct {
	g.Meta `path:"/cms/articles/{id}" tags:"Cms" method:"get" summary:"文章详情"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// CmsArticleDetailRes 详情响应。
type CmsArticleDetailRes struct {
	Detail *CmsArticleDetail `json:"detail"`
}

// CmsArticleDeleteReq 删除文章 (软删)。
type CmsArticleDeleteReq struct {
	g.Meta `path:"/cms/articles/{id}" tags:"Cms" method:"delete" summary:"删除文章"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// CmsArticleDeleteRes 删除响应。
type CmsArticleDeleteRes struct{}

// CmsArticlePublishReq 发布文章 (草稿/已下架 → 已发布, 首次发布写 publish_at)。
type CmsArticlePublishReq struct {
	g.Meta `path:"/cms/articles/{id}/publish" tags:"Cms" method:"post" summary:"发布文章"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// CmsArticlePublishRes 发布响应。
type CmsArticlePublishRes struct{}

// CmsArticleOfflineReq 下架文章 (已发布 → 已下架)。
type CmsArticleOfflineReq struct {
	g.Meta `path:"/cms/articles/{id}/offline" tags:"Cms" method:"post" summary:"下架文章"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// CmsArticleOfflineRes 下架响应。
type CmsArticleOfflineRes struct{}
