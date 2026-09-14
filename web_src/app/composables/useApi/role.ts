/**
 * 角色管理 API (含菜单/API 授权)。
 */
import { useRequest } from '~/composables/useRequest'

export function useRoleApi() {
  const r = useRequest()
  const base = '/system/roles'
  return {
    list: (params: any) => r.get(base, params),
    all: () => r.get(`${base}/all`),
    detail: (id: number) => r.get(`${base}/${id}`),
    create: (data: any) => r.post(base, data),
    update: (id: number, data: any) => r.put(`${base}/${id}`, data),
    remove: (id: number) => r.del(`${base}/${id}`),
    assignMenus: (id: number, menuIds: number[]) =>
      r.put(`${base}/${id}/menus`, { menuIds }),
    getMenus: (id: number) =>
      r.get(`${base}/${id}/menus`),
    assignApis: (id: number, apis: Array<{ path: string; method: string }>) =>
      r.put(`${base}/${id}/apis`, { apis }),
    getApis: (id: number) =>
      r.get(`${base}/${id}/apis`),
  }
}
