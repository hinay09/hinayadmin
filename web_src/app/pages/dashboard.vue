<script setup lang="ts">
/**
 * 仪表盘 (Dashboard)
 * 视觉: 渐变欢迎区 + 迷你趋势 KPI 卡 + 全宽流量趋势 + 角色环图 + 模块排行 + 公告时间线。
 * 图表: ECharts Canvas 无法解析 CSS 变量 (var(--el-*) 会静默失效), 因此配色在客户端
 *      从 Element Plus CSS 变量读取后注入 option, 并随主题色/暗黑模式切换重新读取。
 * 数据: 当前为前端 mock; 后续可替换为后端聚合接口。
 */
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import {
  ArrowDown,
  ArrowUp,
  Avatar,
  Bell,
  Monitor,
  Setting,
  User,
  UserFilled,
  View,
} from '@element-plus/icons-vue'
import { useUserStore } from '~/stores/user'
import { useConfigStore } from '~/stores/config'
import { useSettingsStore } from '~/stores/settings'

definePageMeta({ title: '仪表盘' })
defineOptions({ name: 'dashboard' })

const router = useRouter()
const userStore = useUserStore()
const { userInfo } = storeToRefs(userStore)
const configStore = useConfigStore()
const { siteName } = storeToRefs(configStore)
const settings = useSettingsStore()
const { isDark } = storeToRefs(settings)

/* ---------------- 图表配色: 运行时读取 EP CSS 变量 ---------------- */
type Tone = 'primary' | 'success' | 'warning' | 'danger'

interface ChartPalette {
  primary: string
  success: string
  warning: string
  danger: string
  info: string
  textPrimary: string
  textSecondary: string
  border: string
  split: string
  tooltipBg: string
  cardBg: string
  track: string
}

const palette = ref<ChartPalette>({
  primary: '#409eff',
  success: '#67c23a',
  warning: '#e6a23c',
  danger: '#f56c6c',
  info: '#909399',
  textPrimary: '#303133',
  textSecondary: '#909399',
  border: '#ebeef5',
  split: '#f0f2f5',
  tooltipBg: '#ffffff',
  cardBg: '#ffffff',
  track: 'rgba(0,0,0,.04)',
})

/** 解析当前主题下的实际颜色 (暗黑模式 EP 会整体替换变量值) */
function resolvePalette() {
  if (!import.meta.client) return
  const cs = getComputedStyle(document.documentElement)
  const v = (name: string, fallback: string) => cs.getPropertyValue(name).trim() || fallback
  const dark = isDark.value
  palette.value = {
    primary: v('--el-color-primary', settings.theme || '#409eff'),
    success: v('--el-color-success', '#67c23a'),
    warning: v('--el-color-warning', '#e6a23c'),
    danger: v('--el-color-danger', '#f56c6c'),
    info: v('--el-color-info', '#909399'),
    textPrimary: v('--el-text-color-primary', dark ? '#e5eaf3' : '#303133'),
    textSecondary: v('--el-text-color-secondary', dark ? '#a3a6ad' : '#909399'),
    border: v('--el-border-color-lighter', dark ? 'rgba(255,255,255,.12)' : '#ebeef5'),
    split: dark ? 'rgba(255,255,255,.07)' : '#f0f2f5',
    tooltipBg: dark ? '#1d1e1f' : '#ffffff',
    cardBg: dark ? '#1d1e1f' : '#ffffff',
    track: dark ? 'rgba(255,255,255,.07)' : 'rgba(0,0,0,.04)',
  }
}

const toneColor = (tone: Tone) => palette.value[tone]

/** hex -> rgba (支持 3/6 位; 非 hex 原样返回) */
function alpha(color: string, a: number): string {
  const h = color.replace('#', '').trim()
  const full = h.length === 3 ? h.split('').map(c => c + c).join('') : h
  if (!/^[0-9a-fA-F]{6}$/.test(full)) return color
  const n = Number.parseInt(full, 16)
  return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${a})`
}

/** 纵向渐变 (面积图) */
const verticalGradient = (from: string, to: string) => ({
  type: 'linear',
  x: 0,
  y: 0,
  x2: 0,
  y2: 1,
  colorStops: [
    { offset: 0, color: from },
    { offset: 1, color: to },
  ],
})

/** 横向渐变 (条形图) */
const horizontalGradient = (from: string, to: string) => ({
  type: 'linear',
  x: 0,
  y: 0,
  x2: 1,
  y2: 0,
  colorStops: [
    { offset: 0, color: from },
    { offset: 1, color: to },
  ],
})

/** 统一 tooltip 外观 */
const tooltipStyle = () => {
  const p = palette.value
  return {
    backgroundColor: p.tooltipBg,
    borderColor: p.border,
    borderWidth: 1,
    padding: [10, 14] as [number, number],
    textStyle: { color: p.textPrimary, fontSize: 12 },
    extraCssText: 'border-radius:10px;box-shadow:0 12px 30px rgba(0,0,0,.14);',
  }
}

/* ---------------- KPI 概览 (mock) ---------------- */
interface Kpi {
  label: string
  value: number
  delta: number
  icon: any
  tone: Tone
  spark: number[]
}

const kpis: Kpi[] = [
  { label: '用户总数', value: 1286, delta: 12.5, icon: User, tone: 'primary', spark: [42, 58, 51, 72, 66, 84, 95] },
  { label: '在线用户', value: 87, delta: 4.2, icon: UserFilled, tone: 'success', spark: [56, 48, 62, 55, 71, 64, 78] },
  { label: '今日访问', value: 2240, delta: 6.4, icon: View, tone: 'warning', spark: [64, 58, 66, 61, 70, 68, 74] },
  { label: '公告总数', value: 36, delta: -2.0, icon: Bell, tone: 'danger', spark: [30, 36, 34, 45, 42, 38, 34] },
]

/* 数字入场动效: SSR/首次渲染用最终值 (无水合差异), 挂载后再从 0 递增 */
const displayValues = ref(kpis.map(k => k.value))

function runCountUp() {
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
  displayValues.value = kpis.map(() => 0)
  const start = performance.now()
  const duration = 900
  const tick = (now: number) => {
    const t = Math.min(1, (now - start) / duration)
    const eased = 1 - Math.pow(1 - t, 3)
    displayValues.value = kpis.map(k => Math.round(k.value * eased))
    if (t < 1) requestAnimationFrame(tick)
  }
  requestAnimationFrame(tick)
}

/** KPI 迷你趋势图: 无坐标轴的小面积线 */
function sparkOption(k: Kpi) {
  const c = toneColor(k.tone)
  return {
    animationDuration: 800,
    grid: { left: 0, right: 0, top: 6, bottom: 0 },
    xAxis: {
      type: 'category',
      show: false,
      boundaryGap: false,
      data: k.spark.map((_, i) => i),
    },
    yAxis: {
      type: 'value',
      show: false,
      min: Math.min(...k.spark) * 0.8,
      max: Math.max(...k.spark) * 1.05,
    },
    series: [
      {
        type: 'line',
        smooth: 0.4,
        showSymbol: false,
        silent: true,
        data: k.spark,
        lineStyle: { width: 2, color: c },
        areaStyle: { color: verticalGradient(alpha(c, 0.24), alpha(c, 0)) },
      },
    ],
  }
}

/* ---------------- 折线图: 访问趋势 (7 / 30 天) ---------------- */
const trendData: Record<'7' | '30', { uv: number[], pv: number[] }> = {
  '7': {
    uv: [477, 459, 486, 510, 494, 525, 612],
    pv: [1912, 1840, 1950, 2043, 1988, 2106, 2240],
  },
  '30': {
    uv: [348, 362, 351, 378, 402, 371, 356, 390, 375, 404, 432, 411, 398, 428, 452, 438, 420, 447, 475, 461, 443, 470, 489, 477, 459, 486, 510, 494, 525, 612],
    pv: [1320, 1456, 1390, 1512, 1604, 1488, 1420, 1560, 1495, 1612, 1726, 1650, 1588, 1704, 1812, 1756, 1682, 1795, 1902, 1846, 1778, 1884, 1962, 1912, 1840, 1950, 2043, 1988, 2106, 2240],
  },
}
const trendMode = ref<'7' | '30'>('7')

/** 近 N 天的 M/D 标签 (仅图表客户端渲染时使用) */
function lastDays(n: number): string[] {
  const out: string[] = []
  const today = new Date()
  for (let i = n - 1; i >= 0; i--) {
    const d = new Date(today)
    d.setDate(today.getDate() - i)
    out.push(`${d.getMonth() + 1}/${d.getDate()}`)
  }
  return out
}

const trendSummary = computed(() => {
  const d = trendData[trendMode.value]
  const total = d.pv.reduce((a, b) => a + b, 0)
  return {
    total,
    avg: Math.round(total / d.pv.length),
    peak: Math.max(...d.pv),
  }
})

const trendOption = computed(() => {
  const p = palette.value
  const d = trendData[trendMode.value]
  const labels = lastDays(d.pv.length)

  const lineSeries = (name: string, data: number[], color: string, axisIndex: number, areaAlpha: number) => ({
    name,
    type: 'line',
    smooth: 0.35,
    showSymbol: false,
    symbol: 'circle',
    symbolSize: 8,
    yAxisIndex: axisIndex,
    data,
    lineStyle: {
      width: 3,
      color,
      shadowColor: alpha(color, 0.35),
      shadowBlur: 14,
      shadowOffsetY: 8,
    },
    itemStyle: { color, borderColor: p.cardBg, borderWidth: 2 },
    areaStyle: { color: verticalGradient(alpha(color, areaAlpha), alpha(color, 0)) },
    emphasis: { focus: 'series', scale: 1.6 },
  })

  return {
    animationDuration: 700,
    animationEasing: 'cubicOut',
    grid: { left: 6, right: 14, top: 26, bottom: 4, containLabel: true },
    tooltip: {
      trigger: 'axis',
      ...tooltipStyle(),
      axisPointer: {
        type: 'line',
        lineStyle: { color: alpha(p.primary, 0.35), type: 'dashed' },
      },
      formatter: (params: any[]) => {
        const rows = params.map((item) => {
          const value = typeof item.value === 'number' ? item.value.toLocaleString() : item.value
          return `<div style="display:flex;align-items:center;gap:8px;margin-top:6px">
            <span style="width:8px;height:8px;border-radius:50%;background:${item.color};display:inline-block"></span>
            <span style="color:${p.textSecondary}">${item.seriesName}</span>
            <strong style="margin-left:auto;padding-left:18px;font-variant-numeric:tabular-nums">${value}</strong>
          </div>`
        }).join('')
        return `<div style="font-size:12px;font-weight:600">${params[0]?.axisValue ?? ''}</div>${rows}`
      },
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: labels,
      axisLine: { lineStyle: { color: p.border } },
      axisTick: { show: false },
      axisLabel: {
        color: p.textSecondary,
        fontSize: 12,
        margin: 14,
        hideOverlap: true,
      },
    },
    yAxis: [
      {
        type: 'value',
        splitNumber: 4,
        splitLine: { lineStyle: { color: p.split, type: 'dashed' } },
        axisLabel: { color: alpha(p.primary, 0.8), fontSize: 12 },
      },
      {
        type: 'value',
        position: 'right',
        splitNumber: 4,
        splitLine: { show: false },
        axisLabel: { color: alpha(p.success, 0.8), fontSize: 12 },
      },
    ],
    series: [
      lineSeries('访问量 PV', d.pv, p.primary, 0, 0.22),
      lineSeries('访客数 UV', d.uv, p.success, 1, 0.1),
    ],
  }
})

/* ---------------- 环形图: 角色分布 ---------------- */
interface RoleItem {
  name: string
  value: number
  tone: Tone
}
const roleData: RoleItem[] = [
  { name: '管理员', value: 24, tone: 'primary' },
  { name: '普通用户', value: 386, tone: 'success' },
  { name: '访客', value: 876, tone: 'warning' },
]
const roleTotal = computed(() => roleData.reduce((a, b) => a + b.value, 0))

const roleOption = computed(() => {
  const p = palette.value
  return {
    animationDuration: 800,
    animationType: 'scale',
    animationEasing: 'quarticOut',
    tooltip: {
      trigger: 'item',
      ...tooltipStyle(),
      formatter: (item: any) =>
        `<div style="display:flex;align-items:center;gap:8px">
          <span style="width:8px;height:8px;border-radius:50%;background:${item.color};display:inline-block"></span>
          <span style="color:${p.textSecondary}">${item.name}</span>
        </div>
        <div style="margin-top:6px;font-size:16px;font-weight:600;font-variant-numeric:tabular-nums">
          ${Number(item.value).toLocaleString()}
          <span style="font-size:12px;font-weight:400;color:${p.textSecondary}"> 个账号 · ${item.percent}%</span>
        </div>`,
    },
    series: [
      {
        name: '角色分布',
        type: 'pie',
        radius: ['64%', '84%'],
        center: ['50%', '50%'],
        padAngle: 3,
        avoidLabelOverlap: true,
        label: { show: false },
        labelLine: { show: false },
        itemStyle: {
          borderRadius: 7,
          borderColor: p.cardBg,
          borderWidth: 2,
        },
        emphasis: {
          scale: true,
          scaleSize: 5,
          itemStyle: {
            shadowBlur: 18,
            shadowColor: alpha(p.primary, 0.28),
          },
        },
        data: roleData.map(r => ({
          name: r.name,
          value: r.value,
          itemStyle: { color: toneColor(r.tone) },
        })),
      },
    ],
  }
})

/* ---------------- 横向条形图: 业务模块调用 ---------------- */
const moduleData = [
  { name: 'API 资源', value: 2410 },
  { name: '用户', value: 1820 },
  { name: '角色', value: 642 },
  { name: '公告', value: 580 },
  { name: '菜单', value: 318 },
]

const moduleOption = computed(() => {
  const p = palette.value
  return {
    animationDuration: 800,
    animationEasing: 'cubicOut',
    grid: { left: 4, right: 52, top: 8, bottom: 4, containLabel: true },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow', shadowStyle: { color: p.track } },
      ...tooltipStyle(),
      formatter: (params: any[]) => {
        const item = params[0]
        return `<div style="font-size:12px;color:${p.textSecondary}">${item.name}</div>
          <div style="margin-top:4px;font-size:16px;font-weight:600;font-variant-numeric:tabular-nums">
            ${Number(item.value).toLocaleString()}
            <span style="font-size:12px;font-weight:400;color:${p.textSecondary}"> 次调用</span>
          </div>`
      },
    },
    xAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: p.split, type: 'dashed' } },
      axisLabel: { color: p.textSecondary, fontSize: 11 },
    },
    yAxis: {
      type: 'category',
      inverse: true,
      data: moduleData.map(m => m.name),
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { color: p.textPrimary, fontSize: 13, margin: 12 },
    },
    series: [
      {
        type: 'bar',
        barWidth: 14,
        showBackground: true,
        backgroundStyle: { color: p.track, borderRadius: 8 },
        itemStyle: {
          borderRadius: 8,
          color: horizontalGradient(p.primary, alpha(p.primary, 0.45)),
        },
        label: {
          show: true,
          position: 'right',
          distance: 10,
          color: p.textSecondary,
          fontSize: 12,
          formatter: (item: any) => Number(item.value).toLocaleString(),
        },
        data: moduleData.map(m => m.value),
      },
    ],
  }
})

/* ---------------- 最近公告 (mock) ---------------- */
interface NoticeItem {
  id: number
  title: string
  level: 1 | 2 | 3
  time: string
}
const recentNotices: NoticeItem[] = [
  { id: 1, title: '系统于本周日凌晨 02:00 进行例行维护', level: 3, time: '11-18 09:12' },
  { id: 2, title: 'v2.4.0 版本已发布: 新增 API 资源管理', level: 2, time: '11-16 17:30' },
  { id: 3, title: '请所有同事在月底前完成账号信息核对', level: 2, time: '11-14 10:05' },
  { id: 4, title: '欢迎 3 位新成员加入研发团队', level: 1, time: '11-12 14:48' },
  { id: 5, title: '关于优化登录速度的若干说明', level: 1, time: '11-10 11:20' },
]
const levelText = (l: number) => (l === 3 ? '紧急' : l === 2 ? '重要' : '普通')

/* ---------------- 欢迎区 ---------------- */
const greeting = ref('你好')
const dayText = ref('--')
const dateMeta = reactive({ month: '', weekday: '' })
const WEEKDAYS = ['星期日', '星期一', '星期二', '星期三', '星期四', '星期五', '星期六']

const quickActions = [
  { label: '用户管理', path: '/system/users', icon: User },
  { label: '角色权限', path: '/system/roles', icon: Setting },
  { label: '系统监控', path: '/system/monitor', icon: Monitor },
  { label: '个人中心', path: '/profile', icon: Avatar },
]

onMounted(() => {
  resolvePalette()

  const now = new Date()
  dayText.value = String(now.getDate()).padStart(2, '0')
  dateMeta.month = `${now.getFullYear()} 年 ${now.getMonth() + 1} 月`
  dateMeta.weekday = WEEKDAYS[now.getDay()]
  const hour = now.getHours()
  greeting.value = hour < 6 ? '夜深了' : hour < 9 ? '早上好' : hour < 12 ? '上午好' : hour < 14 ? '中午好' : hour < 18 ? '下午好' : '晚上好'

  runCountUp()
})

/* 主题色 / 暗黑切换后重取图表配色 (等 DOM class 与变量生效) */
watch([() => settings.theme, isDark], () => nextTick(resolvePalette))
</script>

<template>
  <div class="dashboard">
    <!-- 欢迎区 -->
    <section class="hero" :class="{ 'is-dark': isDark }">
      <div class="hero-grid" aria-hidden="true" />
      <div class="hero-body">
        <div class="hero-eyebrow">WORKSPACE · 工作台</div>
        <h1 class="hero-title">
          {{ greeting }}，{{ userInfo?.nickname || userInfo?.username || 'Admin' }}
          <span class="hero-wave">👋</span>
        </h1>
        <p class="hero-sub">欢迎使用 {{ siteName }}，今天也要高效地推进每一件事。</p>
        <div class="hero-actions">
          <button
            v-for="a in quickActions"
            :key="a.path"
            type="button"
            class="hero-btn"
            @click="router.push(a.path)"
          >
            <el-icon :size="14"><component :is="a.icon" /></el-icon>
            {{ a.label }}
          </button>
        </div>
      </div>
      <div class="hero-date">
        <div class="hero-day">{{ dayText }}</div>
        <div class="hero-date-meta">
          <span>{{ dateMeta.month }}</span>
          <span>{{ dateMeta.weekday }}</span>
        </div>
      </div>
    </section>

    <!-- KPI 卡片 -->
    <el-row :gutter="16" class="row">
      <el-col v-for="(k, i) in kpis" :key="k.label" :xs="24" :sm="12" :lg="6">
        <article
          class="kpi-card"
          :style="{
            '--kpi': toneColor(k.tone),
            '--kpi-soft': alpha(toneColor(k.tone), 0.12),
            '--kpi-glow': alpha(toneColor(k.tone), 0.16),
          }"
        >
          <div class="kpi-top">
            <span class="kpi-label">{{ k.label }}</span>
            <span class="kpi-icon">
              <el-icon :size="18"><component :is="k.icon" /></el-icon>
            </span>
          </div>
          <div class="kpi-value">{{ displayValues[i].toLocaleString() }}</div>
          <div class="kpi-foot">
            <span class="kpi-delta" :class="k.delta >= 0 ? 'up' : 'down'">
              <el-icon :size="12"><component :is="k.delta >= 0 ? ArrowUp : ArrowDown" /></el-icon>
              {{ Math.abs(k.delta).toFixed(1) }}%
            </span>
            <span class="kpi-tip">较上周</span>
          </div>
          <div class="kpi-spark">
            <ClientOnly>
              <v-chart :option="sparkOption(k)" autoresize class="spark-chart" />
            </ClientOnly>
          </div>
        </article>
      </el-col>
    </el-row>

    <!-- 访问趋势 -->
    <section class="panel trend-panel">
      <header class="panel-head">
        <div class="panel-heading">
          <h2 class="panel-title">访问趋势</h2>
          <p class="panel-sub">全站流量概览 · PV / UV</p>
        </div>
        <div class="panel-tools">
          <div class="legend-chips">
            <span class="legend-chip"><i :style="{ background: palette.primary }" />访问量 PV</span>
            <span class="legend-chip"><i :style="{ background: palette.success }" />访客数 UV</span>
          </div>
          <div class="panel-stats">
            <div class="pstat">
              <span class="pstat-label">周期访问</span>
              <span class="pstat-value">{{ trendSummary.total.toLocaleString() }}</span>
            </div>
            <div class="pstat">
              <span class="pstat-label">日均</span>
              <span class="pstat-value">{{ trendSummary.avg.toLocaleString() }}</span>
            </div>
            <div class="pstat">
              <span class="pstat-label">峰值</span>
              <span class="pstat-value">{{ trendSummary.peak.toLocaleString() }}</span>
            </div>
          </div>
          <div class="seg">
            <button type="button" :class="{ active: trendMode === '7' }" @click="trendMode = '7'">近 7 天</button>
            <button type="button" :class="{ active: trendMode === '30' }" @click="trendMode = '30'">近 30 天</button>
          </div>
        </div>
      </header>
      <ClientOnly>
        <v-chart :option="trendOption" autoresize class="trend-chart" />
        <template #fallback>
          <div class="chart-skeleton trend-chart" />
        </template>
      </ClientOnly>
    </section>

    <!-- 角色分布 / 模块排行 / 最近公告 -->
    <el-row :gutter="16" class="row">
      <el-col :xs="24" :sm="24" :md="24" :lg="8">
        <section class="panel side-panel">
          <header class="panel-head">
            <div class="panel-heading">
              <h2 class="panel-title">角色分布</h2>
              <p class="panel-sub">账号构成</p>
            </div>
          </header>
          <ClientOnly>
            <div class="donut-wrap">
              <v-chart :option="roleOption" autoresize class="donut-chart" />
              <div class="donut-center">
                <div class="donut-total">{{ roleTotal.toLocaleString() }}</div>
                <div class="donut-label">账号总数</div>
              </div>
            </div>
            <template #fallback>
              <div class="chart-skeleton donut-chart" />
            </template>
          </ClientOnly>
          <ul class="role-list">
            <li v-for="r in roleData" :key="r.name" class="role-item">
              <div class="role-row">
                <span class="role-dot" :style="{ background: toneColor(r.tone) }" />
                <span class="role-name">{{ r.name }}</span>
                <span class="role-value">{{ r.value.toLocaleString() }}</span>
                <span class="role-pct">{{ ((r.value / roleTotal) * 100).toFixed(1) }}%</span>
              </div>
              <div class="role-track">
                <span
                  class="role-fill"
                  :style="{ width: `${(r.value / roleTotal) * 100}%`, background: toneColor(r.tone) }"
                />
              </div>
            </li>
          </ul>
        </section>
      </el-col>

      <el-col :xs="24" :sm="24" :md="24" :lg="8">
        <section class="panel side-panel">
          <header class="panel-head">
            <div class="panel-heading">
              <h2 class="panel-title">业务模块调用</h2>
              <p class="panel-sub">近 7 天 · TOP 5</p>
            </div>
          </header>
          <ClientOnly>
            <v-chart :option="moduleOption" autoresize class="rank-chart" />
            <template #fallback>
              <div class="chart-skeleton rank-chart" />
            </template>
          </ClientOnly>
        </section>
      </el-col>

      <el-col :xs="24" :sm="24" :md="24" :lg="8">
        <section class="panel side-panel">
          <header class="panel-head">
            <div class="panel-heading">
              <h2 class="panel-title">最近公告</h2>
              <p class="panel-sub">系统通知与动态</p>
            </div>
          </header>
          <ul class="notice-list">
            <li v-for="n in recentNotices" :key="n.id" class="notice-item">
              <span class="notice-dot" :class="`lv-${n.level}`" />
              <div class="notice-main">
                <span class="notice-title">{{ n.title }}</span>
                <span class="notice-meta">
                  <span class="notice-level" :class="`lv-${n.level}`">{{ levelText(n.level) }}</span>
                  <span>{{ n.time }}</span>
                </span>
              </div>
            </li>
          </ul>
        </section>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
}

.row {
  margin-top: 16px;
}

/* ---------- 欢迎区 ---------- */
.hero {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 28px;
  padding: 26px 30px;
  border-radius: 18px;
  overflow: hidden;
  color: #fff;
  background:
    radial-gradient(900px 320px at 88% -30%, rgba(255, 255, 255, 0.3), transparent 62%),
    radial-gradient(420px 260px at 12% 120%, rgba(255, 255, 255, 0.14), transparent 70%),
    linear-gradient(118deg, var(--el-color-primary-dark-2) 0%, var(--el-color-primary) 62%, var(--el-color-primary-light-3) 100%);
  box-shadow: 0 18px 42px -26px rgba(0, 0, 0, 0.55);
}

/* 暗黑模式: 压低白色高光, 让渐变回到深色底, 避免整块发白 */
.hero.is-dark {
  background:
    radial-gradient(900px 320px at 88% -30%, rgba(255, 255, 255, 0.12), transparent 62%),
    radial-gradient(420px 260px at 12% 120%, rgba(255, 255, 255, 0.06), transparent 70%),
    linear-gradient(118deg, var(--el-color-primary-dark-2) 0%, var(--el-color-primary) 78%);
}

/* 细网格纹理, 只铺左上角, 增加科技感 */
.hero-grid {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.07) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.07) 1px, transparent 1px);
  background-size: 38px 38px;
  mask-image: radial-gradient(620px 260px at 18% 0%, #000 18%, transparent 76%);
  -webkit-mask-image: radial-gradient(620px 260px at 18% 0%, #000 18%, transparent 76%);
}

.hero-body {
  position: relative;
  z-index: 1;
  min-width: 0;
}

.hero-eyebrow {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 2.4px;
  opacity: 0.75;
}

.hero-eyebrow::before {
  content: '';
  width: 18px;
  height: 1px;
  background: rgba(255, 255, 255, 0.8);
}

.hero-title {
  margin: 10px 0 6px;
  font-size: 26px;
  font-weight: 700;
  line-height: 1.25;
  letter-spacing: 0.2px;
}

.hero-wave {
  display: inline-block;
  margin-left: 2px;
  transform-origin: 70% 70%;
  animation: wave 2.4s ease-in-out 0.6s 1;
}

@keyframes wave {
  0%, 100% { transform: rotate(0deg); }
  20% { transform: rotate(14deg); }
  40% { transform: rotate(-8deg); }
  60% { transform: rotate(10deg); }
  80% { transform: rotate(-4deg); }
}

.hero-sub {
  margin: 0;
  font-size: 13px;
  opacity: 0.8;
}

.hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 20px;
}

.hero-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 14px;
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.24);
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
  font-size: 12.5px;
  font-family: inherit;
  cursor: pointer;
  backdrop-filter: blur(8px);
  transition: background 0.2s ease, border-color 0.2s ease, transform 0.2s ease;
}

.hero-btn:hover {
  background: rgba(255, 255, 255, 0.22);
  border-color: rgba(255, 255, 255, 0.4);
  transform: translateY(-1px);
}

.hero-date {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 14px;
  flex-shrink: 0;
  padding: 14px 20px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.13);
  border: 1px solid rgba(255, 255, 255, 0.22);
  backdrop-filter: blur(10px);
}

.hero-day {
  font-size: 38px;
  font-weight: 700;
  line-height: 1;
  letter-spacing: -1px;
  font-variant-numeric: tabular-nums;
}

.hero-date-meta {
  display: flex;
  flex-direction: column;
  gap: 5px;
  font-size: 12px;
  opacity: 0.9;
  white-space: nowrap;
}

/* ---------- KPI 卡片 ---------- */
.kpi-card {
  position: relative;
  height: 100%;
  padding: 16px 18px 0;
  border-radius: 16px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
  overflow: hidden;
  margin-bottom: 16px;
  transition: transform 0.22s ease, box-shadow 0.22s ease, border-color 0.22s ease;
}

.kpi-card::before {
  content: '';
  position: absolute;
  top: -46px;
  right: -34px;
  width: 130px;
  height: 130px;
  border-radius: 50%;
  pointer-events: none;
  background: radial-gradient(circle, var(--kpi-glow), transparent 70%);
}

.kpi-card:hover {
  transform: translateY(-3px);
  border-color: var(--el-border-color-light);
  box-shadow: 0 18px 34px -24px rgba(0, 0, 0, 0.4);
}

.kpi-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.kpi-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.kpi-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border-radius: 11px;
  color: var(--kpi);
  background: var(--kpi-soft);
}

.kpi-value {
  margin-top: 8px;
  font-size: 30px;
  font-weight: 700;
  line-height: 1.15;
  letter-spacing: -0.6px;
  color: var(--el-text-color-primary);
  font-variant-numeric: tabular-nums;
}

.kpi-foot {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 6px;
  font-size: 12px;
}

.kpi-delta {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 1px 7px 1px 5px;
  border-radius: 999px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.kpi-delta.up {
  color: var(--el-color-success);
  background: var(--el-color-success-light-9);
}

.kpi-delta.down {
  color: var(--el-color-danger);
  background: var(--el-color-danger-light-9);
}

.kpi-tip {
  color: var(--el-text-color-secondary);
}

.kpi-spark {
  margin: 6px -18px 0;
}

.spark-chart {
  height: 46px;
  width: 100%;
}

/* ---------- 面板通用 ---------- */
.panel {
  position: relative;
  height: 100%;
  box-sizing: border-box;
  padding: 18px 20px;
  border-radius: 16px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
  margin-bottom: 16px;
  transition: box-shadow 0.25s ease, border-color 0.25s ease;
}

.panel:hover {
  border-color: var(--el-border-color-light);
  box-shadow: 0 18px 38px -28px rgba(0, 0, 0, 0.45);
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.panel-title::before {
  content: '';
  width: 4px;
  height: 14px;
  border-radius: 2px;
  background: linear-gradient(180deg, var(--el-color-primary), var(--el-color-primary-light-3));
}

.panel-sub {
  margin: 5px 0 0 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.panel-tools {
  display: flex;
  align-items: center;
  gap: 18px;
  flex-wrap: wrap;
}

.panel-stats {
  display: flex;
  gap: 18px;
}

.pstat {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.pstat-label {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

.pstat-value {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  font-variant-numeric: tabular-nums;
}

.legend-chips {
  display: flex;
  align-items: center;
  gap: 14px;
}

.legend-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.legend-chip i {
  width: 8px;
  height: 8px;
  border-radius: 3px;
}

.seg {
  display: inline-flex;
  gap: 2px;
  padding: 3px;
  border-radius: 9px;
  background: var(--el-fill-color-light);
}

.seg button {
  border: 0;
  padding: 5px 12px;
  border-radius: 7px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-family: inherit;
  cursor: pointer;
  transition: background 0.18s ease, color 0.18s ease, box-shadow 0.18s ease;
}

.seg button.active {
  background: var(--el-bg-color);
  color: var(--el-color-primary);
  font-weight: 600;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
}

.trend-chart {
  width: 100%;
  height: 330px;
  margin-top: 10px;
}

.trend-panel {
  /* 与上方 KPI 卡片拉开呼吸感: 相邻外边距会折叠, 这里给足 24px 上层间距 */
  margin-top: 24px;
}

.side-panel {
  display: flex;
  flex-direction: column;
}

/* ---------- 角色环形图 ---------- */
.donut-wrap {
  position: relative;
  margin-top: 4px;
}

.donut-chart {
  width: 100%;
  height: 196px;
}

.donut-center {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  text-align: center;
  pointer-events: none;
}

.donut-total {
  font-size: 28px;
  font-weight: 700;
  line-height: 1.1;
  letter-spacing: -0.6px;
  color: var(--el-text-color-primary);
  font-variant-numeric: tabular-nums;
}

.donut-label {
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.role-list {
  display: flex;
  flex-direction: column;
  gap: 13px;
  list-style: none;
  padding: 0;
  margin: 8px 0 0;
}

.role-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.role-dot {
  width: 9px;
  height: 9px;
  border-radius: 3px;
  flex-shrink: 0;
}

.role-name {
  color: var(--el-text-color-regular);
}

.role-value {
  margin-left: auto;
  font-weight: 600;
  color: var(--el-text-color-primary);
  font-variant-numeric: tabular-nums;
}

.role-pct {
  width: 46px;
  text-align: right;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}

.role-track {
  height: 5px;
  margin-top: 6px;
  border-radius: 3px;
  background: var(--el-fill-color-light);
  overflow: hidden;
}

.role-fill {
  display: block;
  height: 100%;
  border-radius: 3px;
  transition: width 0.6s cubic-bezier(0.22, 1, 0.36, 1);
}

/* ---------- 模块排行 ---------- */
.rank-chart {
  width: 100%;
  height: 252px;
  margin-top: 12px;
}

/* ---------- 公告时间线 ---------- */
.notice-list {
  list-style: none;
  padding: 0;
  margin: 8px 0 0;
}

.notice-item {
  display: flex;
  gap: 12px;
  padding: 11px 10px;
  margin: 0 -10px;
  border-radius: 10px;
  transition: background 0.18s ease;
}

.notice-item:hover {
  background: var(--el-fill-color-lighter);
}

.notice-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-top: 7px;
  flex-shrink: 0;
  box-shadow: 0 0 0 3px var(--el-fill-color-lighter);
}

.lv-1 {
  background: var(--el-color-info);
}

.lv-2 {
  background: var(--el-color-warning);
}

.lv-3 {
  background: var(--el-color-danger);
}

.notice-main {
  flex: 1;
  min-width: 0;
}

.notice-title {
  display: block;
  font-size: 13.5px;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notice-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 5px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}

.notice-level.lv-1 {
  background: var(--el-color-info-light-9);
  color: var(--el-color-info);
}

.notice-level.lv-2 {
  background: var(--el-color-warning-light-9);
  color: var(--el-color-warning);
}

.notice-level.lv-3 {
  background: var(--el-color-danger-light-9);
  color: var(--el-color-danger);
}

.notice-level {
  padding: 0 6px;
  border-radius: 5px;
  font-size: 11px;
  line-height: 18px;
}

/* ---------- 图表骨架 ---------- */
.chart-skeleton {
  border-radius: 12px;
  background: linear-gradient(
    100deg,
    var(--el-fill-color-lighter) 25%,
    var(--el-fill-color-light) 37%,
    var(--el-fill-color-lighter) 63%
  );
  background-size: 400% 100%;
  animation: shimmer 1.4s ease infinite;
}

@keyframes shimmer {
  0% { background-position: 100% 50%; }
  100% { background-position: 0 50%; }
}

/* ---------- 响应式 ---------- */
@media (max-width: 1200px) {
  .panel-stats {
    display: none;
  }
}

@media (max-width: 768px) {
  .hero {
    flex-direction: column;
    align-items: flex-start;
    gap: 18px;
    padding: 22px 20px;
  }

  .hero-title {
    font-size: 22px;
  }

  .hero-date {
    padding: 10px 16px;
  }

  .hero-day {
    font-size: 30px;
  }

  .panel {
    padding: 16px;
  }

  .panel-head {
    gap: 10px;
  }

  .trend-chart {
    height: 280px;
  }

  .donut-chart {
    height: 186px;
  }

  .notice-title {
    white-space: normal;
  }
}

@media (prefers-reduced-motion: reduce) {
  .hero-wave,
  .chart-skeleton {
    animation: none;
  }

  .kpi-card,
  .hero-btn,
  .role-fill {
    transition: none;
  }
}
</style>
