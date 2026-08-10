/**
 * v-permission 指令: 按权限标识隐藏元素。
 * 用法: <el-button v-permission="'system:user:create'">新增</el-button>
 */
import { useUserStore } from '~/stores/user'

function apply(el: HTMLElement, code: string) {
  if (!code) return
  const userStore = useUserStore()
  // 菜单/权限未就绪时不做处理, 避免 SSR 或刷新后首帧误判
  if (!userStore.menusLoaded) return
  if (!userStore.hasPermission(code)) {
    el.parentNode?.removeChild(el)
  }
}

export default defineNuxtPlugin((nuxtApp) => {
  nuxtApp.vueApp.directive('permission', {
    mounted(el: HTMLElement, binding) {
      apply(el, binding.value as string)
    },
    updated(el: HTMLElement, binding) {
      apply(el, binding.value as string)
    },
  })
})
