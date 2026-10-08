/**
 * 启动时恢复用户布局设置 (主题色/暗黑模式/水印等) 并应用到 DOM。
 * 在水合前运行, 与 nuxt.config 的内联脚本 (更早给 <html> 加 dark 类防闪白) 配合。
 */
import { useSettingsStore } from '~/stores/settings'

export default defineNuxtPlugin(() => {
  useSettingsStore().restore()
})
