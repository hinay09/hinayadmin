/**
 * 全局配置 store: 加载 /system/configs/all 的启用配置, 驱动界面展示
 * (站点名称/Logo/页脚版权等), 并缓存到 localStorage 供登录页(未登录态)读取。
 *
 * 配置键与后端 init.sql 种子对齐:
 *   sys.name           系统名称 (侧边栏/登录页/浏览器标题)
 *   sys.logo           系统Logo (图片URL, 空则回退默认图标)
 *   sys.copyright      版权信息 (页脚)
 *   sys.allow_register 是否开放注册 (布尔)
 * 其他自定义键可通过 getText/getNumber/getBool/getJson 泛型方法读取。
 */
import { defineStore } from 'pinia'
import { useConfigApi } from '~/composables/useApi'

export interface ConfigEntry {
  value: string
  type: number
  name: string
}

/** 约定的内置配置键 */
export const CONFIG_KEYS = {
  siteName: 'sys.name',
  siteLogo: 'sys.logo',
  copyright: 'sys.copyright',
  allowRegister: 'sys.allow_register',
} as const

const CACHE_KEY = 'hinay_app_config'
const DEFAULT_SITE_NAME = 'Hinay Admin'

/** 并发去重: 同一会话内多个组件同时 ensureLoaded 只发一次请求 */
let inflight: Promise<void> | null = null

function parseBool(v: string | undefined): boolean {
  return v === 'true' || v === '1'
}

export const useConfigStore = defineStore('appConfig', {
  state: () => ({
    configs: {} as Record<string, ConfigEntry>,
    loaded: false,
  }),

  getters: {
    /** 系统名称(侧边栏/登录页/浏览器标题) */
    siteName: s => s.configs[CONFIG_KEYS.siteName]?.value?.trim() || DEFAULT_SITE_NAME,
    /** Logo 图片 URL, 未配置返回空串由调用方回退默认图标 */
    siteLogo: s => s.configs[CONFIG_KEYS.siteLogo]?.value?.trim() || '',
    /** 页脚版权文字 */
    copyright: s => s.configs[CONFIG_KEYS.copyright]?.value?.trim() || '',
    /** 是否开放注册 */
    allowRegister: s => parseBool(s.configs[CONFIG_KEYS.allowRegister]?.value),
  },

  actions: {
    /** 读文本配置, 空值回退 fallback */
    getText(key: string, fallback = ''): string {
      const v = this.configs[key]?.value?.trim()
      return v ? v : fallback
    },
    /** 读数字配置, 解析失败回退 fallback */
    getNumber(key: string, fallback = 0): number {
      const n = Number(this.configs[key]?.value)
      return Number.isFinite(n) ? n : fallback
    },
    /** 读布尔配置 (true/1 为真) */
    getBool(key: string): boolean {
      return parseBool(this.configs[key]?.value)
    },
    /** 读 JSON 配置, 解析失败返回 fallback */
    getJson<T>(key: string, fallback: T | null = null): T | null {
      try {
        const raw = this.configs[key]?.value
        return raw ? JSON.parse(raw) as T : fallback
      }
      catch {
        return fallback
      }
    },

    /** 从 localStorage 恢复上次缓存(登录页未登录态使用), 仅客户端可调 */
    restore() {
      if (this.loaded || Object.keys(this.configs).length) return
      if (!import.meta.client) return
      try {
        const raw = localStorage.getItem(CACHE_KEY)
        if (!raw) return
        const cached = JSON.parse(raw)
        if (cached && typeof cached === 'object') this.configs = cached
      }
      catch {}
    },

    /** 拉取全部启用配置; 配置加载失败不影响界面, 静默保留现有值 */
    async fetch() {
      try {
        const res = await useConfigApi().all()
        this.configs = res.list || {}
        this.loaded = true
        if (import.meta.client) {
          try {
            localStorage.setItem(CACHE_KEY, JSON.stringify(this.configs))
          }
          catch {}
        }
      }
      catch {}
    },

    /** 会话内只加载一次(布局挂载时调用) */
    ensureLoaded(): Promise<void> {
      if (this.loaded) return Promise.resolve()
      if (inflight) return inflight
      inflight = this.fetch().finally(() => { inflight = null })
      return inflight
    },

    /** 强制重新加载(配置管理页增删改后调用, 界面即时生效) */
    refresh(): Promise<void> {
      return this.fetch()
    },
  },
})
