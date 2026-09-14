/**
 * 系统用户管理 API。
 */
import { useRequest } from '~/composables/useRequest'

export function useUserApi() {
  const r = useRequest()
  const base = '/system/users'
  return {
    list: (params: any) => r.get(base, params),
    detail: (id: number) => r.get(`${base}/${id}`),
    create: (data: any) => r.post(base, data),
    update: (id: number, data: any) => r.put(`${base}/${id}`, data),
    remove: (id: number) => r.del(`${base}/${id}`),
    resetPwd: (id: number, password: string) =>
      r.put(`${base}/${id}/password`, { password }),
  }
}
