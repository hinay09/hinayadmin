/**
 * 全局注册 @element-plus/icons-vue 所有图标,
 * 使 <el-icon><component :is="'Setting'" /></el-icon> 等动态用法生效。
 */
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

export default defineNuxtPlugin((nuxtApp) => {
  for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
    nuxtApp.vueApp.component(key, component)
  }
})
