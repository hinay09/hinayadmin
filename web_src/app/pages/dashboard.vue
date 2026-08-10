<script setup lang="ts">
/**
 * 仪表盘 (Dashboard)
 * - KPI 概览卡片(4)
 * - 访问趋势折线图 + 角色分布饼图
 * - 业务模块调用柱状图 + 最近公告列表
 * 数据来源: 当前为前端 mock; 后续可替换为后端聚合接口。
 */
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import {
  User,
  UserFilled,
  View,
  Bell,
  ArrowUp,
  ArrowDown,
  TrendCharts,
  PieChart,
  DataAnalysis,
  Calendar,
} from '@element-plus/icons-vue'
import { useUserStore } from '~/stores/user'

definePageMeta({ title: '仪表盘' })

const userStore = useUserStore()
const { userInfo } = storeToRefs(userStore)

/* ---------------- KPI 概览 (mock) ---------------- */
interface Kpi {
  label: string
  value: number
  unit?: string
  delta: number
  icon: any
  gradient: string
}
const kpis: Kpi[] = [
  { label: '用户总数', value: 1286, delta: 12.5, icon: User, gradient: 'linear-gradient(135deg,#409EFF,#1890ff)' },
  { label: '在线用户', value: 87, delta: 4.2, icon: UserFilled, gradient: 'linear-gradient(135deg,#67C23A,#3aa45a)' },
  { label: '今日访问', value: 5320, delta: -2.8, icon: View, gradient: 'linear-gradient(135deg,#E6A23C,#d48806)' },
  { label: '公告总数', value: 36, delta: 8.0, icon: Bell, gradient: 'linear-gradient(135deg,#F56C6C,#cf1322)' },
]

/* ---------------- 折线图: 近 7 天访问趋势 ---------------- */
const last7Days = (() => {
  const days: string[] = []
  const today = new Date()
  for (let i = 6; i >= 0; i--) {
    const d = new Date(today)
    d.setDate(today.getDate() - i)
    days.push(`${d.getMonth() + 1}/${d.getDate()}`)
  }
  return days
})()
const uvData = [320, 412, 388, 502, 478, 561, 612]
const pvData = [1280, 1420, 1356, 1820, 1742, 2041, 2240]

const trendOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  legend: { data: ['UV', 'PV'], right: 12, top: 6 },
  grid: { left: 36, right: 24, top: 40, bottom: 28 },
  xAxis: {
    type: 'category',
    boundaryGap: false,
    data: last7Days,
    axisLine: { lineStyle: { color: '#dcdfe6' } },
    axisLabel: { color: '#909399' },
  },
  yAxis: {
    type: 'value',
    splitLine: { lineStyle: { color: '#f0f2f5' } },
    axisLabel: { color: '#909399' },
  },
  series: [
    {
      name: 'UV',
      type: 'line',
      smooth: true,
      symbol: 'circle',
      symbolSize: 6,
      data: uvData,
      itemStyle: { color: '#409EFF' },
      areaStyle: {
        color: {
          type: 'linear',
          x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: 'rgba(64,158,255,0.35)' },
            { offset: 1, color: 'rgba(64,158,255,0.02)' },
          ],
        },
      },
    },
    {
      name: 'PV',
      type: 'line',
      smooth: true,
      symbol: 'circle',
      symbolSize: 6,
      data: pvData,
      itemStyle: { color: '#67C23A' },
      areaStyle: {
        color: {
          type: 'linear',
          x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: 'rgba(103,194,58,0.30)' },
            { offset: 1, color: 'rgba(103,194,58,0.02)' },
          ],
        },
      },
    },
  ],
}))

/* ---------------- 饼图: 角色分布 ---------------- */
const roleOption = computed(() => ({
  tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
  legend: { bottom: 4, left: 'center' },
  series: [
    {
      name: '角色分布',
      type: 'pie',
      radius: ['45%', '70%'],
      center: ['50%', '45%'],
      avoidLabelOverlap: true,
      itemStyle: {
        borderRadius: 6,
        borderColor: '#fff',
        borderWidth: 2,
      },
      label: { show: false },
      emphasis: {
        label: { show: true, fontSize: 14, fontWeight: 600 },
      },
      labelLine: { show: false },
      data: [
        { value: 24, name: '管理员', itemStyle: { color: '#409EFF' } },
        { value: 386, name: '普通用户', itemStyle: { color: '#67C23A' } },
        { value: 876, name: '访客', itemStyle: { color: '#E6A23C' } },
      ],
    },
  ],
}))

/* ---------------- 柱状图: 业务模块调用 ---------------- */
const moduleOption = computed(() => ({
  tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
  grid: { left: 36, right: 24, top: 30, bottom: 28 },
  xAxis: {
    type: 'category',
    data: ['用户', '角色', '菜单', 'API', '公告'],
    axisLine: { lineStyle: { color: '#dcdfe6' } },
    axisLabel: { color: '#909399' },
  },
  yAxis: {
    type: 'value',
    splitLine: { lineStyle: { color: '#f0f2f5' } },
    axisLabel: { color: '#909399' },
  },
  series: [
    {
      name: '调用次数',
      type: 'bar',
      barWidth: 28,
      data: [
        { value: 1820, itemStyle: { color: '#409EFF', borderRadius: [6, 6, 0, 0] } },
        { value: 642, itemStyle: { color: '#67C23A', borderRadius: [6, 6, 0, 0] } },
        { value: 318, itemStyle: { color: '#E6A23C', borderRadius: [6, 6, 0, 0] } },
        { value: 2410, itemStyle: { color: '#909399', borderRadius: [6, 6, 0, 0] } },
        { value: 580, itemStyle: { color: '#F56C6C', borderRadius: [6, 6, 0, 0] } },
      ],
    },
  ],
}))

/* ---------------- 最近公告 (mock) ---------------- */
interface NoticeItem {
  id: number
  title: string
  level: 1 | 2 | 3
  time: string
}
const recentNotices: NoticeItem[] = [
  { id: 1, title: '系统于本周日凌晨 02:00 进行例行维护', level: 3, time: '2026-05-25 09:12' },
  { id: 2, title: 'v2.4.0 版本已发布: 新增 API 资源管理', level: 2, time: '2026-05-23 17:30' },
  { id: 3, title: '请所有同事在 5 月底前完成账号信息核对', level: 2, time: '2026-05-22 10:05' },
  { id: 4, title: '欢迎 3 位新成员加入研发团队', level: 1, time: '2026-05-20 14:48' },
  { id: 5, title: '关于优化登录速度的若干说明', level: 1, time: '2026-05-18 11:20' },
]
const levelText = (l: number) => (l === 3 ? '紧急' : l === 2 ? '重要' : '普通')
const levelTag = (l: number) => (l === 3 ? 'danger' : l === 2 ? 'warning' : 'info')
</script>

<template>
  <div class="dashboard">
    <!-- 欢迎条 -->
    <div class="welcome">
      <div class="welcome-text">
        <div class="hello">
          你好, <span class="name">{{ userInfo?.nickname || userInfo?.username || 'Admin' }}</span> 👋
        </div>
        <div class="sub">欢迎使用 Hinay Admin 后台管理系统, 祝你工作愉快。</div>
      </div>
      <div class="welcome-meta">
        <el-button type="primary" plain :icon="User" size="small" @click="$router.push('/profile')">个人中心</el-button>
        <el-icon class="meta-icon"><Calendar /></el-icon>
        <span>{{ new Date().toLocaleDateString('zh-CN') }}</span>
      </div>
    </div>

    <!-- KPI 卡片 -->
    <el-row :gutter="16" class="row-mt">
      <el-col v-for="(k, i) in kpis" :key="i" :xs="24" :sm="12" :md="12" :lg="6" :xl="6">
        <el-card shadow="hover" class="kpi-card" body-style="padding:18px;">
          <div class="kpi-row">
            <div class="kpi-meta">
              <div class="kpi-label">{{ k.label }}</div>
              <div class="kpi-value">{{ k.value.toLocaleString() }}</div>
              <div class="kpi-delta" :class="k.delta >= 0 ? 'up' : 'down'">
                <el-icon><component :is="k.delta >= 0 ? ArrowUp : ArrowDown" /></el-icon>
                <span>{{ Math.abs(k.delta).toFixed(1) }}%</span>
                <span class="delta-tip">较上周</span>
              </div>
            </div>
            <div class="kpi-icon" :style="{ background: k.gradient }">
              <el-icon :size="22"><component :is="k.icon" /></el-icon>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 折线图 + 饼图 -->
    <el-row :gutter="16" class="row-mt">
      <el-col :xs="24" :sm="24" :md="24" :lg="16" :xl="16">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-header">
              <el-icon class="chart-header-icon"><TrendCharts /></el-icon>
              <span>近 7 天访问趋势</span>
            </div>
          </template>
          <ClientOnly>
            <v-chart :option="trendOption" autoresize style="height:320px" />
            <template #fallback>
              <div class="chart-loading">图表加载中...</div>
            </template>
          </ClientOnly>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="24" :md="24" :lg="8" :xl="8">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-header">
              <el-icon class="chart-header-icon"><PieChart /></el-icon>
              <span>角色分布</span>
            </div>
          </template>
          <ClientOnly>
            <v-chart :option="roleOption" autoresize style="height:320px" />
            <template #fallback>
              <div class="chart-loading">图表加载中...</div>
            </template>
          </ClientOnly>
        </el-card>
      </el-col>
    </el-row>

    <!-- 柱状图 + 最近公告 -->
    <el-row :gutter="16" class="row-mt">
      <el-col :xs="24" :sm="24" :md="24" :lg="12" :xl="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-header">
              <el-icon class="chart-header-icon"><DataAnalysis /></el-icon>
              <span>业务模块调用</span>
            </div>
          </template>
          <ClientOnly>
            <v-chart :option="moduleOption" autoresize style="height:300px" />
            <template #fallback>
              <div class="chart-loading">图表加载中...</div>
            </template>
          </ClientOnly>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="24" :md="24" :lg="12" :xl="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-header">
              <el-icon class="chart-header-icon"><Bell /></el-icon>
              <span>最近公告</span>
            </div>
          </template>
          <ul class="notice-list">
            <li v-for="n in recentNotices" :key="n.id" class="notice-item">
              <el-tag :type="levelTag(n.level)" size="small" class="notice-tag">
                {{ levelText(n.level) }}
              </el-tag>
              <span class="notice-title">{{ n.title }}</span>
              <span class="notice-time">{{ n.time }}</span>
            </li>
          </ul>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
}
.row-mt {
  margin-top: 16px;
}
/* 顶部欢迎条 */
.welcome {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18px 22px;
  border-radius: 8px;
  background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
  color: #fff;
  box-shadow: 0 4px 16px rgba(24, 144, 255, 0.18);
}
.welcome-text .hello {
  font-size: 18px;
  font-weight: 600;
  letter-spacing: 0.4px;
}
.welcome-text .name {
  text-decoration: underline;
  text-underline-offset: 4px;
}
.welcome-text .sub {
  margin-top: 6px;
  font-size: 13px;
  opacity: 0.85;
}
.welcome-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  opacity: 0.9;
}
.meta-icon {
  font-size: 16px;
}

/* KPI 卡片 */
.kpi-card {
  border-radius: 8px;
  margin-bottom: 16px;
}
.kpi-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.kpi-meta {
  flex: 1;
}
.kpi-label {
  font-size: 13px;
  color: #909399;
}
.kpi-value {
  font-size: 26px;
  font-weight: 600;
  color: #303133;
  margin: 6px 0 4px;
  line-height: 1.2;
}
.kpi-delta {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
}
.kpi-delta.up {
  color: #67c23a;
}
.kpi-delta.down {
  color: #f56c6c;
}
.kpi-delta .delta-tip {
  color: #909399;
  margin-left: 4px;
}
.kpi-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 6px 14px rgba(0, 0, 0, 0.08);
  flex-shrink: 0;
}

/* 图表卡片 */
.chart-card {
  border-radius: 8px;
  margin-bottom: 16px;
}
.chart-header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  color: #303133;
}
.chart-header-icon {
  color: #409eff;
  font-size: 16px;
}
.chart-loading {
  height: 320px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #909399;
}

/* 公告列表 */
.notice-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.notice-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 0;
  border-bottom: 1px dashed #ebeef5;
}
.notice-item:last-child {
  border-bottom: none;
}
.notice-tag {
  flex-shrink: 0;
}
.notice-title {
  flex: 1;
  font-size: 14px;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.notice-time {
  font-size: 12px;
  color: #909399;
  flex-shrink: 0;
}

@media (max-width: 768px) {
  .welcome {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
  .notice-title {
    white-space: normal;
  }
}
</style>
