/**
 * 菜单管理 API。
 */
import { useRequest } from '~/composables/useRequest'

export function useMenuApi() {
  const r = useRequest()
  const base = '/system/menus'
  return {
    list: (params?: any) => r.get(base, params),
    tree: () => r.get(`${base}/tree`),
    detail: (id: number) => r.get(`${base}/${id}`),
    create: (data: any) => r.post(base, data),
    update: (id: number, data: any) => r.put(`${base}/${id}`, data),
    remove: (id: number) => r.del(`${base}/${id}`),
  }
}
