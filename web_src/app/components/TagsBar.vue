<script setup lang="ts">
/**
 * 多标签页导航条: 已访问页面标签 + 右键菜单 (刷新/关闭/关闭其他/关闭全部)。
 * 关闭当前激活标签时自动跳转到相邻标签; 仪表盘为固定标签不可关闭。
 */
import { ref, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { Close } from '@element-plus/icons-vue'
import { useTagsStore } from '~/stores/tags'

const route = useRoute()
const router = useRouter()
const tags = useTagsStore()

// 右键菜单状态
const menuVisible = ref(false)
const menuX = ref(0)
const menuY = ref(0)
const menuPath = ref('')

function openMenu(e: MouseEvent, path: string) {
  menuPath.value = path
  menuX.value = e.clientX
  menuY.value = e.clientY
  menuVisible.value = true
}

function closeMenu() {
  menuVisible.value = false
}

onMounted(() => {
  document.addEventListener('click', closeMenu)
})
onBeforeUnmount(() => {
  document.removeEventListener('click', closeMenu)
})

/** 关闭标签: 若关闭的是当前页则跳转相邻标签 */
function handleClose(tagPath: string) {
  const next = tags.removeTag(tagPath)
  if (next && route.path === tagPath) {
    router.push(next)
  }
}

/** 刷新: 出缓存 + 换 key 触发重建, 下一拍恢复缓存 */
async function handleRefresh(tagPath: string) {
  const tag = tags.visited.find(t => t.path === tagPath)
  if (!tag) return
  tags.bumpRefresh(tag.path, tag.name)
  if (route.path !== tagPath) {
    router.push(tagPath)
  }
  await nextTick()
  tags.addCached(tag.name)
}

function handleCloseOthers(tagPath: string) {
  tags.closeOthers(tagPath)
  if (route.path !== tagPath) router.push(tagPath)
}

function handleCloseAll() {
  const next = tags.closeAll()
  if (next && route.path !== next) router.push(next)
}
</script>

<template>
  <div class="tags-bar">
    <div class="tags-scroll">
      <div
        v-for="tag in tags.visited"
        :key="tag.path"
        class="tag-item"
        :class="{ active: route.path === tag.path }"
        @click="router.push(tag.path)"
        @click.right.prevent="openMenu($event, tag.path)"
      >
        <span class="tag-dot" />
        <span class="tag-title">{{ tag.title }}</span>
        <el-icon
          v-if="!tag.affix"
          class="tag-close"
          @click.stop="handleClose(tag.path)"
        >
          <Close />
        </el-icon>
      </div>
    </div>

    <!-- 右键菜单 -->
    <Teleport to="body">
      <div
        v-if="menuVisible"
        class="tags-context-menu"
        :style="{ left: `${menuX}px`, top: `${menuY}px` }"
        @click.stop
      >
        <div class="menu-item" @click="handleRefresh(menuPath); closeMenu()">刷新页面</div>
        <div
          class="menu-item"
          :class="{ disabled: tags.visited.find(t => t.path === menuPath)?.affix }"
          @click="handleClose(menuPath); closeMenu()"
        >关闭当前</div>
        <div class="menu-item" @click="handleCloseOthers(menuPath); closeMenu()">关闭其他</div>
        <div class="menu-item" @click="handleCloseAll(); closeMenu()">关闭全部</div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.tags-bar {
  background: var(--el-bg-color);
  border-bottom: 1px solid #f0f2f5;
  padding: 3px 12px;
}

.tags-scroll {
  display: flex;
  align-items: center;
  gap: 6px;
  overflow-x: auto;
  /* 纵向留 2px: overflow-x:auto 会把 overflow-y 计算成 auto, 没有它 hover 上移的 1px 会被裁掉 */
  padding: 2px 0;
  scrollbar-width: none;
}
.tags-scroll::-webkit-scrollbar {
  display: none;
}

.tag-item {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  font-size: 12px;
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-blank);
  cursor: pointer;
  white-space: nowrap;
  user-select: none;
  transition: all .2s;
}
/* hover 只作用于未激活标签: 激活态是常驻的更深一层, 不被瞬时反馈覆盖 */
.tag-item:not(.active):hover {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary-light-7);
  background: var(--el-color-primary-light-9);
  transform: translateY(-1px);
}
.tag-item.active {
  background: var(--el-color-primary-light-8);
  color: var(--el-color-primary);
  border-color: var(--el-color-primary-light-5);
  font-weight: 500;
  box-shadow: 0 1px 3px rgba(16, 31, 61, 0.08);
}
.tag-item.active .tag-dot {
  background: var(--el-color-primary);
}

.tag-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--el-border-color);
  flex-shrink: 0;
}
.tag-item:not(.active):hover .tag-dot {
  background: var(--el-color-primary);
}

.tag-close {
  font-size: 12px;
  border-radius: 50%;
  transition: all .15s;
}
.tag-close:hover {
  background: rgba(0, 0, 0, .1);
  color: inherit;
}
.tag-item.active .tag-close:hover {
  background: var(--el-color-primary-light-8);
  color: var(--el-color-primary);
}

.tags-context-menu {
  position: fixed;
  z-index: 3000;
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  box-shadow: var(--el-box-shadow-light);
  padding: 6px 0;
  min-width: 128px;
}
.menu-item {
  padding: 6px 16px;
  font-size: 13px;
  color: var(--el-text-color-regular);
  cursor: pointer;
}
.menu-item:hover {
  background: var(--el-fill-color-light);
  color: var(--el-color-primary);
}
.menu-item.disabled {
  color: var(--el-text-color-disabled);
  pointer-events: none;
}
</style>
