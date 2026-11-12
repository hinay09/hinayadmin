<script setup lang="ts">
/**
 * 默认布局: el-container + 侧边栏菜单 + 顶部用户信息 + 面包屑。
 * 布局设置 (stores/settings) 驱动: 布局模式 (经典整树 / Snowy 式双层菜单) /
 * 固定头部 / 侧边栏 Logo / 多标签页 / 全屏水印,
 * 顶栏齿轮 (SettingDrawer) 可切换暗黑模式与主题色。
 */
import { computed, ref, onMounted, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { ElMessageBox, ElMessage } from 'element-plus'
import {
  Fold,
  Expand,
  Refresh,
  FullScreen,
  Histogram,
  ArrowDown,
  SwitchButton,
  User,
  House,
} from '@element-plus/icons-vue'
import { useUserStore } from '~/stores/user'
import { useConfigStore } from '~/stores/config'
import { useSettingsStore } from '~/stores/settings'
import { useAuthApi } from '~/composables/useApi'
import { markSessionDead } from '~/composables/useRequest'
import { filterDisplayMenus, findRootByPath } from '~/utils/router'
import type { MenuNode } from '~/stores/user'

const userStore = useUserStore()
const { userInfo, menus } = storeToRefs(userStore)
const configStore = useConfigStore()
const { siteName, siteLogo, copyright } = storeToRefs(configStore)
const settingsStore = useSettingsStore()
const { fixedHeader, showLogo, showTags } = storeToRefs(settingsStore)
const route = useRoute()
const router = useRouter()
const api = useAuthApi()

// 全局配置(sys_config)驱动品牌区/页脚展示; 会话内只拉取一次, 失败静默回退默认值
onMounted(() => {
  configStore.ensureLoaded()
})

// 后端 /auth/menus 已返回菜单树, 仅过滤掉按钮节点。
const menuTree = computed(() => filterDisplayMenus(menus.value))
const activeMenu = computed(() => route.path)
const collapse = ref(false)

// 双层菜单布局 (Snowy 式): 窄图标栏放一级菜单, 子菜单面板放当前一级的二级
const isMix = computed(() => settingsStore.layout === 'dual')

/**
 * 当前激活的一级菜单 (双层布局用): 路由命中菜单时更新;
 * 菜单外页面 (如个人中心) 保持上一个激活项, 从未命中过则默认第一个根。
 */
const activeRoot = ref<MenuNode | null>(null)
watch(
  [() => route.path, menuTree],
  () => {
    const matched = findRootByPath(menuTree.value, route.path)
    if (matched) {
      activeRoot.value = matched
      return
    }
    if (!activeRoot.value && menuTree.value.length) activeRoot.value = menuTree.value[0]
  },
  { immediate: true },
)

// 双层布局下当前一级是否有二级 (无二级时子菜单面板整个隐藏, 内容区通栏)
const hasPanel = computed(() => !!(activeRoot.value?.children?.length))

// 侧边菜单数据源: 经典模式渲染整树, 双层模式只渲染激活一级的二级
const sidebarMenus = computed(() => (isMix.value ? activeRoot.value?.children || [] : menuTree.value))

// aside 宽度: 经典 = 220/64 (collapse); 双层 = 图标栏 64 + 面板 220 (面板隐藏时仅 64)
const asideWidth = computed(() => {
  if (!isMix.value) return collapse.value ? '64px' : '220px'
  return !hasPanel.value || collapse.value ? '64px' : '284px'
})

// 图标栏点击: 有二级仅切换面板内容不跳转 (收起时同样不跳, 交给悬浮附加框选择),
// 由用户点选具体二级菜单后再导航; 无二级直接打开该页面
function handleRailClick(node: MenuNode) {
  if (node.children?.length) {
    activeRoot.value = node
    return
  }
  router.push(node.path)
}

/**
 * 根据当前路由 path 在菜单树中追溯路径, 用于面包屑显示。
 */
const breadcrumb = computed<MenuNode[]>(() => {
  const path = route.path
  const trail: MenuNode[] = []
  const dfs = (nodes: MenuNode[], chain: MenuNode[]): boolean => {
    for (const n of nodes) {
      const next = [...chain, n]
      if (n.path === path) {
        trail.push(...next)
        return true
      }
      if (n.children?.length && dfs(n.children, next)) return true
    }
    return false
  }
  dfs(menuTree.value, [])
  return trail
})

async function handleLogout() {
  try {
    await ElMessageBox.confirm('确认退出登录?', '提示', { type: 'warning' })
  }
  catch {
    return
  }
  try {
    // 先标记会话失效: 登出瞬间在途请求(铃铛轮询等)的 401 走静默处理, 不再误弹"登录已过期"
    markSessionDead(userStore.token)
    await api.logout()
  }
  catch {}
  userStore.reset()
  ElMessage.success('已退出')
  router.replace('/login')
}

function handleDropdown(cmd: string) {
  if (cmd === 'logout') handleLogout()
  else if (cmd === 'profile') router.push('/profile')
}

function handleRefresh() {
  if (typeof window !== 'undefined') window.location.reload()
}

function handleFullscreen() {
  if (typeof document === 'undefined') return
  const el = document.documentElement
  if (!document.fullscreenElement) {
    el.requestFullscreen?.().catch(() => {})
  }
  else {
    document.exitFullscreen?.().catch(() => {})
  }
}
</script>

<template>
  <el-container class="app-layout">
    <el-aside :width="asideWidth" class="app-aside" :class="{ 'app-aside--dual': isMix }">
      <!-- 双层菜单: 窄图标栏 (一级菜单, 图标+名称竖排, 面板收起时常驻) -->
      <div v-if="isMix" class="rail">
        <div v-if="showLogo" class="rail-logo">
          <div class="logo-badge">
            <img v-if="siteLogo" :src="siteLogo" class="logo-img" alt="logo">
            <el-icon v-else :size="17" class="logo-icon"><Histogram /></el-icon>
          </div>
        </div>
        <div class="rail-menu">
          <el-popover
            v-for="m in menuTree"
            :key="m.id"
            trigger="hover"
            placement="right-start"
            :show-arrow="false"
            :offset="2"
            :width="200"
            :disabled="!collapse || !m.children?.length"
            popper-class="rail-flyout-popper"
          >
            <template #reference>
              <div
                class="rail-item"
                :class="{ 'is-active': activeRoot?.id === m.id }"
                @click="handleRailClick(m)"
              >
                <el-icon :size="18"><component :is="m.icon || 'Menu'" /></el-icon>
                <span class="rail-item__name">{{ m.name }}</span>
              </div>
            </template>
            <!-- 收起面板时悬浮弹出二级附加框: 复用经典「收起菜单」的 el-menu 原生样式 -->
            <div class="rail-flyout">
              <el-menu
                :default-active="activeMenu"
                router
                unique-opened
                class="rail-flyout-menu"
              >
                <SidebarItem v-for="c in m.children" :key="c.id" :item="c" />
              </el-menu>
            </div>
          </el-popover>
        </div>
      </div>
      <!-- 菜单列: 经典模式 = Logo + 整树菜单; 双层模式 = 面板头 + 当前一级的二级 -->
      <div v-show="!isMix || (hasPanel && !collapse)" class="menu-col">
        <div v-if="isMix && activeRoot" class="panel-head">
          <el-icon :size="16"><component :is="activeRoot.icon || 'Folder'" /></el-icon>
          <span class="panel-head__name">{{ activeRoot.name }}</span>
        </div>
        <div v-else-if="!isMix && showLogo" class="logo">
          <div class="logo-badge">
            <img v-if="siteLogo" :src="siteLogo" class="logo-img" alt="logo">
            <el-icon v-else :size="17" class="logo-icon"><Histogram /></el-icon>
          </div>
          <span v-if="!collapse" class="logo-text">{{ siteName }}</span>
        </div>
        <el-menu
          :default-active="activeMenu"
          :collapse="collapse && !isMix"
          router
          unique-opened
          class="app-menu"
        >
          <SidebarItem v-for="m in sidebarMenus" :key="m.id" :item="m" />
        </el-menu>
      </div>
    </el-aside>
    <el-container class="app-body" :class="{ 'app-body--scroll': !fixedHeader }">
      <el-header class="app-header">
        <div class="header-left">
          <el-tooltip :content="isMix ? (collapse ? '展开面板' : '收起面板') : (collapse ? '展开菜单' : '收起菜单')" placement="bottom">
            <el-icon class="header-action" @click="collapse = !collapse">
              <component :is="collapse ? Expand : Fold" />
            </el-icon>
          </el-tooltip>
          <el-breadcrumb separator="/" class="app-breadcrumb">
            <el-breadcrumb-item :to="{ path: '/dashboard' }">
              <el-icon class="bc-home"><House /></el-icon>
              <span>首页</span>
            </el-breadcrumb-item>
            <el-breadcrumb-item v-for="(b, idx) in breadcrumb" :key="b.id">
              <span :class="{ 'bc-current': idx === breadcrumb.length - 1 }">{{ b.name }}</span>
            </el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="header-right">
          <MenuSearch />
          <el-tooltip content="刷新" placement="bottom">
            <el-icon class="header-action" @click="handleRefresh"><Refresh /></el-icon>
          </el-tooltip>
          <el-tooltip content="全屏" placement="bottom">
            <el-icon class="header-action" @click="handleFullscreen"><FullScreen /></el-icon>
          </el-tooltip>
          <SettingDrawer />
          <TenantSwitcher />
          <MessageBell />
          <el-divider direction="vertical" />
          <el-dropdown @command="handleDropdown">
            <span class="user-info">
              <span class="avatar-wrap">
                <el-avatar :size="30" :src="userInfo?.avatar" class="user-avatar">
                  {{ userInfo?.nickname?.charAt(0) || 'U' }}
                </el-avatar>
              </span>
              <span class="username">{{ userInfo?.nickname || userInfo?.username }}</span>
              <el-icon class="caret"><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile" :icon="User">个人中心</el-dropdown-item>
                <el-dropdown-item divided command="logout" :icon="SwitchButton">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>
      <TagsBar v-if="showTags" />
      <el-main class="app-main">
        <slot />
      </el-main>
      <el-footer v-if="copyright" class="app-footer" height="36px">{{ copyright }}</el-footer>
    </el-container>
    <!-- 全屏防泄密水印 (布局设置开关, 默认关闭) -->
    <AppWatermark />
  </el-container>
</template>

<style scoped>
.app-layout {
  height: 100vh;
}
/* 右列: 默认固定头部 (内容区独立滚动); 关闭固定头部后整列滚动, 头部随内容滚走 */
.app-body {
  overflow: hidden;
}
.app-body--scroll {
  overflow-y: auto;
}
.app-body--scroll .app-main {
  overflow: visible;
}
.app-aside {
  background: var(--el-bg-color);
  transition: width .2s;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 2px 0 10px rgba(24, 60, 120, 0.06);
}
/* 双层菜单布局: aside 变为横向两列 (窄图标栏 + 子菜单面板) */
.app-aside--dual {
  flex-direction: row;
}
/* 窄图标栏: 一级菜单, 图标+名称竖排, 常驻不随面板收起 */
.rail {
  width: 64px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--el-border-color-lighter);
}
.rail-logo {
  height: 60px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.rail-menu {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 8px 4px 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  scrollbar-width: thin;
  scrollbar-color: var(--el-border-color-light) transparent;
}
.rail-menu::-webkit-scrollbar {
  width: 5px;
}
.rail-menu::-webkit-scrollbar-thumb {
  background: var(--el-border-color-light);
  border-radius: 3px;
}
/* 一级菜单项: 激活态与菜单项同款主色系 */
.rail-item {
  width: 56px;
  min-height: 52px;
  padding: 7px 4px;
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  cursor: pointer;
  color: var(--el-text-color-regular);
  transition: background-color .2s, color .2s;
}
.rail-item:hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.rail-item.is-active {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-8);
  font-weight: 500;
}
/* 名称最多两行居中, 超出省略 */
.rail-item__name {
  font-size: 12px;
  line-height: 1.3;
  text-align: center;
  word-break: break-all;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  overflow: hidden;
}
/* 面板收起时悬浮一级菜单弹出的二级附加框 (el-popover teleport 到 body, 仅重置内边距) */
:global(.rail-flyout-popper.el-popper) {
  padding: 6px;
}
/* 宽度由 popover 的 width prop 显式给定, 内容 100% 填充, 悬浮高亮不会超出弹层 */
.rail-flyout {
  width: 100%;
  max-height: 60vh;
  overflow-y: auto;
}
/* 弹层内菜单: 复用经典「收起菜单」的 EP 原生样式, 仅去右边线 + 紧凑行高 */
.rail-flyout-menu {
  border-right: none;
}
.rail-flyout-menu :deep(.el-menu-item),
.rail-flyout-menu :deep(.el-sub-menu__title) {
  height: 40px;
  line-height: 40px;
}
/* 菜单列: 经典模式 = Logo + 整树; 双层模式 = 面板头 + 当前一级的二级 */
.menu-col {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.panel-head {
  height: 60px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  color: var(--el-text-color-primary);
  font-size: 14px;
  font-weight: 600;
}
.panel-head .el-icon {
  color: var(--el-color-primary);
}
.panel-head__name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.logo {
  height: 60px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: var(--el-text-color-primary);
  border-bottom: 1px solid var(--el-border-color-lighter);
  user-select: none;
}
.logo-badge {
  width: 30px;
  height: 30px;
  border-radius: 9px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--el-color-primary-light-3), var(--el-color-primary));
  box-shadow: 0 4px 10px rgba(47, 111, 216, 0.28);
  transition: transform .25s;
}
/* hover 品牌区时徽章俏皮微倾 */
.logo:hover .logo-badge {
  transform: rotate(-8deg) scale(1.06);
}
.logo-icon {
  color: #fff;
}
.logo-img {
  width: 18px;
  height: 18px;
  object-fit: contain;
  border-radius: 3px;
}
.logo-text {
  font-size: 17px;
  font-weight: 600;
  letter-spacing: 1px;
  white-space: nowrap;
}
.app-menu {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  border-right: none;
  padding: 8px 10px 16px;
  scrollbar-width: thin;
  scrollbar-color: var(--el-border-color-light) transparent;
  /* 暗黑下菜单背景走 EP 变量, 这里显式跟随避免 aside/菜单出现色差 */
  background: transparent;
}
.app-menu.el-menu--collapse {
  padding: 8px 5px 16px;
}
.app-menu::-webkit-scrollbar {
  width: 5px;
}
.app-menu::-webkit-scrollbar-thumb {
  background: var(--el-border-color-light);
  border-radius: 3px;
}
.app-menu:not(.el-menu--collapse) {
  width: 220px;
}
/* 菜单项胶囊化: 圆角 + 悬浮高亮; 激活态 = 浅主色底 + 主色刻度条 + 主色图标 */
.app-menu :deep(.el-menu-item),
.app-menu :deep(.el-sub-menu__title) {
  height: 44px;
  line-height: 44px;
  border-radius: 8px;
  margin: 2px 0;
  transition: background-color .2s, color .2s;
}
.app-menu :deep(.el-menu-item:hover),
.app-menu :deep(.el-sub-menu__title:hover) {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
/* 悬浮时图标染主色, 与激活态呼应 */
.app-menu :deep(.el-menu-item .el-icon),
.app-menu :deep(.el-sub-menu__title .el-icon) {
  transition: color .2s, transform .2s;
}
.app-menu :deep(.el-menu-item:hover .el-icon),
.app-menu :deep(.el-sub-menu__title:hover .el-icon) {
  color: var(--el-color-primary);
}
.app-menu :deep(.el-menu-item.is-active) {
  position: relative;
  background: var(--el-color-primary-light-8);
  color: var(--el-color-primary);
  font-weight: 500;
}
.app-menu :deep(.el-menu-item.is-active)::before {
  content: '';
  position: absolute;
  left: 5px;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 18px;
  border-radius: 2px;
  background: var(--el-color-primary);
  box-shadow: 0 0 8px rgba(64, 158, 255, 0.4);
}
.app-menu :deep(.el-menu-item.is-active .el-icon) {
  color: var(--el-color-primary);
}
.app-header {
  height: 60px;
  flex-shrink: 0;
  background: var(--el-bg-color);
  border-bottom: none;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.06);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  position: relative;
  z-index: 5;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}
.app-breadcrumb {
  font-size: 14px;
}
.app-breadcrumb :deep(.el-breadcrumb__item .el-breadcrumb__inner) {
  color: var(--el-text-color-regular);
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.bc-home {
  font-size: 14px;
}
.bc-current {
  color: var(--el-text-color-primary);
  font-weight: 500;
}
.header-right {
  display: flex;
  align-items: center;
  gap: 4px;
}
.header-action {
  font-size: 18px;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--el-text-color-regular);
  transition: background-color .2s, color .2s;
}
.header-action:hover {
  background-color: var(--el-fill-color);
  color: var(--el-color-primary);
}
.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 0 10px;
  height: 40px;
  border-radius: 8px;
  transition: background-color .2s;
}
.user-info:hover {
  background-color: var(--el-fill-color);
}
.avatar-wrap {
  position: relative;
  display: inline-flex;
}
/* 在线状态点 */
.avatar-wrap::after {
  content: '';
  position: absolute;
  right: -1px;
  bottom: -1px;
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: #2fbf71;
  border: 2px solid var(--el-bg-color);
}
.user-avatar {
  background: linear-gradient(135deg, var(--el-color-primary-light-3), var(--el-color-primary));
  color: #fff;
  font-weight: 600;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.18);
}
.username {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}
.caret {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.app-main {
  background: var(--el-bg-color-page);
  padding: 16px;
}
.app-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--el-bg-color-page);
  border-top: 1px solid var(--el-border-color-lighter);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  padding: 0;
  flex-shrink: 0;
}
</style>
