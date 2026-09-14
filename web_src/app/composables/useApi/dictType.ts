/**
 * 字典类型管理 API。
 */
import { useRequest } from '~/composables/useRequest'

export function useDictTypeApi() {
  const r = useRequest()
  const base = '/system/dict-types'
  return {
    list: (params: { keyword?: string; page?: number; pageSize?: number }) =>
      r.get<{ list: any[]; total: number }>(base, params),
    all: () => r.get<{ list: any[] }>(`${base}/all`),
    create: (data: { typeCode: string; typeName: string; status?: number; remark?: string }) =>
      r.post<{ id: number }>(base, data),
    update: (id: number, data: { typeCode: string; typeName: string; status?: number; remark?: string }) =>
      r.put(`${base}/${id}`, data),
    remove: (id: number) => r.del(`${base}/${id}`),
  }
}
