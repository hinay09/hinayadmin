/**
 * 多标签页 store: 记录已访问页面 (TagsBar 展示) 与 KeepAlive 缓存名单。
 *
 * 约定: 页面组件通过 defineOptions({ name }) 显式命名且与路由名一致
 * (如 pages/system/users/index.vue -> 'system-users'), KeepAlive 的 include
 * 按 name 匹配; 未命名的页面正常渲染但不缓存。
 */
import { defineStore } from 'pinia'

export interface TagView {
  /** 路由 path (去 query), 标签唯一标识 */
  path: string
  /** 页面标题 (definePageMeta title) */
  title: string
  /** 组件/路由名, KeepAlive 缓存键 */
  name: string
  /** 固定标签 (不可关闭), 如仪表盘 */
  affix?: boolean
}

export const useTagsStore = defineStore('tags', {
  state: () => ({
    visited: [] as TagView[],
    /** KeepAlive include 名单 (与 visited 同步, 关闭标签即出缓存) */
    cachedNames: [] as string[],
    /** 页面刷新种子: path -> 计数, 变化即强制重建组件 */
    refreshSeed: {} as Record<string, number>,
  }),

  actions: {
    /** 路由变化时登记标签 (登录页/无标题页跳过) */
    addTag(route: { path: string; meta?: { title?: string }; name?: string | symbol }) {
      const title = route.meta?.title
      const name = typeof route.name === 'string' ? route.name : ''
      if (!title || !name || route.path === '/login') return
      if (this.visited.some(t => t.path === route.path)) return
      this.visited.push({
        path: route.path,
        title,
        name,
        affix: route.path === '/dashboard',
      })
      if (!this.cachedNames.includes(name)) this.cachedNames.push(name)
    },

    /**
     * 关闭标签; 若关闭的是当前激活标签, 返回应跳转的相邻标签 path。
     */
    removeTag(path: string): string | undefined {
      const idx = this.visited.findIndex(t => t.path === path)
      if (idx < 0) return undefined
      const tag = this.visited[idx]
      if (tag.affix) return undefined
      this.visited.splice(idx, 1)
      this.delCached(tag.name)
      delete this.refreshSeed[tag.path]
      // 由调用方比对当前路由决定是否跳转
      const neighbor = this.visited[idx] || this.visited[idx - 1]
      return neighbor?.path
    },

    /** 关闭其他标签 (保留固定标签与指定标签) */
    closeOthers(path: string) {
      this.visited = this.visited.filter(t => t.affix || t.path === path)
      this.syncCache()
    },

    /** 关闭全部标签 (保留固定标签), 返回应跳转的固定标签 path */
    closeAll(): string | undefined {
      this.visited = this.visited.filter(t => t.affix)
      this.syncCache()
      return this.visited[0]?.path
    },

    /** 从缓存名单移除 (刷新页面用, 之后再加回触发重建) */
    delCached(name: string) {
      this.cachedNames = this.cachedNames.filter(n => n !== name)
    },

    /** 重新加入缓存名单 */
    addCached(name: string) {
      if (name && !this.cachedNames.includes(name)) this.cachedNames.push(name)
    },

    /** 刷新指定页面: 先出缓存 + 换 key, 组件销毁重建后再回缓存 */
    bumpRefresh(path: string, name: string) {
      this.delCached(name)
      this.refreshSeed = { ...this.refreshSeed, [path]: (this.refreshSeed[path] || 0) + 1 }
    },

    /** 按 visited 重建缓存名单 */
    syncCache() {
      this.cachedNames = this.visited.map(t => t.name)
    },
  },
})
