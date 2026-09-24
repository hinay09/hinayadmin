<script setup lang="ts">
/**
 * 默认布局: el-container + 侧边栏菜单 + 顶部用户信息 + 面包屑。
 */
import { computed, ref, onMounted } from 'vue'
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
import { useAuthApi } from '~/composables/useApi'
import { filterDisplayMenus } from '~/utils/router'
import type { MenuNode } from '~/stores/user'

const userStore = useUserStore()
const { userInfo, menus } = storeToRefs(userStore)
const configStore = useConfigStore()
const { siteName, siteLogo, copyright } = storeToRefs(configStore)
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
    <el-aside :width="collapse ? '64px' : '220px'" class="app-aside">
      <div class="logo">
        <img v-if="siteLogo" :src="siteLogo" class="logo-img" alt="logo">
        <el-icon v-else class="logo-icon"><Histogram /></el-icon>
        <span v-if="!collapse" class="logo-text">{{ siteName }}</span>
      </div>
      <el-menu
        :default-active="activeMenu"
        :collapse="collapse"
        router
        unique-opened
        class="app-menu"
        background-color="#001529"
        text-color="#cfd8dc"
        active-text-color="#409EFF"
      >
        <template v-for="m in menuTree" :key="m.id">
          <el-sub-menu v-if="m.children && m.children.length" :index="String(m.id)">
            <template #title>
              <el-icon><component :is="m.icon || 'Folder'" /></el-icon>
              <span>{{ m.name }}</span>
            </template>
            <el-menu-item
              v-for="c in m.children"
              :key="c.id"
              :index="c.path"
            >
              <el-icon><component :is="c.icon || 'Document'" /></el-icon>
              <span>{{ c.name }}</span>
            </el-menu-item>
          </el-sub-menu>
          <el-menu-item v-else :index="m.path">
            <el-icon><component :is="m.icon || 'Menu'" /></el-icon>
            <template #title>{{ m.name }}</template>
          </el-menu-item>
        </template>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="app-header">
        <div class="header-left">
          <el-icon class="collapse-btn" @click="collapse = !collapse">
            <component :is="collapse ? Expand : Fold" />
          </el-icon>
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
          <el-tooltip content="刷新" placement="bottom">
            <el-icon class="header-action" @click="handleRefresh"><Refresh /></el-icon>
          </el-tooltip>
          <el-tooltip content="全屏" placement="bottom">
            <el-icon class="header-action" @click="handleFullscreen"><FullScreen /></el-icon>
          </el-tooltip>
          <MessageBell />
          <el-divider direction="vertical" />
          <el-dropdown @command="handleDropdown">
            <span class="user-info">
              <el-avatar :size="30" :src="userInfo?.avatar">
                {{ userInfo?.nickname?.charAt(0) || 'U' }}
              </el-avatar>
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
      <el-main class="app-main">
        <slot />
      </el-main>
      <el-footer v-if="copyright" class="app-footer" height="36px">{{ copyright }}</el-footer>
    </el-container>
  </el-container>
</template>

<style scoped>
.app-layout {
  height: 100vh;
}
.app-aside {
  background: #001529;
  transition: width .2s;
  overflow-x: hidden;
}
.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #fff;
  font-size: 18px;
  font-weight: 600;
  letter-spacing: 1px;
  background: linear-gradient(135deg, #1f2d3d, #001529);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}
.logo-icon {
  font-size: 22px;
  color: #409eff;
}
.logo-img {
  width: 26px;
  height: 26px;
  object-fit: contain;
  border-radius: 4px;
}
.logo-text {
  white-space: nowrap;
}
.app-menu {
  border-right: none;
}
.app-menu:not(.el-menu--collapse) {
  width: 220px;
}
.app-header {
  height: 60px;
  background: #fff;
  border-bottom: 1px solid #ebeef5;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}
.collapse-btn {
  font-size: 20px;
  cursor: pointer;
  color: #606266;
  transition: color .2s;
}
.collapse-btn:hover {
  color: #409eff;
}
.app-breadcrumb {
  font-size: 14px;
}
.app-breadcrumb :deep(.el-breadcrumb__item .el-breadcrumb__inner) {
  color: #606266;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.bc-home {
  font-size: 14px;
}
.bc-current {
  color: #303133;
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
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: #606266;
  transition: background-color .2s, color .2s;
}
.header-action:hover {
  background-color: #f0f2f5;
  color: #409eff;
}
.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 0 8px;
  height: 40px;
  border-radius: 6px;
  transition: background-color .2s;
}
.user-info:hover {
  background-color: #f0f2f5;
}
.username {
  font-size: 14px;
  color: #303133;
}
.caret {
  font-size: 12px;
  color: #909399;
}
.app-main {
  background: #f0f2f5;
  padding: 16px;
}
.app-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f0f2f5;
  border-top: 1px solid #ebeef5;
  color: #909399;
  font-size: 12px;
  padding: 0;
}
</style>
