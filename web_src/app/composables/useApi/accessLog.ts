/**
 * 接口日志 API (全量访问流水 + 慢接口标记)。
 */
import { useRequest } from '~/composables/useRequest'

export function useAccessLogApi() {
  const r = useRequest()
  const base = '/system/access-logs'
  return {
    list: (params: {
      keyword?: string
      username?: string
      method?: string
      slow?: number
      result?: string
      minDuration?: number
      startAt?: string
      endAt?: string
      page?: number
      pageSize?: number
    }) => r.get<{ list: any[]; total: number }>(base, params),
  }
}
