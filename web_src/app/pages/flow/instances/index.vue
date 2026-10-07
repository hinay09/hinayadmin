<script setup lang="ts">
/**
 * 流程实例管理 (管理员): 跨用户列出全部审批记录, 运行中的实例可动态加签/减签或终止。
 * 数据走 scope=all 全局视角 (后端仅放行管理员); 加签/减签锚定当前待办节点任一任务,
 * 中止复用实例终止 (待办作废、原同意置已失效)。
 */
import { computed, onActivated, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Minus, Plus, Search, VideoPause, View } from '@element-plus/icons-vue'
import { useUserStore } from '~/stores/user'
import {
  useFlowApi, flowInstStatusMap,
  type FlowInstanceItem, type FlowInstanceDetail, type FlowTaskItem, type FlowUserOption,
} from '~/composables/useApi/flow'

definePageMeta({ title: '流程实例管理' })
defineOptions({ name: 'flow-instances' })

const api = useFlowApi()
const userStore = useUserStore()
const isAdmin = computed(() => userStore.isAdmin)

const loading = ref(false)
const list = ref<FlowInstanceItem[]>([])
const total = ref(0)
const query = reactive({ keyword: '', status: undefined as number | undefined, flowKey: '', page: 1, pageSize: 10 })

const users = ref<FlowUserOption[]>([])

async function load() {
  loading.value = true
  try {
    const res = await api.instList({
      scope: 'all',
      keyword: query.keyword,
      status: query.status,
      flowKey: query.flowKey || undefined,
      page: query.page,
      pageSize: query.pageSize,
    })
    list.value = res.list || []
    total.value = res.total || 0
  } finally { loading.value = false }
}

function statusTag(s: number) { return flowInstStatusMap[s]?.tag || 'info' }
function statusText(s: number) { return flowInstStatusMap[s]?.text || '未知' }
/** 运行中的实例才有节点级调整与终止入口 */
const isRunning = (row: FlowInstanceItem) => row.status === 1

function gotoDetail(id: number) {
  navigateTo({ path: '/flow/detail', query: { id: String(id) } })
}

// ============================================================
// 加签/减签: 拉实例详情取当前待办节点 (条件分支可能多个), 按节点锚定任一待办
// ============================================================
const signVisible = ref(false)
const signMode = ref<'append' | 'reduce'>('append')
const signLoading = ref(false)
const signDetail = ref<FlowInstanceDetail | null>(null)
const signNodeId = ref('')
const signUserIds = ref<number[]>([])
const signComment = ref('')

/** 当前待办审批任务按节点聚合 (保留节点内首次到达顺序) */
const pendingNodes = computed(() => {
  const nodes: { nodeId: string, nodeName: string, signType: number, tasks: FlowTaskItem[] }[] = []
  const byId = new Map<string, (typeof nodes)[number]>()
  for (const t of signDetail.value?.tasks || []) {
    if (t.nodeType !== 1 || t.status !== 1) continue
    let n = byId.get(t.nodeId)
    if (!n) {
      n = { nodeId: t.nodeId, nodeName: t.nodeName, signType: t.signType, tasks: [] }
      byId.set(t.nodeId, n)
      nodes.push(n)
    }
    n.tasks.push(t)
  }
  return nodes
})
const currentNode = computed(() => pendingNodes.value.find(n => n.nodeId === signNodeId.value) || null)
/** 所选节点待办审批人 ID 集 (加签时置灰) */
const nodeAssigneeIds = computed(() => new Set(currentNode.value?.tasks.map(t => t.assigneeId) || []))
/** 锚点任务: 所选节点任一待办 (后端对管理员放行非本人任务) */
const anchorTaskId = computed(() => currentNode.value?.tasks[0]?.id || 0)

function resetSign() {
  signNodeId.value = ''
  signUserIds.value = []
  signComment.value = ''
}

async function openSign(row: FlowInstanceItem, mode: 'append' | 'reduce') {
  resetSign()
  signMode.value = mode
  signLoading.value = true
  signVisible.value = true
  try {
    signDetail.value = await api.instDetail(row.id)
    if (!pendingNodes.value.length) {
      signVisible.value = false
      ElMessage.warning('该流程当前没有待办审批节点')
      return
    }
    signNodeId.value = pendingNodes.value[0].nodeId
  } catch {
    signVisible.value = false
  } finally { signLoading.value = false }
}

async function confirmSign() {
  if (!anchorTaskId.value) { signVisible.value = false; return }
  if (!signUserIds.value.length) {
    ElMessage.warning(signMode.value === 'append' ? '请选择加签人' : '请选择要移除的审批人')
    return
  }
  if (signMode.value === 'reduce' && signUserIds.value.length >= (currentNode.value?.tasks.length || 0)) {
    ElMessage.warning('不能移除全部待办审批人, 至少保留一人')
    return
  }
  signLoading.value = true
  try {
    if (signMode.value === 'append') {
      await api.taskAppend(anchorTaskId.value, signUserIds.value, signComment.value.trim())
      ElMessage.success('已加签, 该节点转为会签')
    } else {
      await api.taskReduce(anchorTaskId.value, signUserIds.value, signComment.value.trim())
      ElMessage.success('已减签')
    }
    signVisible.value = false
    load()
  } catch {} finally { signLoading.value = false }
}

// ============================================================
// 中止: 运行中实例立即结束 (待办作废, 不可恢复)
// ============================================================
async function terminate(row: FlowInstanceItem) {
  const r = await ElMessageBox.prompt(
    `终止「${row.title}」? 待办将全部作废, 流程立即结束且不可恢复。`, '中止流程', {
      confirmButtonText: '确认中止',
      cancelButtonText: '取消',
      inputType: 'textarea',
      inputPlaceholder: '中止原因 (选填)',
      type: 'warning',
    }).catch(() => null)
  if (r === null) return
  try {
    await api.instTerminate(row.id, (r.value || '').trim())
    ElMessage.success('已中止流程')
    load()
  } catch {}
}

onMounted(async () => {
  load()
  try {
    const opt = await api.options()
    users.value = opt.users || []
  } catch {}
})
onActivated(() => load())
</script>

<template>
  <div class="page">
    <el-card>
      <el-alert type="info" :closable="false" show-icon style="margin-bottom:14px"
        title="管理员视角: 列出全部用户的审批记录。运行中的流程可对当前节点加签/减签 (节点转为会签) 或中止 (待办作废, 立即结束)。" />

      <el-form inline @submit.prevent>
        <el-form-item label="搜索">
          <el-input v-model="query.keyword" placeholder="标题/流程名" clearable
            @keyup.enter="() => { query.page = 1; load() }" />
        </el-form-item>
        <el-form-item label="流程标识">
          <el-input v-model="query.flowKey" placeholder="如 biz_leave" clearable style="width:150px"
            @keyup.enter="() => { query.page = 1; load() }" />
        </el-form-item>
        <el-form-item label="状态">
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
        <el-table-column label="当前节点" width="130" show-overflow-tooltip>
          <template #default="{ row }">{{ row.currentNodes || '-' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status)">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="发起时间" width="170" />
        <el-table-column prop="finishedAt" label="结束时间" width="170" />
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" :icon="View" @click="gotoDetail(row.id)">详情</el-button>
            <template v-if="isAdmin && isRunning(row)">
              <el-button v-permission="'flow:task:handle'" link type="primary" :icon="Plus" @click="openSign(row, 'append')">加签</el-button>
              <el-button v-permission="'flow:task:handle'" link type="warning" :icon="Minus" @click="openSign(row, 'reduce')">减签</el-button>
              <el-button link type="danger" :icon="VideoPause" @click="terminate(row)">中止</el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination v-model:current-page="query.page" v-model:page-size="query.pageSize"
        :total="total" :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next, jumper"
        style="margin-top:16px;justify-content:flex-end"
        @current-change="load" @size-change="load" />
    </el-card>

    <!-- 加签/减签弹窗: 选当前待办节点 + 选人 (锚定该节点任一待办任务) -->
    <el-dialog v-model="signVisible" :title="signMode === 'append' ? '加签' : '减签'" width="520px" append-to-body destroy-on-close>
      <div v-loading="signLoading">
        <el-alert type="info" :closable="false" show-icon style="margin-bottom:14px"
          :title="signMode === 'append'
            ? '加签后该节点转为会签: 原审批人与新加人均须同意, 节点才通过; 任一人驳回仍整单退回。'
            : '移除后其待办作废; 已同意的不回退。该节点至少保留一位待办审批人。'" />
        <el-form label-width="90px">
          <el-form-item label="目标节点" required>
            <el-select v-model="signNodeId" style="width:100%">
              <el-option v-for="n in pendingNodes" :key="n.nodeId" :value="n.nodeId"
                :label="`${n.nodeName}（${n.tasks.map(t => t.assigneeName).join('、')}）`" />
            </el-select>
          </el-form-item>
          <el-form-item :label="signMode === 'append' ? '加签给' : '移除谁'" required>
            <el-select v-if="signMode === 'append'" v-model="signUserIds" multiple filterable placeholder="选择要加入的审批人" style="width:100%">
              <el-option v-for="u in users" :key="u.id" :label="u.nickname || u.username" :value="u.id"
                :disabled="nodeAssigneeIds.has(u.id)" />
            </el-select>
            <el-select v-else v-model="signUserIds" multiple filterable placeholder="选择要移除的审批人" style="width:100%">
              <el-option v-for="t in currentNode?.tasks || []" :key="t.assigneeId" :label="t.assigneeName" :value="t.assigneeId" />
            </el-select>
          </el-form-item>
          <el-form-item label="说明">
            <el-input v-model="signComment" type="textarea" :rows="3" maxlength="500" show-word-limit
              placeholder="选填, 将记入流转记录" />
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="signVisible = false">取消</el-button>
        <el-button :type="signMode === 'append' ? 'primary' : 'danger'" :loading="signLoading" @click="confirmSign">
          {{ signMode === 'append' ? '确认加签' : '确认减签' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>
