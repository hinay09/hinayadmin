<script setup lang="ts">
/**
 * 服务器监控 (仅超管): CPU/内存/磁盘/主机信息/Go 运行时 + MySQL/Redis 连接状态。
 * - CPU 占用为自上次轮询以来的增量值, 刷新间隔即采样窗口 (默认 5s)
 * - 三组接口独立拉取、独立错误态: Redis 挂了不影响服务器指标展示
 * - 页面隐藏时暂停轮询 (visibilitychange), 回到页面立即采样一次
 */
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { useMonitorApi, type ServerMonitor, type MysqlMonitor, type RedisMonitor } from '~/composables/useApi'

definePageMeta({ title: '服务器监控' })
defineOptions({ name: 'system-monitor' })

const api = useMonitorApi()

/* ---- 数据与加载态 ---- */
const REFRESH_MS = 5000
const server = ref<ServerMonitor | null>(null)
const mysql = ref<MysqlMonitor | null>(null)
const redis = ref<RedisMonitor | null>(null)
const serverErr = ref('')
const mysqlErr = ref('')
const redisErr = ref('')
const autoRefresh = ref(true)
let timer: ReturnType<typeof setInterval> | null = null

async function loadServer() {
  try {
    server.value = await api.server()
    serverErr.value = ''
  }
  catch (e: any) {
    serverErr.value = e?.message || '加载失败'
  }
}
async function loadMysql() {
  try {
    mysql.value = await api.mysql()
    mysqlErr.value = ''
  }
  catch (e: any) {
    mysqlErr.value = e?.message || '加载失败'
  }
}
async function loadRedis() {
  try {
    redis.value = await api.redis()
    redisErr.value = ''
  }
  catch (e: any) {
    redisErr.value = e?.message || '加载失败'
  }
}
function loadAll() {
  loadServer()
  loadMysql()
  loadRedis()
}

function startTimer() {
  stopTimer()
  timer = setInterval(() => {
    // CPU 增量采样只依赖 server 指标轮询; db 状态变化慢, 同频拉取也无妨
    loadServer()
    loadMysql()
    loadRedis()
  }, REFRESH_MS)
}
function stopTimer() {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}
function onVisibility() {
  if (document.hidden) {
    stopTimer()
  }
  else if (autoRefresh.value) {
    loadAll()
    startTimer()
  }
}

onMounted(() => {
  loadAll()
  startTimer()
  document.addEventListener('visibilitychange', onVisibility)
})
onBeforeUnmount(() => {
  stopTimer()
  document.removeEventListener('visibilitychange', onVisibility)
})
function toggleAuto(v: boolean | string | number) {
  if (v) {
    loadAll()
    startTimer()
  }
  else stopTimer()
}

/* ---- 展示辅助 ---- */
function fmtBytes(n?: number): string {
  if (n == null || Number.isNaN(n)) return '-'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let v = n
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v >= 100 ? v.toFixed(0) : v.toFixed(1)} ${units[i]}`
}
function fmtUptime(sec?: number): string {
  if (!sec || sec <= 0) return '-'
  const d = Math.floor(sec / 86400)
  const h = Math.floor((sec % 86400) / 3600)
  const m = Math.floor((sec % 3600) / 60)
  if (d > 0) return `${d} 天 ${h} 小时`
  if (h > 0) return `${h} 小时 ${m} 分`
  return `${m} 分 ${sec % 60} 秒`
}
function fmtNum(n?: number): string {
  if (n == null) return '-'
  return n.toLocaleString()
}
/** 用量颜色分档 */
function pctColor(pct?: number): string {
  if (pct == null) return '#409eff'
  if (pct >= 90) return '#f56c6c'
  if (pct >= 75) return '#e6a23c'
  return '#409eff'
}
const cpuPerCoreMax = computed(() => Math.min(16, server.value?.cpu.perCore?.length || 0))

/* 单卡片错误占位 */
function cardError(msg: string) {
  return msg || ''
}
</script>

<template>
  <div class="page monitor-page">
    <!-- 工具栏 -->
    <div class="toolbar">
      <span class="toolbar-title">服务器监控</span>
      <span v-if="server" class="toolbar-sub">{{ server.host.hostname }} · {{ server.host.platform }}</span>
      <div class="toolbar-actions">
        <el-switch v-model="autoRefresh" active-text="自动刷新" @change="toggleAuto" />
        <el-button type="primary" :icon="Refresh" @click="loadAll">刷新</el-button>
      </div>
    </div>

    <!-- 第一行: CPU / 内存 / Go 运行时 -->
    <el-row :gutter="16" class="card-row">
      <el-col :xs="24" :md="8">
        <el-card shadow="never" class="mono-card">
          <template #header><span class="card-title">CPU</span></template>
          <div v-if="serverErr" class="card-err">{{ cardError(serverErr) }}</div>
          <template v-else-if="server">
            <div class="gauge-row">
              <el-progress
                type="dashboard"
                :percentage="Math.min(100, server.cpu.usagePercent)"
                :color="pctColor(server.cpu.usagePercent)"
                :stroke-width="10"
              >
                <template #default>
                  <div class="gauge-num">{{ server.cpu.usagePercent.toFixed(1) }}%</div>
                  <div class="gauge-label">总体占用</div>
                </template>
              </el-progress>
              <div class="gauge-side">
                <div class="kv"><span>核心</span><b>{{ server.cpu.coresPhysical }} 物理核 / {{ server.cpu.coresLogical }} 逻辑核</b></div>
                <div class="kv"><span>型号</span><b class="ellipsis" :title="server.cpu.modelName">{{ server.cpu.modelName || '-' }}</b></div>
                <div class="kv"><span>负载</span><b>{{ server.cpu.load1.toFixed(2) }} / {{ server.cpu.load5.toFixed(2) }} / {{ server.cpu.load15.toFixed(2) }}</b></div>
              </div>
            </div>
            <!-- 每核占用: 最多展示 16 核, 更多折叠为提示 -->
            <div v-if="server.cpu.perCore?.length" class="core-grid">
              <div v-for="i in cpuPerCoreMax" :key="i" class="core-item">
                <span class="core-name">{{ i - 1 }}</span>
                <el-progress
                  :percentage="Math.min(100, server.cpu.perCore[i - 1])"
                  :color="pctColor(server.cpu.perCore[i - 1])"
                  :stroke-width="8"
                  :show-text="false"
                />
                <span class="core-val">{{ server.cpu.perCore[i - 1].toFixed(0) }}%</span>
              </div>
            </div>
            <div v-if="server.cpu.perCore?.length > 16" class="core-more">… 共 {{ server.cpu.perCore.length }} 核</div>
          </template>
          <div v-else v-loading="true" class="card-loading" />
        </el-card>
      </el-col>

      <el-col :xs="24" :md="8">
        <el-card shadow="never" class="mono-card">
          <template #header><span class="card-title">内存</span></template>
          <div v-if="serverErr" class="card-err">{{ cardError(serverErr) }}</div>
          <template v-else-if="server">
            <div class="gauge-row">
              <el-progress
                type="dashboard"
                :percentage="Math.min(100, server.memory.usedPercent)"
                :color="pctColor(server.memory.usedPercent)"
                :stroke-width="10"
              >
                <template #default>
                  <div class="gauge-num">{{ server.memory.usedPercent.toFixed(1) }}%</div>
                  <div class="gauge-label">已用</div>
                </template>
              </el-progress>
              <div class="gauge-side">
                <div class="kv"><span>总量</span><b>{{ fmtBytes(server.memory.total) }}</b></div>
                <div class="kv"><span>已用</span><b>{{ fmtBytes(server.memory.used) }}</b></div>
                <div class="kv"><span>可用</span><b>{{ fmtBytes(server.memory.available) }}</b></div>
              </div>
            </div>
          </template>
          <div v-else v-loading="true" class="card-loading" />
        </el-card>
      </el-col>

      <el-col :xs="24" :md="8">
        <el-card shadow="never" class="mono-card">
          <template #header><span class="card-title">Go 运行时</span></template>
          <div v-if="serverErr" class="card-err">{{ cardError(serverErr) }}</div>
          <template v-else-if="server">
            <div class="kv-list">
              <div class="kv"><span>版本</span><b>{{ server.go.goVersion }}</b></div>
              <div class="kv"><span>goroutine</span><b :class="{ 'hot': server.go.goroutines > 1000 }">{{ fmtNum(server.go.goroutines) }}</b></div>
              <div class="kv"><span>GOMAXPROCS</span><b>{{ server.go.goMaxProcs }}</b></div>
              <div class="kv"><span>堆已分配</span><b>{{ fmtBytes(server.go.heapAlloc) }}</b></div>
              <div class="kv"><span>堆申请(OS)</span><b>{{ fmtBytes(server.go.heapSys) }}</b></div>
              <div class="kv"><span>运行时申请总量</span><b>{{ fmtBytes(server.go.sys) }}</b></div>
              <div class="kv"><span>GC 次数</span><b>{{ fmtNum(server.go.numGC) }}</b></div>
              <div class="kv"><span>GC 累计停顿</span><b>{{ server.go.gcPauseTotalMs.toFixed(1) }} ms</b></div>
              <div class="kv"><span>最近 GC 停顿</span><b>{{ server.go.lastGCPauseMs.toFixed(2) }} ms</b></div>
            </div>
          </template>
          <div v-else v-loading="true" class="card-loading" />
        </el-card>
      </el-col>
    </el-row>

    <!-- 第二行: 主机信息 / 磁盘 -->
    <el-row :gutter="16" class="card-row">
      <el-col :xs="24" :md="10">
        <el-card shadow="never" class="mono-card">
          <template #header><span class="card-title">服务器信息</span></template>
          <div v-if="serverErr" class="card-err">{{ cardError(serverErr) }}</div>
          <template v-else-if="server">
            <div class="kv-list">
              <div class="kv"><span>主机名</span><b>{{ server.host.hostname || '-' }}</b></div>
              <div class="kv"><span>操作系统</span><b>{{ server.host.platform || server.host.os }}</b></div>
              <div class="kv"><span>内核</span><b class="ellipsis">{{ server.host.kernelVersion || '-' }} ({{ server.host.kernelArch }})</b></div>
              <div class="kv"><span>开机时间</span><b>{{ server.host.bootTime ? new Date(server.host.bootTime * 1000).toLocaleString() : '-' }}</b></div>
              <div class="kv"><span>已运行</span><b>{{ fmtUptime(server.host.uptimeSec) }}</b></div>
              <div class="kv"><span>应用运行</span><b>{{ fmtUptime(server.host.appUptimeSec) }}</b></div>
              <div class="kv"><span>服务器时间</span><b>{{ server.host.serverTime ? new Date(server.host.serverTime * 1000).toLocaleString() : '-' }}</b></div>
            </div>
          </template>
          <div v-else v-loading="true" class="card-loading" />
        </el-card>
      </el-col>

      <el-col :xs="24" :md="14">
        <el-card shadow="never" class="mono-card">
          <template #header><span class="card-title">磁盘</span></template>
          <div v-if="serverErr" class="card-err">{{ cardError(serverErr) }}</div>
          <div v-else-if="server" class="disk-list">
            <div v-for="d in server.disks" :key="d.mount" class="disk-item">
              <div class="disk-head">
                <el-tooltip :content="`${d.device} (${d.fstype})`" placement="top">
                  <span class="disk-mount ellipsis">{{ d.mount }}</span>
                </el-tooltip>
                <span class="disk-size">{{ fmtBytes(d.used) }} / {{ fmtBytes(d.total) }}</span>
              </div>
              <el-progress
                :percentage="Math.min(100, d.usedPercent)"
                :color="pctColor(d.usedPercent)"
                :stroke-width="10"
              />
            </div>
            <div v-if="!server.disks?.length" class="card-err">未采集到磁盘分区</div>
          </div>
          <div v-else v-loading="true" class="card-loading" />
        </el-card>
      </el-col>
    </el-row>

    <!-- 第三行: MySQL / Redis -->
    <el-row :gutter="16" class="card-row">
      <el-col :xs="24" :md="12">
        <el-card shadow="never" class="mono-card">
          <template #header>
            <div class="card-head-row">
              <span class="card-title">MySQL</span>
              <el-tag v-if="mysql" size="small" type="success" effect="plain">{{ mysql.version }}</el-tag>
            </div>
          </template>
          <div v-if="mysqlErr" class="card-err">{{ cardError(mysqlErr) }}</div>
          <template v-else-if="mysql">
            <div class="kv-list">
              <div class="kv"><span>已运行</span><b>{{ fmtUptime(mysql.uptimeSec) }}</b></div>
              <div class="kv"><span>当前连接</span><b>{{ mysql.threadsConnected }} (活跃 {{ mysql.threadsRunning }})</b></div>
              <div class="kv"><span>历史峰值连接</span><b>{{ mysql.maxUsedConnections }}</b></div>
              <div class="kv"><span>失败连接</span><b :class="{ 'hot': mysql.abortedConnects > 0 }">{{ fmtNum(mysql.abortedConnects) }}</b></div>
              <div class="kv"><span>慢查询</span><b :class="{ 'hot': mysql.slowQueries > 0 }">{{ fmtNum(mysql.slowQueries) }}</b></div>
              <div class="kv"><span>累计执行语句</span><b>{{ fmtNum(mysql.queries) }}</b></div>
              <div class="kv"><span>累计流量</span><b>入 {{ fmtBytes(mysql.bytesReceived) }} / 出 {{ fmtBytes(mysql.bytesSent) }}</b></div>
              <div class="kv">
                <span>应用连接池</span>
                <b :title="`${mysql.pool.host} / ${mysql.pool.name}`">
                  {{ mysql.pool.inUse }} 使用 / {{ mysql.pool.idle }} 空闲 / {{ mysql.pool.open }} 总计
                </b>
              </div>
              <div class="kv"><span>池等待</span><b>{{ fmtNum(mysql.pool.waitCount) }} 次 / {{ mysql.pool.waitDurationMs }} ms</b></div>
            </div>
          </template>
          <div v-else v-loading="true" class="card-loading" />
        </el-card>
      </el-col>

      <el-col :xs="24" :md="12">
        <el-card shadow="never" class="mono-card">
          <template #header>
            <div class="card-head-row">
              <span class="card-title">Redis</span>
              <el-tag v-if="redis" size="small" type="success" effect="plain">{{ redis.version }}</el-tag>
            </div>
          </template>
          <div v-if="redisErr" class="card-err">{{ cardError(redisErr) }}</div>
          <template v-else-if="redis">
            <div class="kv-list">
              <div class="kv"><span>已运行</span><b>{{ fmtUptime(redis.uptimeSec) }}</b></div>
              <div class="kv"><span>客户端连接</span><b>{{ redis.connectedClients }}</b></div>
              <div class="kv"><span>数据占用内存</span><b>{{ redis.usedMemoryHuman }} (峰值 {{ redis.usedMemoryPeakHuman }})</b></div>
              <div class="kv"><span>键数量</span><b>{{ fmtNum(redis.dbSize) }}</b></div>
              <div class="kv"><span>每秒操作</span><b>{{ redis.opsPerSec }}</b></div>
              <div class="kv">
                <span>缓存命中率</span>
                <b :class="{ 'hot': redis.hitRatePercent >= 0 && redis.hitRatePercent < 80 }">
                  {{ redis.hitRatePercent >= 0 ? redis.hitRatePercent.toFixed(1) + '%' : '暂无访问' }}
                </b>
              </div>
              <div class="kv"><span>命中 / 未命中</span><b>{{ fmtNum(redis.keyspaceHits) }} / {{ fmtNum(redis.keyspaceMisses) }}</b></div>
              <div class="kv"><span>累计连接数</span><b>{{ fmtNum(redis.totalConnections) }}</b></div>
            </div>
          </template>
          <div v-else v-loading="true" class="card-loading" />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.monitor-page {
  display: flex;
  flex-direction: column;
  gap: 0;
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
}
.toolbar-title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}
.toolbar-sub {
  font-size: 13px;
  color: #909399;
}
.toolbar-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 12px;
}

.card-row {
  margin-bottom: 16px;
}
.mono-card {
  height: 100%;
}
.card-title {
  font-weight: 600;
  color: #303133;
}
.card-head-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.card-err {
  padding: 24px 0;
  text-align: center;
  color: #f56c6c;
  font-size: 13px;
}
.card-loading {
  min-height: 160px;
}

/* 仪表盘行: 环形图 + 右侧键值 */
.gauge-row {
  display: flex;
  align-items: center;
  gap: 20px;
}
.gauge-row :deep(.el-progress--dashboard) {
  width: 130px;
}
.gauge-num {
  font-size: 18px;
  font-weight: 700;
  color: #303133;
}
.gauge-label {
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
}
.gauge-side {
  flex: 1;
  min-width: 0;
}

/* 键值列表 */
.kv-list .kv {
  padding: 5px 0;
}
.kv {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 13px;
}
.kv > span {
  color: #909399;
  flex-shrink: 0;
}
.kv > b {
  font-weight: 500;
  color: #303133;
  text-align: right;
  min-width: 0;
}
.kv > b.hot {
  color: #f56c6c;
}
.ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 每核占用网格: 双列, 核多时截断展示 */
.core-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 4px 16px;
  margin-top: 14px;
}
.core-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #606266;
}
.core-item :deep(.el-progress) {
  flex: 1;
}
.core-name {
  width: 34px;
  text-align: right;
  color: #909399;
}
.core-val {
  width: 34px;
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.core-more {
  margin-top: 6px;
  font-size: 12px;
  color: #909399;
  text-align: center;
}

/* 磁盘列表 */
.disk-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.disk-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
  font-size: 13px;
}
.disk-mount {
  color: #303133;
  font-weight: 500;
  max-width: 60%;
}
.disk-size {
  color: #909399;
  font-size: 12px;
}

/* 窄屏: 仪表盘行允许换行 */
@media (max-width: 768px) {
  .gauge-row {
    flex-wrap: wrap;
    justify-content: center;
  }
}
</style>
