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
        <div class="logo-badge">
          <img v-if="siteLogo" :src="siteLogo" class="logo-img" alt="logo">
          <el-icon v-else :size="17" class="logo-icon"><Histogram /></el-icon>
        </div>
        <span v-if="!collapse" class="logo-text">{{ siteName }}</span>
      </div>
      <el-menu
        :default-active="activeMenu"
        :collapse="collapse"
        router
        unique-opened
        class="app-menu"
        background-color="#0e1d3a"
        text-color="#a3b1c6"
        active-text-color="#ffffff"
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
          <el-tooltip :content="collapse ? '展开菜单' : '收起菜单'" placement="bottom">
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
              <el-avatar :size="30" :src="userInfo?.avatar" class="user-avatar">
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
      <TagsBar />
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
  background: #0e1d3a;
  transition: width .2s;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 2px 0 12px rgba(0, 21, 41, 0.1);
}
.logo {
  height: 60px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #fff;
  /* 顶部微亮的品牌区, 向下融入侧边栏底色 */
  background: linear-gradient(180deg, rgba(35, 62, 108, 0.85), rgba(14, 29, 58, 0));
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  user-select: none;
}
.logo-badge {
  width: 30px;
  height: 30px;
  border-radius: 8px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #409eff, #2f6fd8);
  box-shadow: 0 4px 10px rgba(64, 158, 255, 0.3);
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
  scrollbar-color: rgba(255, 255, 255, 0.15) transparent;
}
.app-menu.el-menu--collapse {
  padding: 8px 5px 16px;
}
.app-menu::-webkit-scrollbar {
  width: 5px;
}
.app-menu::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.15);
  border-radius: 3px;
}
.app-menu:not(.el-menu--collapse) {
  width: 220px;
}
/* 菜单项胶囊化: 圆角 + 悬浮高亮 + 激活渐变底 */
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
  color: #fff;
  background: rgba(255, 255, 255, 0.07);
}
.app-menu :deep(.el-menu-item.is-active) {
  background: linear-gradient(90deg, #409eff, #2f7ce0);
  color: #fff;
  box-shadow: 0 4px 12px rgba(64, 158, 255, 0.35);
}
.app-header {
  height: 60px;
  background: #fff;
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
  border-radius: 8px;
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
  padding: 0 10px;
  height: 40px;
  border-radius: 8px;
  transition: background-color .2s;
}
.user-info:hover {
  background-color: #f0f2f5;
}
.user-avatar {
  background: linear-gradient(135deg, #409eff, #2f7ce0);
  color: #fff;
  font-weight: 600;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.18);
}
.username {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}
.caret {
  font-size: 12px;
  color: #909399;
}
.app-main {
  background: #f5f7fa;
  padding: 16px;
}
.app-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f7fa;
  border-top: 1px solid #eef1f6;
  color: #909399;
  font-size: 12px;
  padding: 0;
}
</style>
