/**
 * 用户会话 store: token / userInfo / 菜单 / 权限标识
 *
 * token 持久化在 cookie(而非 localStorage): cookie 在 SSR 阶段可读,
 * 刷新页面时服务端路由守卫即可识别登录态, 避免先跳 /login 再弹回的闪跳。
 * cookie 由前端 JS 读写(非 HttpOnly), 仍需配合 Authorization 头使用,
 * API 侧不认 cookie 鉴权, 因此不引入 CSRF 面。
 */
import { defineStore } from 'pinia'

export interface LoginUser {
  userId: number
  username: string
  nickname: string
  avatar: string
  email?: string
  phone?: string
  roles: string[]
}

export interface MenuNode {
  id: number
  parentId: number
  name: string
  type: number
  path: string
  component: string
  icon: string
  permission: string
  children?: MenuNode[]
}

const TOKEN_COOKIE = 'hinay_token'
const EXPIRE_COOKIE = 'hinay_token_expire'

/** 站内安全跳转: 仅允许以单个 / 开头的路径, 拒绝 //host、/\host 与外部 URL */
export function safeRedirect(raw: string | undefined | null): string {
  if (raw && raw.startsWith('/') && !raw.startsWith('//') && !raw.startsWith('/\\')) {
    return raw
  }
  return '/'
}

export const useUserStore = defineStore('user', {
  state: () => ({
    token: '' as string,
    expireAt: 0 as number,
    userInfo: null as LoginUser | null,
    menus: [] as MenuNode[],
    permissions: [] as string[],
    menusLoaded: false,
  }),

  getters: {
    isLogin: (s) => !!s.token,
    roles: (s) => s.userInfo?.roles ?? [],
    isAdmin: (s) => (s.userInfo?.roles ?? []).includes('admin'),
  },

  actions: {
    setToken(token: string, expireAt?: number) {
      this.token = token
      if (expireAt) this.expireAt = expireAt
      // 每次写入时计算 maxAge, 让 cookie 与 token 有效期同步过期
      const opts: Record<string, any> = { path: '/', sameSite: 'lax' }
      if (this.expireAt) {
        const maxAge = this.expireAt - Math.floor(Date.now() / 1000)
        if (maxAge > 0) opts.maxAge = maxAge
      }
      const tokenCookie = useCookie<string | null>(TOKEN_COOKIE, opts)
      const expireCookie = useCookie<string | null>(EXPIRE_COOKIE, opts)
      if (token) {
        tokenCookie.value = token
        if (this.expireAt) expireCookie.value = String(this.expireAt)
      }
      else {
        tokenCookie.value = null
        expireCookie.value = null
        this.expireAt = 0
      }
    },

    restore() {
      if (this.token) return
      // cookie 在 SSR 与客户端均可用
      const tokenCookie = useCookie<string | null>(TOKEN_COOKIE)
      const expireCookie = useCookie<string | null>(EXPIRE_COOKIE)
      if (tokenCookie.value) {
        const e = Number(expireCookie.value || 0)
        // 已过期的 token 不恢复, 避免带着失效凭据发请求
        if (e && e < Date.now() / 1000) return
        this.token = tokenCookie.value
        if (e) this.expireAt = e
        return
      }
      // 旧版 localStorage 迁移: 首次访问时搬迁到 cookie 后清除
      if (import.meta.client) {
        const t = localStorage.getItem('hinay_token')
        if (!t) return
        const e = Number(localStorage.getItem('hinay_token_expire') || 0)
        localStorage.removeItem('hinay_token')
        localStorage.removeItem('hinay_token_expire')
        if (e && e < Date.now() / 1000) return
        this.setToken(t, e || undefined)
      }
    },

    setUserInfo(u: LoginUser | null) {
      this.userInfo = u
    },

    setMenus(menus: MenuNode[], permissions: string[]) {
      this.menus = menus || []
      this.permissions = permissions || []
      this.menusLoaded = true
    },

    hasPermission(code: string): boolean {
      if (!code) return true
      if (this.roles.includes('admin')) return true
      return this.permissions.includes(code)
    },

    reset() {
      this.setToken('')
      this.userInfo = null
      this.menus = []
      this.permissions = []
      this.menusLoaded = false
    },
  },
})
