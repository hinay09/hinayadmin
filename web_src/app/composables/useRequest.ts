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
/** 已确认失效的 token (refresh 失败时记录): 同一 token 的后续 401 不再反复
 *  refresh/弹错 —— 否则消息铃铛等轮询组件在登出后每个周期都会制造一次
 *  401 → refresh → 401 → "登录已过期" 弹窗的死循环。换新 token 登录后自动解除。 */
let deadToken: string | null = null

function addPending(cb: (token: string) => void) {
  pendingQueue.push(cb)
}

function retryPending(newToken: string) {
  pendingQueue.forEach(cb => cb(newToken))
  pendingQueue = []
}

/** 登出流程先行调用: 标记当前 token 即将失效, 并发在途请求的 401 静默处理, 不再弹错。 */
export function markSessionDead(token: string) {
  if (token) deadToken = token
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
  deadToken = null // 续签成功即建立新会话, 解除失效闩锁
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
    // silent: 业务失败时不自动弹错误提示, 由调用方汇总展示 (批量循环调用场景)
    const { silent, ...fetchOptions } = options
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
        ...fetchOptions,
        headers,
      })
      // GoFrame MiddlewareHandlerResponse 返回 {code,message,data}
      if (res && typeof res === 'object' && 'code' in res) {
        if (res.code === 0) {
          return (res.data ?? null) as T
        }
        if (!silent) ElMessage.error(res.message || '请求失败')
        const bizErr: any = new Error(res.message || `code=${res.code}`)
        bizErr.code = res.code
        throw bizErr
      }
      return res as unknown as T
    }
    catch (err: any) {
      const status = err?.response?.status
      const data = err?.response?._data

      // 401 且不是 refresh 接口自身 → 尝试静默续期
      if ((status === 401 || data?.code === 40100) && url !== '/auth/refresh' && !import.meta.server) {
        // 会话已死(无 token / token 已确认失效 / 登出流程中): 静默回登录页,
        // 不再 refresh、不再重复弹"登录已过期" —— 切断轮询组件的 401 循环
        const curToken = userStore.token
        if (!curToken || curToken === deadToken) {
          userStore.reset()
          await navigateTo('/login')
          throw err
        }
        if (!isRefreshing) {
          isRefreshing = true
          try {
            const newToken = await doRefresh(config, userStore)
            // 重试当前请求
            const retryHeaders = { ...headers, Authorization: `Bearer ${newToken}` }
            const retryRes = await $fetch<ApiResult<T>>(fullUrl, { ...fetchOptions, headers: retryHeaders })
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
            // 续期失败: 记录失效 token(后续同 token 的 401 静默), 清空队列并跳转登录
            deadToken = curToken
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
                const retryRes = await $fetch<ApiResult<T>>(fullUrl, { ...fetchOptions, headers: retryHeaders })
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
      else if (data?.code === 42800 || status === 428 || err?.code === 42800) {
        // 强制改密 (HTTP 428 Precondition Required): 密码被管理员重置/导入或已过期,
        // 同步标志并锁定到个人中心改密页。后台轮询会反复命中, 已在改密页时静默去重。
        userStore.mustChangePwd = true
        if (import.meta.client && !window.location.pathname.startsWith('/profile')) {
          ElMessage.warning(data?.message || '密码已过期或被重置, 请先修改密码')
          await navigateTo({
            path: '/profile',
            query: { tab: 'password', redirect: window.location.pathname + window.location.search },
          })
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
        if (!silent) ElMessage.error(data.message)
      }
      else if (!err?.message?.startsWith('code=')) {
        if (!silent) ElMessage.error(err?.message || '网络错误')
      }
      throw err
    }
  }

  /**
   * 文件下载 (Excel 导出/模板等): 携带 Authorization 请求二进制流并触发浏览器保存。
   * 从 Content-Disposition 解析文件名 (支持 filename*=UTF-8'')。
   */
  async function download(url: string, query?: Record<string, any>): Promise<void> {
    userStore.restore()
    const fullUrl = new URL(`${config.public.apiBase}${url}`, window.location.origin)
    if (query) {
      for (const [k, v] of Object.entries(query)) {
        if (v !== undefined && v !== null && v !== '') {
          fullUrl.searchParams.set(k, String(v))
        }
      }
    }
    const res = await $fetch.raw(fullUrl.toString(), {
      responseType: 'blob',
      headers: userStore.token ? { Authorization: `Bearer ${userStore.token}` } : {},
    })
    if (res.status !== 200) {
      ElMessage.error('下载失败')
      return
    }
    const disposition = res.headers.get('content-disposition') || ''
    let filename = 'download.xlsx'
    const utf8Match = disposition.match(/filename\*=UTF-8''([^;]+)/)
    if (utf8Match) {
      filename = decodeURIComponent(utf8Match[1])
    }
    else if (disposition.includes('filename=')) {
      filename = decodeURIComponent(disposition.split('filename=')[1]?.replace(/"/g, '') || filename)
    }
    const blob = res._data as Blob
    const link = document.createElement('a')
    link.href = URL.createObjectURL(blob)
    link.download = filename
    link.click()
    URL.revokeObjectURL(link.href)
  }

  return {
    get: <T = any>(url: string, query?: any, opts?: { silent?: boolean }) =>
      request<T>(url, { method: 'GET', query, silent: opts?.silent }),
    post: <T = any>(url: string, body?: any, opts?: { silent?: boolean }) =>
      request<T>(url, { method: 'POST', body, silent: opts?.silent }),
    put: <T = any>(url: string, body?: any, opts?: { silent?: boolean }) =>
      request<T>(url, { method: 'PUT', body, silent: opts?.silent }),
    del: <T = any>(url: string, query?: any, opts?: { silent?: boolean }) =>
      request<T>(url, { method: 'DELETE', query, silent: opts?.silent }),
    download,
  }
}
