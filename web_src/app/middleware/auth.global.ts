/**
 * 全局路由守卫: 未登录跳 /login; 已登录拉取菜单。
 */
import { useUserStore } from '~/stores/user'
import { useAuthApi } from '~/composables/useApi'

export default defineNuxtRouteMiddleware(async (to) => {
  const userStore = useUserStore()
  userStore.restore()

  const isLoginPage = to.path === '/login'

  if (!userStore.token) {
    if (isLoginPage) return
    return navigateTo({ path: '/login', query: { redirect: to.fullPath } })
  }

  if (isLoginPage) {
    return navigateTo('/')
  }

  // 已登录但未拉取菜单 -> 拉取
  if (!userStore.menusLoaded && import.meta.client) {
    try {
      const api = useAuthApi()
      const [info, menusRes] = await Promise.all([
        api.userInfo(),
        api.menus(),
      ])
      userStore.setUserInfo(info)
      userStore.setMenus(menusRes.menus, menusRes.permissions)
    }
    catch {
      userStore.reset()
      return navigateTo('/login')
    }
  }
})
