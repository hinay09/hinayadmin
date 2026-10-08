/**
 * 主题工具 (仿 RuoYi 主题机制): 运行时重写 Element Plus 主色 CSS 变量 + 暗黑模式切换。
 *
 * Element Plus 2.x 配色全部走 CSS 变量: --el-color-primary 及派生色
 * light-3..9(与底色混合)、dark-2(与黑混合)。换主题色只需在 <html> 上重写这些变量;
 * 暗黑模式下 light-N 按官方 dark/css-vars.css 的口径改为与暗底 #141414 混合。
 * 暗黑模式本身由 nuxt.config 引入的 element-plus/theme-chalk/dark/css-vars.css
 * 驱动, 这里只负责切换 <html> 的 dark 类。
 */

/** 暗黑模式下 light 系列的混合底色 (与 EP 官方暗黑变量口径一致) */
const DARK_MIX_BASE = '#141414'
const LIGHT_MIX_BASE = '#ffffff'

/** 校验 6 位 hex 颜色 */
export function isHexColor(v: string): boolean {
  return /^#[0-9a-fA-F]{6}$/.test(v)
}

/** hex 加权混合: weight 为 c2 所占比例 (0-1), 返回 6 位 hex */
export function mixColor(c1: string, c2: string, weight: number): string {
  const parse = (hex: string) => {
    const n = hex.replace('#', '')
    return [0, 2, 4].map(i => Number.parseInt(n.slice(i, i + 2), 16))
  }
  const [r1, g1, b1] = parse(c1)
  const [r2, g2, b2] = parse(c2)
  const mix = (a: number, b: number) => Math.round(a + (b - a) * weight)
  const toHex = (n: number) => n.toString(16).padStart(2, '0')
  return `#${toHex(mix(r1, r2))}${toHex(mix(g1, g2))}${toHex(mix(b1, b2))}`
}

/** 生成主色及其派生色变量表 */
export function themeVars(color: string, isDark: boolean): Record<string, string> {
  const vars: Record<string, string> = { '--el-color-primary': color }
  const base = isDark ? DARK_MIX_BASE : LIGHT_MIX_BASE
  for (let level = 3; level <= 9; level++) {
    vars[`--el-color-primary-light-${level}`] = mixColor(color, base, level / 10)
  }
  vars['--el-color-primary-dark-2'] = mixColor(color, '#000000', 0.2)
  return vars
}

/** 将主题色写入 <html> (仅客户端) */
export function applyTheme(color: string, isDark: boolean): void {
  if (!import.meta.client || !isHexColor(color)) return
  const style = document.documentElement.style
  for (const [k, v] of Object.entries(themeVars(color, isDark))) {
    style.setProperty(k, v)
  }
}

/** 切换暗黑模式: <html> 加/去 dark 类, 并同步 colorScheme 让原生控件/滚动条跟随 */
export function applyDark(isDark: boolean): void {
  if (!import.meta.client) return
  document.documentElement.classList.toggle('dark', isDark)
  document.documentElement.style.colorScheme = isDark ? 'dark' : 'light'
}
