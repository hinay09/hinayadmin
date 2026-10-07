<script setup lang="ts">
/**
 * 我的审批中心: 待我审批 / 我已审批 / 我发起的 / 抄送我的 四个视角。
 */
import { onActivated, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Bell, Search, View, RefreshLeft } from '@element-plus/icons-vue'
import { useFlowApi, flowInstStatusMap, type FlowInstanceItem } from '~/composables/useApi/flow'

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
const query = reactive({ keyword: '', status: undefined as number | undefined, page: 1, pageSize: 10 })
const counts = reactive({ todo: 0, cc: 0 })

async function load() {
  loading.value = true
  try {
    const res = await api.instList({
      scope: activeTab.value,
      keyword: query.keyword,
      status: activeTab.value === 'mine' ? query.status : undefined,
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
  load()
}

function statusTag(s: number) { return flowInstStatusMap[s]?.tag || 'info' }
function statusText(s: number) { return flowInstStatusMap[s]?.text || '未知' }

function gotoDetail(id: number) {
  navigateTo({ path: '/flow/detail', query: { id: String(id) } })
}

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
  if (!row.taskId) return
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
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="() => { query.page = 1; load() }">查询</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="list" border stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="title" label="申请标题" min-width="170" show-overflow-tooltip />
        <el-table-column prop="flowName" label="流程" width="130" show-overflow-tooltip />
        <el-table-column prop="startUserName" label="发起人" width="100" />
        <el-table-column v-if="activeTab === 'todo'" prop="currentNodes" label="当前节点" width="120" show-overflow-tooltip />
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
            <el-button v-if="activeTab === 'ccme'" link type="success" @click="handleRead(row)">标记已读</el-button>
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
