/**
 * 租户套餐 API (平台超管专属, docs/multi-tenant.md §7/§7.2)。
 */
import { useRequest } from '~/composables/useRequest'

export interface TenantPackageItem {
  id: number
  name: string
  menuCount: number
  tenantCount: number
  status: number
  remark?: string
  createdAt?: string
}

/** 启用套餐轻量行 (含授权集模板, 下拉预填用)。 */
export interface TenantPackageLite {
  id: number
  name: string
  menuIds: number[]
}

export interface TenantPackageDetail extends TenantPackageItem {
  menuIds: number[]
  updatedAt?: string
}

export interface TenantPackageSaveData {
  name: string
  menuIds: number[]
  status?: number
  remark?: string
}

export function useTenantPackageApi() {
  const r = useRequest()
  const base = '/system/tenant-packages'
  return {
    list: (params: any) => r.get(base, params),
    listAll: () => r.get<{ list: TenantPackageLite[] }>(`${base}/all`),
    detail: (id: number) => r.get<{ package: TenantPackageDetail }>(`${base}/${id}`),
    create: (data: TenantPackageSaveData) => r.post<{ id: number }>(base, data),
    update: (id: number, data: TenantPackageSaveData) => r.put(`${base}/${id}`, data),
    remove: (id: number) => r.del(`${base}/${id}`),
    sync: (id: number) => r.post<{ count: number }>(`${base}/${id}/sync`),
  }
}
