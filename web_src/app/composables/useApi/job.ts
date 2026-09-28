/**
 * 定时任务 API。
 */
import { useRequest } from '~/composables/useRequest'

export function useJobApi() {
  const r = useRequest()
  const base = '/system/jobs'
  return {
    list: (params: { keyword?: string; page?: number; pageSize?: number }) =>
      r.get<{ list: any[]; total: number }>(base, params),
    create: (data: { name: string; handler: string; cronExpr: string; params?: string; remark?: string }) =>
      r.post(base, data),
    update: (id: number, data: { name: string; handler: string; cronExpr: string; params?: string; remark?: string }) =>
      r.put(`${base}/${id}`, data),
    delete: (id: number) => r.del(`${base}/${id}`),
    changeStatus: (id: number, status: number) =>
      r.put(`${base}/${id}/status`, { status }),
    run: (id: number) => r.post(`${base}/${id}/run`),
    logs: (params: { jobId?: number; status?: string; startAt?: string; endAt?: string; page?: number; pageSize?: number }) =>
      r.get<{ list: any[]; total: number }>(`${base}/logs`, params),
    handlers: () => r.get<{ handlers: string[] }>(`${base}/handlers`),
  }
}
