/**
 * 全局路由守卫: 未登录跳 /login; 已登录拉取菜单。
 * token 持久化在 cookie, SSR 阶段即可识别登录态, 刷新不闪跳 /login。
 */
import { useUserStore, safeRedirect } from '~/stores/user'
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
    // 已登录访问登录页: 回到来源页(如有合法 redirect 参数)或首页
    return navigateTo(safeRedirect(to.query.redirect as string))
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

  // 路由级权限守卫(纵深防御): 系统管理页面要求菜单授权, 未授权直接回首页。
  // 后端对每个 API 仍会做 Casbin 校验, 此处仅避免无权限用户看到管理页面壳子。
  if (to.path.startsWith('/system') && userStore.menusLoaded && !userStore.isAdmin) {
    const allowed = collectMenuPaths(userStore.menus)
      .some(p => to.path === p || to.path.startsWith(`${p}/`))
    if (!allowed) {
      return navigateTo('/')
    }
  }
})

// collectMenuPaths 收集菜单树中所有叶子菜单的 path。
function collectMenuPaths(menus: { path: string; children?: unknown[] }[]): string[] {
  const paths: string[] = []
  for (const m of menus || []) {
    if (m.path) paths.push(m.path)
    if (m.children?.length) paths.push(...collectMenuPaths(m.children as { path: string; children?: unknown[] }[]))
  }
  return paths
}
