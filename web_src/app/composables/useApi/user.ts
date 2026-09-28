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
    /** 用户列表导出 (xlsx 下载) */
    export: (params?: { keyword?: string; status?: number }) => r.download(`${base}/export`, params),
    /** 用户导入模板下载 */
    importTemplate: () => r.download(`${base}/import-template`),
    /** 用户导入 (multipart 上传) */
    import: (file: File) => {
      const fd = new FormData()
      fd.append('file', file)
      return r.post<{ successCount: number; failCount: number; errors: string[] }>(`${base}/import`, fd)
    },
  }
}
