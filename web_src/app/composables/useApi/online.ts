/**
 * 在线用户 API。
 */
import { useRequest } from '~/composables/useRequest'

export function useOnlineApi() {
  const r = useRequest()
  const base = '/system/online'
  return {
    list: (params: { username?: string; page?: number; pageSize?: number }) =>
      r.get<{ list: any[]; total: number }>(base, params),
    kick: (sessionId: string) => r.del(`${base}/${sessionId}`),
  }
}
