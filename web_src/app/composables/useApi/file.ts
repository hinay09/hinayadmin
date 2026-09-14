/**
 * 文件管理 API。
 */
import { useRequest } from '~/composables/useRequest'

export function useFileApi() {
  const r = useRequest()
  const base = '/system/files'
  return {
    list: (params: { keyword?: string; mimeType?: string; page?: number; pageSize?: number }) =>
      r.get<{ list: any[]; total: number }>(base, params),
    upload: (file: File) => {
      const formData = new FormData()
      formData.append('file', file)
      return r.post<{ id: number; name: string; originalName: string; url: string; size: number; extension: string }>(
        `${base}/upload`, formData,
      )
    },
    remove: (id: number) => r.del(`${base}/${id}`),
  }
}
