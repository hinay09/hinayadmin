/**
 * 组织机构管理 API。
 */
import { useRequest } from '~/composables/useRequest'

export function useOrgApi() {
  const r = useRequest()
  const base = '/system/orgs'
  return {
    tree: () => r.get(`${base}/tree`),
    list: (params?: any) => r.get(base, params),
    detail: (id: number) => r.get(`${base}/${id}`),
    create: (data: any) => r.post(base, data),
    update: (id: number, data: any) => r.put(`${base}/${id}`, data),
    remove: (id: number) => r.del(`${base}/${id}`),
  }
}
