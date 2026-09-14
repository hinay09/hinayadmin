/**
 * 操作日志 API。
 */
import { useRequest } from '~/composables/useRequest'

export function useAuditLogApi() {
  const r = useRequest()
  const base = '/system/audit-logs'
  return {
    list: (params: { keyword?: string; action?: string; startAt?: string; endAt?: string; page?: number; pageSize?: number }) =>
      r.get<{ list: any[]; total: number }>(base, params),
  }
}
