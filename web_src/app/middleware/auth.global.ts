/**
 * 全局路由守卫: 未登录跳 /login; 已登录拉取菜单。
 * token 持久化在 cookie, SSR 阶段即可识别登录态, 刷新不闪跳 /login。
 */
import { useUserStore, safeRedirect } from '~/stores/user'
import { useAuthApi } from '~/composables/useApi'

/** 拉取会话(用户信息+菜单)并写入 store; 返回是否成功 (失败如 token 失效, 由调用方跳登录页) */
async function loadSession(): Promise<boolean> {
  const userStore = useUserStore()
  try {
    const api = useAuthApi()
    const [info, menusRes] = await Promise.all([
      api.userInfo(),
      api.menus(),
    ])
    userStore.setUserInfo(info)
    userStore.setMenus(menusRes.menus, menusRes.permissions)
    return true
  }
  catch {
    // useRequest 的 401 分支通常已 reset+跳转过, 这里再兜底清一次会话
    userStore.reset()
    return false
  }
}

/* 并发去重: 水合后的首次拉取与期间的路由切换共享同一次请求 */
let sessionInflight: Promise<boolean> | null = null
function ensureSession(): Promise<boolean> {
  if (!sessionInflight) {
    sessionInflight = loadSession().finally(() => {
      sessionInflight = null
    })
  }
  return sessionInflight
}

export default defineNuxtRouteMiddleware(async (to) => {
  const userStore = useUserStore()
  userStore.restore()

  const isLoginPage = to.path === '/login'

  if (!userStore.token) {
    // 登录/注册页对匿名访客开放 (注册入口受 sys.allow_register 开关控制, 由页面自判)
    if (isLoginPage || to.path === '/register') return
    return navigateTo({ path: '/login', query: { redirect: to.fullPath } })
  }

  if (isLoginPage) {
    // 已登录访问登录页: 回到来源页(如有合法 redirect 参数)或首页
    return navigateTo(safeRedirect(to.query.redirect as string))
  }

  // 强制改密: 密码被管理员创建/重置/导入, 或已过有效期 -> 锁定在个人中心改密页,
  // 放行 /profile 本身, 其余页面一律带回跳参数重定向。
  if (userStore.mustChangePwd && to.path !== '/profile') {
    return navigateTo({ path: '/profile', query: { tab: 'password', redirect: to.fullPath } })
  }

  // 已登录但未拉取菜单 -> 拉取。
  // 注意: 客户端首次导航 (app:created 钩子) 先于 mount/水合执行, 若在水合中同步拉取并写入
  // store, 水合 vdom 将带着菜单/用户名与服务端渲染的空菜单 HTML 不一致 -> hydration mismatch。
  // 因此水合中的首次导航推迟到水合完成 (app:suspense:resolve) 后再拉取, 挂载完成即填充;
  // 后续路由切换照常在导航前 await, 路由级权限守卫不受影响。
  if (!userStore.menusLoaded && import.meta.client) {
    const nuxtApp = useNuxtApp()
    if (nuxtApp.isHydrating) {
      // hook 回调经 runWithContext 执行, navigateTo 可直接用
      nuxtApp.hooks.hookOnce('app:suspense:resolve', async () => {
        if (!await ensureSession()) await navigateTo('/login')
      })
    }
    else {
      if (!await ensureSession()) return navigateTo('/login')
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
