/**
 * 内容管理 CMS API (栏目 + 文章)。
 */
import { useRequest } from '~/composables/useRequest'

export interface CmsCategoryItem {
  id: number
  name: string
  /** 栏目编码 (租户内唯一, 创建后不可改) */
  code: string
  sort: number
  status: number
  remark: string
}

/** 文章状态: 0=草稿, 1=已发布, 2=已下架 */
export type CmsArticleStatus = 0 | 1 | 2

export interface CmsArticleItem {
  id: number
  categoryId: number
  categoryName?: string
  title: string
  /** 封面 (出口已重写为可展示 URL) */
  cover: string
  summary: string
  tags: string
  status: CmsArticleStatus
  top: number
  publishAt?: string
  createdAt?: string
}

export interface CmsArticleDetail extends CmsArticleItem {
  /** 正文 HTML (图片引用出站重写后可直接回显编辑器) */
  content: string
  /** 封面持久引用 (表单回存用; cover 字段是展示 URL, 预签名会过期不可回存) */
  coverRef: string
}

export interface CmsArticlePayload {
  categoryId: number
  title: string
  cover?: string
  summary?: string
  content?: string
  tags?: string
  top?: number
  /** 仅新增时可带: 1=直接发布 */
  status?: CmsArticleStatus
}

export function useCmsCategoryApi() {
  const r = useRequest()
  const base = '/cms/categories'
  return {
    list: (params: { keyword?: string, status?: number, page?: number, pageSize?: number }) =>
      r.get<{ list: CmsCategoryItem[], total: number }>(base, params),
    /** 启用栏目全量 (文章表单下拉) */
    all: () => r.get<{ list: CmsCategoryItem[] }>(`${base}/all`),
    create: (data: { name: string, code: string, sort?: number, status?: number, remark?: string }) =>
      r.post<{ id: number }>(base, data),
    /** code 不可改 */
    update: (id: number, data: { name: string, sort?: number, status?: number, remark?: string }) =>
      r.put(`${base}/${id}`, data),
    remove: (id: number) => r.del(`${base}/${id}`),
  }
}

export function useCmsArticleApi() {
  const r = useRequest()
  const base = '/cms/articles'
  return {
    list: (params: { keyword?: string, categoryId?: number, status?: number, page?: number, pageSize?: number }) =>
      r.get<{ list: CmsArticleItem[], total: number }>(base, params),
    detail: (id: number) => r.get<{ detail: CmsArticleDetail }>(`${base}/${id}`),
    create: (data: CmsArticlePayload) => r.post<{ id: number }>(base, data),
    /** 状态不在修改里动 —— 发布/下架是独立动作 */
    update: (id: number, data: Omit<CmsArticlePayload, 'status'>) => r.put(`${base}/${id}`, data),
    remove: (id: number) => r.del(`${base}/${id}`),
    publish: (id: number) => r.post(`${base}/${id}/publish`),
    offline: (id: number) => r.post(`${base}/${id}/offline`),
  }
}
