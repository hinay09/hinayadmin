/**
 * API 资源登记管理 API。
 */
import { useRequest } from '~/composables/useRequest'

export function useApiResourceApi() {
  const r = useRequest()
  const base = '/system/apis'
  return {
    list: (params?: { groupName?: string; path?: string; page?: number; pageSize?: number }) =>
      r.get<{ list: any[]; total: number }>(base, params),
    all: () => r.get<{ list: any[] }>(`${base}/all`),
    create: (data: { path: string; method: string; groupName?: string; description?: string }) =>
      r.post(base, data),
    update: (id: number, data: { path: string; method: string; groupName?: string; description?: string }) =>
      r.put(`${base}/${id}`, data),
    remove: (id: number) =>
      r.del(`${base}/${id}`),
  }
}
