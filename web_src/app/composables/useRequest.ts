/**
 * 统一请求封装: 注入 Authorization, 处理后端 {code,message,data} 协议,
 * 支持 token 静默续期 (401 时自动 refresh, 并发请求队列化)。
 */
import { ElMessage } from 'element-plus'
import { useUserStore } from '~/stores/user'

export interface ApiResult<T = any> {
  code: number
  message: string
  data: T
}

/* -------- token 刷新队列 (模块级单例, 客户端全局共享) -------- */
let isRefreshing = false
let pendingQueue: Array<(token: string) => void> = []

function addPending(cb: (token: string) => void) {
  pendingQueue.push(cb)
}

function retryPending(newToken: string) {
  pendingQueue.forEach(cb => cb(newToken))
  pendingQueue = []
}

async function doRefresh(config: ReturnType<typeof useRuntimeConfig>, userStore: ReturnType<typeof useUserStore>): Promise<string> {
  const res = await $fetch<ApiResult<{ token: string; expireAt: number }>>(
    `${config.public.apiBase}/auth/refresh`,
    {
      method: 'POST',
      headers: { Authorization: `Bearer ${userStore.token}` },
    },
  )
  if (res.code !== 0) throw new Error(res.message || 'refresh failed')
  userStore.setToken(res.data.token, res.data.expireAt)
  return res.data.token
}

export function useRequest() {
  const config = useRuntimeConfig()
  const userStore = useUserStore()

  async function request<T = any>(
    url: string,
    options: any = {},
  ): Promise<T> {
    userStore.restore()
    const headers: Record<string, string> = {
      ...(options.headers || {}),
    }
    if (userStore.token) {
      headers['Authorization'] = `Bearer ${userStore.token}`
    }

    // 仅允许站内相对路径拼接 apiBase, 拒绝调用方传入完整外部 URL
    const fullUrl = `${config.public.apiBase}${url}`

    try {
      const res = await $fetch<ApiResult<T>>(fullUrl, {
        ...options,
        headers,
      })
      // GoFrame MiddlewareHandlerResponse 返回 {code,message,data}
      if (res && typeof res === 'object' && 'code' in res) {
        if (res.code === 0) {
          return (res.data ?? null) as T
        }
        ElMessage.error(res.message || '请求失败')
        throw new Error(res.message || `code=${res.code}`)
      }
      return res as unknown as T
    }
    catch (err: any) {
      const status = err?.response?.status
      const data = err?.response?._data

      // 401 且不是 refresh 接口自身 → 尝试静默续期
      if ((status === 401 || data?.code === 40100) && url !== '/auth/refresh' && !import.meta.server) {
        if (!isRefreshing) {
          isRefreshing = true
          try {
            const newToken = await doRefresh(config, userStore)
            // 重试当前请求
            const retryHeaders = { ...headers, Authorization: `Bearer ${newToken}` }
            const retryRes = await $fetch<ApiResult<T>>(fullUrl, { ...options, headers: retryHeaders })
            if (retryRes && typeof retryRes === 'object' && 'code' in retryRes) {
              if (retryRes.code === 0) {
                retryPending(newToken)
                return (retryRes.data ?? null) as T
              }
              throw new Error(retryRes.message || 'retry failed')
            }
            retryPending(newToken)
            return retryRes as unknown as T
          }
          catch (refreshErr) {
            // 续期失败: 清空队列并跳转登录
            pendingQueue.forEach(cb => cb(''))
            pendingQueue = []
            ElMessage.error('登录已过期, 请重新登录')
            userStore.reset()
            await navigateTo('/login')
            throw refreshErr
          }
          finally {
            isRefreshing = false
          }
        }
        else {
          // 已有 refresh 进行中, 将当前请求加入队列
          return new Promise<T>((resolve, reject) => {
            addPending(async (newToken: string) => {
              if (!newToken) {
                reject(new Error('refresh failed'))
                return
              }
              try {
                const retryHeaders = { ...headers, Authorization: `Bearer ${newToken}` }
                const retryRes = await $fetch<ApiResult<T>>(fullUrl, { ...options, headers: retryHeaders })
                if (retryRes && typeof retryRes === 'object' && 'code' in retryRes) {
                  if (retryRes.code === 0) {
                    resolve((retryRes.data ?? null) as T)
                    return
                  }
                  reject(new Error(retryRes.message || 'retry failed'))
                  return
                }
                resolve(retryRes as unknown as T)
              }
              catch (retryErr) {
                reject(retryErr)
              }
            })
          })
        }
      }

      if (status === 401 || data?.code === 40100) {
        ElMessage.error('登录已过期, 请重新登录')
        userStore.reset()
        if (import.meta.client) {
          await navigateTo('/login')
        }
      }
      else if (status === 403 || data?.code === 40300) {
        // 触发 Nuxt 全局错误页显示 403
        if (import.meta.client) {
          showError({ statusCode: 403, message: '无访问权限' })
        }
        else {
          ElMessage.error('无访问权限')
        }
      }
      else if (data?.message) {
        ElMessage.error(data.message)
      }
      else if (!err?.message?.startsWith('code=')) {
        ElMessage.error(err?.message || '网络错误')
      }
      throw err
    }
  }

  return {
    get: <T = any>(url: string, query?: any) =>
      request<T>(url, { method: 'GET', query }),
    post: <T = any>(url: string, body?: any) =>
      request<T>(url, { method: 'POST', body }),
    put: <T = any>(url: string, body?: any) =>
      request<T>(url, { method: 'PUT', body }),
    del: <T = any>(url: string) =>
      request<T>(url, { method: 'DELETE' }),
  }
}
