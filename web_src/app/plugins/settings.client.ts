/**
 * 启动时恢复用户布局设置, 分两步以避免 hydration mismatch:
 *
 * 1. 水合前: 只把暗黑模式/主题色作为 DOM 副作用应用到 <html> (与 nuxt.config 的
 *    内联防闪白脚本同理)。DOM 类名/CSS 变量不参与 vnode 比对, 不会引发水合不一致,
 *    又保住了首帧颜色与用户设置一致。
 * 2. 水合完成后 (app:suspense:resolve): 再把完整设置写入 store。fixedHeader/showLogo/
 *    showTags 等字段影响模板结构, 而 SSR 只能按默认值渲染 —— 若在水合前写入 store,
 *    客户端 vdom 会与服务端 HTML 不一致 ("Hydration completed but contains mismatches")。
 */
import { useSettingsStore, SETTINGS_LS_KEY } from '~/stores/settings'
import { applyDark, applyTheme, isHexColor } from '~/utils/theme'

export default defineNuxtPlugin((nuxtApp) => {
  // 第 1 步: DOM 副作用 (仅暗黑类 + 主题色变量, 不碰 store 状态)
  try {
    const raw = localStorage.getItem(SETTINGS_LS_KEY)
    if (raw) {
      const s = JSON.parse(raw)
      if (s && typeof s === 'object') {
        const dark = !!s.isDark
        applyDark(dark)
        if (typeof s.theme === 'string' && isHexColor(s.theme)) {
          applyTheme(s.theme.toLowerCase(), dark)
        }
      }
    }
  }
  catch {}

  // 第 2 步: 水合完成后再恢复 store 状态, 布局开关随反应式更新
  nuxtApp.hooks.hookOnce('app:suspense:resolve', () => {
    useSettingsStore().restore()
  })
})
