/**
 * 布局设置 store (仿 RuoYi useSettingsStore): 主题色 / 暗黑模式 / 固定头部 /
 * 侧边栏 Logo / 多标签页 / 页面水印, 持久化到 localStorage (键 hinay_layout_setting)。
 *
 * 主题副作用 (DOM class / CSS 变量) 在 action 内即时应用; 启动恢复走
 * plugins/settings.client.ts, nuxt.config 的内联脚本会在更早的水合前加 dark 类防闪白。
 * 注意: 内联脚本读的 JSON 键与本处持久化字段需保持一致。
 */
import { defineStore } from 'pinia'
import { applyDark, applyTheme, isHexColor } from '~/utils/theme'

export interface LayoutSettings {
  /** 主题色 (6 位 hex), 运行时重写 --el-color-primary 系列变量 */
  theme: string
  /** 暗黑模式 */
  isDark: boolean
  /** 固定头部 (关闭后头部随内容滚动) */
  fixedHeader: boolean
  /** 侧边栏显示 Logo */
  showLogo: boolean
  /** 显示多标签页 */
  showTags: boolean
  /** 开启全屏水印 */
  watermark: boolean
  /** 水印自定义文案, 空则默认取当前用户 "昵称(用户名)" */
  watermarkText: string
}

export const DEFAULT_LAYOUT: LayoutSettings = {
  theme: '#409eff',
  isDark: false,
  fixedHeader: true,
  showLogo: true,
  showTags: true,
  watermark: false,
  watermarkText: '',
}

const LS_KEY = 'hinay_layout_setting'

export const useSettingsStore = defineStore('appSettings', {
  state: () => ({ ...DEFAULT_LAYOUT }),

  actions: {
    /** 切换暗黑模式 (不传参则取反) */
    toggleDark(v?: boolean) {
      this.update({ isDark: v ?? !this.isDark })
    },

    /** 设置主题色 (非法 hex 忽略) */
    setTheme(color: string) {
      if (!isHexColor(color)) return
      this.update({ theme: color.toLowerCase() })
    },

    /** 恢复默认设置 */
    reset() {
      this.update({ ...DEFAULT_LAYOUT })
    },

    /** 批量更新设置并应用 + 持久化 */
    update(partial: Partial<LayoutSettings>) {
      Object.assign(this.$state, partial)
      if (import.meta.client) {
        applyDark(this.isDark)
        applyTheme(this.theme, this.isDark)
        try {
          localStorage.setItem(LS_KEY, JSON.stringify({
            theme: this.theme,
            isDark: this.isDark,
            fixedHeader: this.fixedHeader,
            showLogo: this.showLogo,
            showTags: this.showTags,
            watermark: this.watermark,
            watermarkText: this.watermarkText,
          }))
        }
        catch {}
      }
    },

    /** 启动时从 localStorage 恢复并应用到 DOM (仅客户端) */
    restore() {
      if (!import.meta.client) return
      try {
        const raw = localStorage.getItem(LS_KEY)
        if (raw) {
          Object.assign(this.$state, { ...DEFAULT_LAYOUT, ...JSON.parse(raw) })
        }
      }
      catch {}
      applyDark(this.isDark)
      applyTheme(this.theme, this.isDark)
    },
  },
})
