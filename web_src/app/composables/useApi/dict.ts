/**
 * 字典数据管理 API。
 */
import { useRequest } from '~/composables/useRequest'

export function useDictApi() {
  const r = useRequest()
  const base = '/system/dict-types'
  return {
    listByType: (typeId: number, params?: { keyword?: string; page?: number; pageSize?: number }) =>
      r.get<{ list: any[]; total: number }>(`${base}/${typeId}/items`, params),
    create: (typeId: number, data: { dictLabel: string; dictValue?: string; sort?: number; status?: number; remark?: string }) =>
      r.post<{ id: number }>(`${base}/${typeId}/items`, data),
    update: (typeId: number, id: number, data: { dictLabel: string; dictValue?: string; sort?: number; status?: number; remark?: string }) =>
      r.put(`${base}/${typeId}/items/${id}`, data),
    remove: (typeId: number, id: number) => r.del(`${base}/${typeId}/items/${id}`),
    sort: (typeId: number, items: Array<{ id: number; sort: number }>) =>
      r.put(`${base}/${typeId}/items/sort`, { items }),
    all: () => r.get<{ list: any }>('/system/dicts/all'),
  }
}
