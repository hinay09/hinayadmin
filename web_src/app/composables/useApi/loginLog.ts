/**
 * 登录日志 API。
 */
import { useRequest } from '~/composables/useRequest'

export function useLoginLogApi() {
  const r = useRequest()
  const base = '/system/login-logs'
  return {
    list: (params: { username?: string; ip?: string; status?: string; startAt?: string; endAt?: string; page?: number; pageSize?: number }) =>
      r.get<{ list: any[]; total: number }>(base, params),
    delete: (id: number) => r.del(`${base}/${id}`),
  }
}
