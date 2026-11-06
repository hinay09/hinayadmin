<script setup lang="ts">
/**
 * 菜单搜索 (顶栏放大镜 / Ctrl+K): 弹窗内按 名称/路径/所属目录 模糊过滤当前用户可见菜单,
 * 键盘 ↑↓ 选择、Enter 跳转; 空关键词时展示最近访问标签作为快捷入口。
 * - 数据源为 userStore.menus (后端按角色/租户下发), 搜索结果与侧边栏天然一致, 不会出现越权入口;
 * - 纯客户端过滤, 不新增后端接口; 多关键词空格分隔, 逐词 AND 匹配。
 */
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Search, Clock } from '@element-plus/icons-vue'
import { useUserStore } from '~/stores/user'
import { useTagsStore } from '~/stores/tags'
import { filterDisplayMenus } from '~/utils/router'
import type { MenuNode } from '~/stores/user'

/** 可跳转的叶子菜单 (type=2 且有 path), trail 为上级目录名轨迹 */
interface SearchEntry {
  id: number
  name: string
  path: string
  icon: string
  /** 面包屑文本 (不含自身), 如 "系统管理" */
  trail: string
}

const MAX_RESULTS = 50

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const tagsStore = useTagsStore()

const visible = ref(false)
const keyword = ref('')
const activeIdx = ref(0)
const inputRef = ref<{ focus: () => void } | null>(null)
const listRef = ref<HTMLElement | null>(null)

/** 扁平化当前用户可见菜单, 只保留可点击跳转的节点 (含目录名轨迹, 供展示与匹配) */
const entries = computed<SearchEntry[]>(() => {
  const out: SearchEntry[] = []
  const walk = (nodes: MenuNode[], trail: string[]) => {
    for (const n of nodes) {
      if (n.type === 2 && n.path) {
        out.push({ id: n.id, name: n.name, path: n.path, icon: n.icon || '', trail: trail.join(' / ') })
      }
      if (n.children?.length) walk(n.children, [...trail, n.name])
    }
  }
  walk(filterDisplayMenus(userStore.menus), [])
  return out
})

/** 空关键词时的快捷入口: 最近访问标签倒序 (排除固定工作台), 至多 8 条 */
const recents = computed(() => {
  return [...tagsStore.visited].reverse().filter(t => !t.affix).slice(0, 8)
})

/** 拆分关键词 (小写、去空), 多个词逐词 AND 命中 */
const terms = computed(() => keyword.value.trim().toLowerCase().split(/\s+/).filter(Boolean))

const filtered = computed(() => {
  const ts = terms.value
  if (!ts.length) return { list: [] as SearchEntry[], truncated: false }
  const list: SearchEntry[] = []
  let truncated = false
  for (const e of entries.value) {
    const hay = `${e.trail} ${e.name} ${e.path}`.toLowerCase()
    if (ts.every(t => hay.includes(t))) {
      if (list.length >= MAX_RESULTS) {
        truncated = true
        break
      }
      list.push(e)
    }
  }
  return { list, truncated }
})

const results = computed(() => filtered.value.list)

/**
 * 按关键词把文本切成命中/未命中片段 (供高亮): 多关键词逐一细分,
 * 已命中片段不再细分, 未命中返回原文单片段。
 */
function segments(text: string): { text: string, hit: boolean }[] {
  let segs = [{ text, hit: false }]
  for (const t of terms.value) {
    const next: { text: string, hit: boolean }[] = []
    for (const s of segs) {
      if (s.hit || !s.text) {
        next.push(s)
        continue
      }
      let rest = s.text
      let cut = rest.toLowerCase().indexOf(t)
      while (cut >= 0) {
        if (cut > 0) next.push({ text: rest.slice(0, cut), hit: false })
        next.push({ text: rest.slice(cut, cut + t.length), hit: true })
        rest = rest.slice(cut + t.length)
        cut = rest.toLowerCase().indexOf(t)
      }
      if (rest) next.push({ text: rest, hit: false })
    }
    segs = next
  }
  return segs
}

function open() {
  visible.value = true
}

function go(path: string) {
  visible.value = false
  if (path && path !== route.path) router.push(path)
}

function goActive() {
  const item = results.value[activeIdx.value]
  if (item) go(item.path)
}

/** 输入框键盘导航: ↑↓ 循环切换 (滚动跟随), Enter 跳转 (Esc 由弹窗默认行为关闭) */
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    if (!results.value.length) return
    const delta = e.key === 'ArrowDown' ? 1 : -1
    activeIdx.value = (activeIdx.value + delta + results.value.length) % results.value.length
    nextTick(() => listRef.value?.querySelector('.ms-item.active')?.scrollIntoView({ block: 'nearest' }))
  }
  else if (e.key === 'Enter') {
    e.preventDefault()
    goActive()
  }
}

/** 全局快捷键 Ctrl/Cmd + K 唤起搜索 */
function onGlobalKeydown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && typeof e.key === 'string' && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    open()
  }
}

async function onOpened() {
  activeIdx.value = 0
  await nextTick()
  inputRef.value?.focus()
}

function onClosed() {
  keyword.value = ''
  activeIdx.value = 0
}

watch(keyword, () => {
  activeIdx.value = 0
})

onMounted(() => {
  window.addEventListener('keydown', onGlobalKeydown)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onGlobalKeydown)
})
</script>

<template>
  <el-tooltip content="菜单搜索 (Ctrl + K)" placement="bottom">
    <el-icon class="ms-trigger" @click="open">
      <Search />
    </el-icon>
  </el-tooltip>

  <el-dialog
    v-model="visible"
    class="menu-search-dialog"
    append-to-body
    :show-close="false"
    width="560px"
    top="12vh"
    @opened="onOpened"
    @closed="onClosed"
  >
    <div class="ms-panel">
      <div class="ms-input-wrap">
        <el-icon class="ms-input-icon"><Search /></el-icon>
        <el-input
          ref="inputRef"
          v-model="keyword"
          placeholder="搜索菜单: 名称 / 路径 / 所属目录, 多词空格分隔"
          size="large"
          clearable
          @keydown="onKeydown"
        />
      </div>
      <div ref="listRef" class="ms-list">
        <template v-if="results.length">
          <div
            v-for="(item, i) in results"
            :key="item.id"
            class="ms-item"
            :class="{ active: i === activeIdx }"
            @click="go(item.path)"
            @mousemove="activeIdx = i"
          >
            <el-icon class="ms-item-icon"><component :is="item.icon || 'Document'" /></el-icon>
            <span class="ms-item-name">
              <template v-for="(seg, si) in segments(item.name)" :key="si">
                <mark v-if="seg.hit">{{ seg.text }}</mark>
                <template v-else>{{ seg.text }}</template>
              </template>
            </span>
            <span class="ms-item-trail">
              <template v-for="(seg, si) in segments(item.trail)" :key="si">
                <mark v-if="seg.hit">{{ seg.text }}</mark>
                <template v-else>{{ seg.text }}</template>
              </template>
            </span>
          </div>
          <div v-if="filtered.truncated" class="ms-more">仅展示前 {{ MAX_RESULTS }} 条, 请继续输入缩小范围</div>
        </template>
        <template v-else-if="!terms.length">
          <div v-if="recents.length" class="ms-group">最近访问</div>
          <div v-for="r in recents" :key="r.path" class="ms-item" @click="go(r.path)">
            <el-icon class="ms-item-icon"><Clock /></el-icon>
            <span class="ms-item-name">{{ r.title }}</span>
            <span class="ms-item-trail">{{ r.path }}</span>
          </div>
          <div v-if="!recents.length" class="ms-empty">输入关键词, 按名称 / 路径 / 所属目录快速定位功能菜单</div>
        </template>
        <div v-else class="ms-empty">未找到与「{{ keyword.trim() }}」匹配的菜单</div>
      </div>
      <div class="ms-footer">
        <span><kbd>↑</kbd><kbd>↓</kbd> 切换</span>
        <span><kbd>Enter</kbd> 打开</span>
        <span><kbd>Esc</kbd> 关闭</span>
        <span v-if="results.length" class="ms-count">{{ results.length }} 条结果</span>
      </div>
    </div>
  </el-dialog>
</template>

<!-- 触发按钮随布局渲染, 用 scoped; 弹窗 append-to-body 脱离组件树, 骨架样式需全局 (类名带 ms-/menu-search 前缀防污染) -->
<style scoped>
.ms-trigger {
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
.ms-trigger:hover {
  background-color: var(--el-fill-color);
  color: var(--el-color-primary);
}
.ms-panel {
  display: flex;
  flex-direction: column;
}
.ms-input-wrap {
  position: relative;
  padding: 12px 14px 8px;
}
.ms-input-icon {
  position: absolute;
  left: 26px;
  /* 上内边距 12px + large 输入框高 40px 的一半, 精确对齐输入框中线 */
  top: 32px;
  transform: translateY(-50%);
  color: var(--el-text-color-secondary);
  pointer-events: none;
}
.ms-input-wrap :deep(.el-input__wrapper) {
  padding-left: 34px;
  border-radius: 8px;
}
.ms-list {
  max-height: 52vh;
  overflow-y: auto;
  padding: 4px 8px 8px;
  scrollbar-width: thin;
  scrollbar-color: var(--el-border-color-light) transparent;
}
.ms-group {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  padding: 6px 10px 4px;
}
.ms-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 10px;
  border-radius: 8px;
  cursor: pointer;
  transition: background-color .15s;
}
.ms-item:hover,
.ms-item.active {
  background: var(--el-fill-color-light);
}
.ms-item.active .ms-item-icon,
.ms-item.active .ms-item-name {
  color: var(--el-color-primary);
}
.ms-item-icon {
  flex-shrink: 0;
  font-size: 16px;
  color: var(--el-text-color-secondary);
}
.ms-item-name {
  flex-shrink: 0;
  font-size: 14px;
  color: var(--el-text-color-primary);
  max-width: 55%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ms-item-trail {
  flex: 1;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: right;
}
.ms-item mark {
  background: transparent;
  color: var(--el-color-primary);
  font-weight: 600;
}
.ms-more,
.ms-empty {
  padding: 18px 10px;
  text-align: center;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.ms-footer {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 8px 14px;
  border-top: 1px solid var(--el-border-color-lighter);
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.ms-footer kbd {
  display: inline-block;
  min-width: 18px;
  padding: 1px 5px;
  margin-right: 3px;
  text-align: center;
  border: 1px solid var(--el-border-color);
  border-bottom-width: 2px;
  border-radius: 4px;
  background: var(--el-fill-color-light);
  font-family: inherit;
  font-size: 11px;
}
.ms-count {
  margin-left: auto;
}
</style>

<style>
/* 弹窗骨架 (append-to-body 到文档根, scoped 作用域够不到): 隐藏默认头部、去 body 内边距 */
.menu-search-dialog {
  max-width: 92vw;
}
.menu-search-dialog .el-dialog__header {
  display: none;
}
.menu-search-dialog .el-dialog__body {
  padding: 0;
}
</style>
