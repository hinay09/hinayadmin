/**
 * 用户会话 store: token / userInfo / 菜单 / 权限标识
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
      if (import.meta.client) {
        if (token) {
          localStorage.setItem('hinay_token', token)
          if (expireAt) localStorage.setItem('hinay_token_expire', String(expireAt))
        }
        else {
          localStorage.removeItem('hinay_token')
          localStorage.removeItem('hinay_token_expire')
        }
      }
    },

    restore() {
      if (import.meta.client && !this.token) {
        const t = localStorage.getItem('hinay_token')
        if (t) {
          this.token = t
          const e = localStorage.getItem('hinay_token_expire')
          if (e) this.expireAt = Number(e)
        }
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
      this.expireAt = 0
      this.userInfo = null
      this.menus = []
      this.permissions = []
      this.menusLoaded = false
    },
  },
})
