/**
 * 全局配置管理 API。
 */
import { useRequest } from '~/composables/useRequest'

export interface ConfigItem {
  id: number
  configKey: string
  configValue: string
  configType: number
  name: string
  remark: string
  status: number
  sort: number
  createdAt: string
  updatedAt: string
}

export function useConfigApi() {
  const r = useRequest()
  const base = '/system/configs'
  return {
    list: (params: { keyword?: string; configType?: number; page?: number; pageSize?: number }) =>
      r.get<{ list: ConfigItem[]; total: number }>(base, params),
    all: () => r.get<{ list: Record<string, { value: string; type: number; name: string }> }>(`${base}/all`),
    create: (data: { configKey: string; configValue?: string; configType?: number; name: string; remark?: string; status?: number; sort?: number }) =>
      r.post<{ id: number }>(base, data),
    update: (id: number, data: { configKey: string; configValue?: string; configType?: number; name: string; remark?: string; status?: number; sort?: number }) =>
      r.put(`${base}/${id}`, data),
    remove: (id: number) => r.del(`${base}/${id}`),
  }
}
