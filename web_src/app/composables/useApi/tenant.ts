/**
 * 租户管理 API (平台超管专属, docs/multi-tenant.md §7/§7.1)。
 */
import { useRequest } from '~/composables/useRequest'

export interface TenantItem {
  id: number
  name: string
  code: string
  contact: string
  plan: string
  packageId?: number
  packageName?: string
  quotaUsers: number
  quotaStorage: number
  expireAt?: string | null
  status: number
  remark?: string
  createdAt?: string
}

export interface TenantLiteItem {
  id: number
  code: string
  name: string
  status: number
}

export interface TenantCreateData {
  name: string
  code: string
  contact?: string
  plan?: string
  packageId?: number
  expireAt?: string
  remark?: string
  adminUsername: string
  adminNickname?: string
  adminPassword?: string
  menuIds: number[]
}

export function useTenantApi() {
  const r = useRequest()
  const base = '/system/tenants'
  return {
    list: (params: any) => r.get(base, params),
    listAll: () => r.get<{ list: TenantLiteItem[] }>(`${base}/all`),
    detail: (id: number) => r.get<{ tenant: TenantItem }>(`${base}/${id}`),
    create: (data: TenantCreateData) => r.post<{ id: number, adminUserId: number }>(base, data),
    update: (id: number, data: Partial<TenantCreateData> & { status?: number }) => r.put(`${base}/${id}`, data),    remove: (id: number) => r.del(`${base}/${id}`),
    getMenus: (id: number) => r.get<{ menuIds: number[] }>(`${base}/${id}/menus`),
    setMenus: (id: number, menuIds: number[]) => r.put(`${base}/${id}/menus`, { menuIds }),
    resetConfigs: (id: number) => r.post(`${base}/${id}/configs/reset`),
  }
}
