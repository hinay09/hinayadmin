<script setup lang="ts">
/**
 * 我的审批中心: 待我审批 / 我已审批 / 我发起的 / 抄送我的 四个视角。
 * 待办支持批量同意与停留时长展示 (超 48h 标红), 抄送支持未读筛选与批量已读。
 */
import { onActivated, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Bell, Check, Search, View, Refresh, RefreshLeft } from '@element-plus/icons-vue'
import { useFlowApi, flowInstStatusMap, flowTaskStatusMap, type FlowInstanceItem } from '~/composables/useApi/flow'

definePageMeta({ title: '我的审批' })
defineOptions({ name: 'flow-center' })

const api = useFlowApi()

const scopes = [
  { key: 'todo', label: '待我审批' },
  { key: 'done', label: '我已审批' },
  { key: 'mine', label: '我发起的' },
  { key: 'ccme', label: '抄送我的' },
] as const

const activeTab = ref<'todo' | 'done' | 'mine' | 'ccme'>('todo')
const loading = ref(false)
const list = ref<FlowInstanceItem[]>([])
const total = ref(0)
const query = reactive({
  keyword: '',
  status: undefined as number | undefined,
  taskStatus: undefined as number | undefined, // ccme 视角: 1=未读, 2=已阅
  page: 1,
  pageSize: 10,
})
const counts = reactive({ todo: 0, cc: 0 })
const selected = ref<FlowInstanceItem[]>([])
const batchRunning = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await api.instList({
      scope: activeTab.value,
      keyword: query.keyword,
      status: activeTab.value === 'mine' ? query.status : undefined,
      taskStatus: activeTab.value === 'ccme' ? query.taskStatus : undefined,
      page: query.page,
      pageSize: query.pageSize,
    })
    list.value = res.list || []
    total.value = res.total || 0
  } finally { loading.value = false }
}

async function loadCounts() {
  try {
    const c = await api.taskCount()
    counts.todo = c.todo
    counts.cc = c.cc
  } catch {}
}

function switchTab() {
  query.page = 1
  query.status = undefined
  query.taskStatus = undefined
  selected.value = []
  load()
}

function resetQuery() {
  query.keyword = ''
  query.status = undefined
  query.taskStatus = undefined
  query.page = 1
  load()
}

function statusTag(s: number) { return flowInstStatusMap[s]?.tag || 'info' }
function statusText(s: number) { return flowInstStatusMap[s]?.text || '未知' }

function gotoDetail(id: number) {
  navigateTo({ path: '/flow/detail', query: { id: String(id) } })
}

/* -------- 停留时长 (待办) -------- */

// 服务端时间为 "YYYY-MM-DD HH:mm:ss", Safari 不认空格分隔, 统一替换为 ISO 的 "T"
function parseServerTime(s?: string | null): number | null {
  if (!s) return null
  const t = new Date(s.includes(' ') ? s.replace(' ', 'T') : s).getTime()
  return Number.isNaN(t) ? null : t
}

function waitDuration(s?: string | null): string {
  const t = parseServerTime(s)
  if (t == null) return '-'
  const mins = Math.max(0, Math.floor((Date.now() - t) / 60000))
  if (mins < 1) return '刚刚'
  if (mins < 60) return `${mins} 分钟`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return mins % 60 ? `${hours} 小时 ${mins % 60} 分` : `${hours} 小时`
  return hours % 24 ? `${Math.floor(hours / 24)} 天 ${hours % 24} 小时` : `${Math.floor(hours / 24)} 天`
}

// 超 48 小时未处理视为积压, 标红提示
function isOverdue(s?: string | null): boolean {
  const t = parseServerTime(s)
  return t != null && Date.now() - t > 48 * 3600 * 1000
}

/* -------- 批量操作 (复用单条接口, 静默逐条调用后汇总结果) -------- */

function onSelectionChange(rows: FlowInstanceItem[]) { selected.value = rows }

// 抄送视角仅未读行可勾选 (已阅无需再标)
function ccSelectable(row: FlowInstanceItem): boolean {
  return activeTab.value !== 'ccme' || row.taskStatus === 1
}

function ccRowClass({ row }: { row: FlowInstanceItem }): string {
  return activeTab.value === 'ccme' && row.taskStatus === 1 ? 'cc-unread-row' : ''
}

function esc(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

// 批量结果提示: 全部成功用轻提示, 部分失败弹框逐条列出
function reportBatchResult(ok: number, fails: string[], action: string) {
  if (fails.length === 0) {
    ElMessage.success(`已${action} ${ok} 条`)
    return
  }
  const html = `成功 ${ok} 条, 失败 ${fails.length} 条:<br>${fails.map(f => esc(f)).join('<br>')}`
  ElMessageBox.alert(html, `${action}结果`, { dangerouslyUseHTMLString: true, type: 'warning' })
}

async function handleBatchApprove() {
  const rows = selected.value.filter(r => r.taskId)
  if (!rows.length) return
  const r = await ElMessageBox.prompt(`将批量同意所选 ${rows.length} 条待办审批。`, '批量同意', {
    confirmButtonText: '确定同意',
    cancelButtonText: '取消',
    inputType: 'textarea',
    inputPlaceholder: '审批意见 (选填, 应用于全部所选)',
  }).catch(() => null)
  if (r === null) return
  const comment = (r.value || '').trim()
  batchRunning.value = true
  let ok = 0
  const fails: string[] = []
  try {
    for (const row of rows) {
      try {
        await api.taskApprove(row.taskId, comment, { silent: true })
        ok++
      } catch (e: any) {
        fails.push(`「${row.title}」: ${e?.message || '失败'}`)
      }
    }
  } finally { batchRunning.value = false }
  reportBatchResult(ok, fails, '同意')
  load()
  loadCounts()
}

async function handleBatchRead() {
  const rows = selected.value.filter(r => r.taskId && r.taskStatus === 1)
  if (!rows.length) {
    ElMessage.info('所选没有未读的抄送')
    return
  }
  batchRunning.value = true
  let ok = 0
  const fails: string[] = []
  try {
    for (const row of rows) {
      try {
        await api.taskRead(row.taskId, { silent: true })
        ok++
      } catch (e: any) {
        fails.push(`「${row.title}」: ${e?.message || '失败'}`)
      }
    }
  } finally { batchRunning.value = false }
  reportBatchResult(ok, fails, '标记已读')
  load()
  loadCounts()
}

/* -------- 单条操作 -------- */

async function handleCancel(row: FlowInstanceItem) {
  await ElMessageBox.confirm(`撤销「${row.title}」? 未处理的待办将全部作废。`, '撤销确认', { type: 'warning' })
  try {
    await api.instCancel(row.id)
    ElMessage.success('已撤销')
    load()
    loadCounts()
  } catch {}
}

async function handleRead(row: FlowInstanceItem) {
  if (!row.taskId || row.taskStatus !== 1) return
  try {
    await api.taskRead(row.taskId)
    ElMessage.success('已读')
    load()
    loadCounts()
  } catch {}
}

async function handleUrge(row: FlowInstanceItem) {
  const r = await ElMessageBox.prompt('将向当前待办审批人发送催办提醒 (10 分钟内限一次)。', '催办', {
    confirmButtonText: '发送催办',
    cancelButtonText: '取消',
    inputType: 'textarea',
    inputPlaceholder: '催办说明 (选填)',
  }).catch(() => null)
  if (r === null) return
  try {
    await api.instUrge(row.id, (r.value || '').trim())
    ElMessage.success('已发送催办提醒')
    load()
  } catch {}
}

onMounted(() => { load(); loadCounts() })
onActivated(() => { load(); loadCounts() })
</script>

<template>
  <div class="page">
    <el-card>
      <el-tabs v-model="activeTab" @tab-change="switchTab">
        <el-tab-pane name="todo">
          <template #label>
            待我审批<el-badge v-if="counts.todo" :value="counts.todo" style="margin-left:6px" />
          </template>
        </el-tab-pane>
        <el-tab-pane label="我已审批" name="done" />
        <el-tab-pane label="我发起的" name="mine" />
        <el-tab-pane name="ccme">
          <template #label>
            抄送我的<el-badge v-if="counts.cc" :value="counts.cc" style="margin-left:6px" />
          </template>
        </el-tab-pane>
      </el-tabs>

      <el-form inline @submit.prevent>
        <el-form-item label="搜索">
          <el-input v-model="query.keyword" placeholder="标题/流程名" clearable
            @keyup.enter="() => { query.page = 1; load() }" />
        </el-form-item>
        <el-form-item v-if="activeTab === 'mine'" label="状态">
          <el-select v-model="query.status" clearable placeholder="全部" style="width:120px">
            <el-option v-for="(v, k) in flowInstStatusMap" :key="k" :label="v.text" :value="Number(k)" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="activeTab === 'ccme'" label="阅读">
          <el-select v-model="query.taskStatus" clearable placeholder="全部" style="width:110px"
            @change="() => { query.page = 1; load() }">
            <el-option label="未读" :value="1" />
            <el-option label="已阅" :value="2" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="() => { query.page = 1; load() }">查询</el-button>
          <el-button :icon="Refresh" @click="resetQuery">重置</el-button>
        </el-form-item>
      </el-form>

      <div v-if="activeTab === 'todo' || activeTab === 'ccme'" class="batch-bar">
        <el-button v-if="activeTab === 'todo'" type="success" :icon="Check"
          :disabled="!selected.length" :loading="batchRunning" @click="handleBatchApprove">
          批量同意{{ selected.length ? ` (${selected.length})` : '' }}
        </el-button>
        <el-button v-if="activeTab === 'ccme'" type="primary"
          :disabled="!selected.length" :loading="batchRunning" @click="handleBatchRead">
          批量已读{{ selected.length ? ` (${selected.length})` : '' }}
        </el-button>
      </div>

      <el-table v-loading="loading" :data="list" border stripe
        :row-class-name="ccRowClass" @selection-change="onSelectionChange">
        <el-table-column v-if="activeTab === 'todo' || activeTab === 'ccme'"
          type="selection" width="42" :selectable="ccSelectable" />
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="title" label="申请标题" min-width="170" show-overflow-tooltip />
        <el-table-column prop="flowName" label="流程" width="130" show-overflow-tooltip />
        <el-table-column prop="startUserName" label="发起人" width="100" />
        <el-table-column v-if="activeTab === 'todo'" prop="currentNodes" label="当前节点" width="120" show-overflow-tooltip />
        <el-table-column v-if="activeTab === 'todo'" label="停留时长" width="110">
          <template #default="{ row }">
            <span :class="{ 'wait-overdue': isOverdue(row.taskReceiveTime) }">{{ waitDuration(row.taskReceiveTime) }}</span>
          </template>
        </el-table-column>
        <el-table-column v-if="activeTab === 'done'" label="我的处理" width="100">
          <template #default="{ row }">
            <el-tag :type="flowTaskStatusMap[row.taskStatus]?.tag || 'info'">
              {{ flowTaskStatusMap[row.taskStatus]?.text || '-' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column v-if="activeTab === 'ccme'" label="阅读状态" width="90">
          <template #default="{ row }">
            <el-tag v-if="row.taskStatus === 1" type="danger">未读</el-tag>
            <el-tag v-else type="info">已阅</el-tag>
          </template>
        </el-table-column>
        <el-table-column v-if="activeTab === 'mine' || activeTab === 'done'" label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status)">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="发起时间" width="170" />
        <el-table-column v-if="activeTab !== 'todo'" prop="finishedAt" label="结束时间" width="170" />
        <el-table-column label="操作" width="230" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" :icon="View" @click="gotoDetail(row.id)">
              {{ activeTab === 'todo' ? '去审批' : activeTab === 'mine' && (row.status === 6 || row.status === 4) ? '重新提交' : '详情' }}
            </el-button>
            <el-button v-if="activeTab === 'mine' && (row.status === 1 || row.status === 6)" v-permission="'flow:instance:cancel'"
              link type="warning" :icon="RefreshLeft" @click="handleCancel(row)">撤销</el-button>
            <el-button v-if="activeTab === 'mine' && row.status === 1"
              link type="primary" :icon="Bell" @click="handleUrge(row)">催办</el-button>
            <el-button v-if="activeTab === 'ccme' && row.taskStatus === 1" link type="success" @click="handleRead(row)">标记已读</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination v-model:current-page="query.page" v-model:page-size="query.pageSize"
        :total="total" :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next, jumper"
        style="margin-top:16px;justify-content:flex-end"
        @current-change="load" @size-change="load" />
    </el-card>
  </div>
</template>

<style scoped>
.batch-bar {
  display: flex;
  justify-content: flex-end;
  margin: 2px 0 12px;
}

.wait-overdue {
  color: var(--el-color-danger);
  font-weight: 600;
}

/* 抄送未读行浅色高亮 (el-table 斑马纹在单元格上, 需 !important 覆盖) */
:deep(.cc-unread-row .el-table__cell) {
  background: var(--el-color-primary-light-9) !important;
}
</style>
