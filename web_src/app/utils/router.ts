/**
 * 菜单/路由相关工具
 */
import type { MenuNode } from '~/stores/user'

/**
 * 将扁平菜单数组转换为树
 */
export function buildMenuTree(list: MenuNode[]): MenuNode[] {
  const map = new Map<number, MenuNode>()
  const roots: MenuNode[] = []
  for (const item of list) {
    map.set(item.id, { ...item, children: [] })
  }
  for (const item of list) {
    const node = map.get(item.id)!
    if (item.parentId && map.has(item.parentId)) {
      const parent = map.get(item.parentId)!
      parent.children = parent.children || []
      parent.children.push(node)
    }
    else {
      roots.push(node)
    }
  }
  return roots
}

/**
 * 仅保留可显示菜单 (type=1 目录, type=2 菜单), 过滤按钮 (type=3)
 */
export function filterDisplayMenus(tree: MenuNode[]): MenuNode[] {
  const walk = (arr: MenuNode[]): MenuNode[] => {
    return arr
      .filter(n => n.type === 1 || n.type === 2)
      .map(n => ({
        ...n,
        children: n.children ? walk(n.children) : [],
      }))
  }
  return walk(tree)
}

/**
 * 提取所有可点击菜单 (type=2) 的 path 列表, 用于 Tab 等
 */
export function flattenLeafPaths(tree: MenuNode[]): string[] {
  const out: string[] = []
  const walk = (arr: MenuNode[]) => {
    for (const n of arr) {
      if (n.type === 2 && n.path) out.push(n.path)
      if (n.children?.length) walk(n.children)
    }
  }
  walk(tree)
  return out
}
