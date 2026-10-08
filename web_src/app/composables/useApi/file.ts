/**
 * 文件管理 API。
 *
 * 上传默认走预签名直传 (S3 兼容存储模式): 先取预签名 PUT 地址, 文件直传
 * 对象存储不经过应用服务器, 再调确认接口落库; 本地存储 (mode=server) 或
 * 预签名链路不可用时, 自动回落服务端中转上传。
 */
import { useRequest } from '~/composables/useRequest'

interface UploadResult {
  id: number
  name: string
  originalName: string
  url: string
  size: number
  extension: string
}

interface PresignResult {
  mode: 'presign' | 'server'
  uploadUrl?: string
  method?: string
  key?: string
  contentType?: string
  expireSec?: number
}

export function useFileApi() {
  const r = useRequest()
  const base = '/system/files'

  // 服务端中转上传 (multipart 走应用服务器)
  function uploadViaServer(file: File): Promise<UploadResult> {
    const formData = new FormData()
    formData.append('file', file)
    return r.post<UploadResult>(`${base}/upload`, formData)
  }

  return {
    list: (params: { keyword?: string; mimeType?: string; page?: number; pageSize?: number }) =>
      r.get<{ list: any[]; total: number }>(base, params),

    upload: async (file: File): Promise<UploadResult> => {
      try {
        const pre = await r.post<PresignResult>(`${base}/presign`, {
          originalName: file.name,
          contentType: file.type || 'application/octet-stream',
          size: file.size,
        })
        if (pre?.mode === 'presign' && pre.uploadUrl && pre.key) {
          // 直传对象存储: 预签名地址自带鉴权, 不携带 Authorization;
          // Content-Type 参与了签名, 必须原样携带
          await $fetch(pre.uploadUrl, {
            method: pre.method || 'PUT',
            body: file,
            headers: { 'Content-Type': pre.contentType || file.type || 'application/octet-stream' },
          })
          return await r.post<UploadResult>(`${base}/presign/confirm`, {
            key: pre.key,
            originalName: file.name,
            contentType: pre.contentType,
          })
        }
      }
      catch {
        // 预签名不可用 (本地存储/权限/CORS): 回落服务端中转
      }
      return uploadViaServer(file)
    },

    remove: (id: number) => r.del(`${base}/${id}`),
  }
}
