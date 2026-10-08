<script setup lang="ts">
/**
 * 流程实例详情: 状态头图 + 流程进度 (按实际路径) + 审批操作/重新提交 + 申请表单 + 流转时间线 + 任务明细。
 * 驳回语义: 退回发起人 (实例不终止), 发起人在本页修改表单后可重新提交 (流程从头重走) 或撤销;
 * 撤销后发起人仍可修改表单重新提交 (复用实例快照, 历史轮次保留)。
 */
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowLeft, Bell, Check, CircleCheck, CircleClose, Clock, Close, Connection, EditPen,
  Minus, Plus, Position, RefreshLeft, Share, Stamp, VideoPause, Warning,
} from '@element-plus/icons-vue'
import { useUserStore } from '~/stores/user'
import {
  useFlowApi, flowInstStatusMap, flowTaskStatusMap, flowActionMap, collectSelfSelectNodes,
  flowStatesFromTasks, flowEndState, parseFileItems,
  type FlowFileItem, type FlowFormField, type FlowInstanceDetail, type FlowNode, type FlowUserOption,
} from '~/composables/useApi/flow'
import FormRender from '~/components/flow/FormRender.vue'
import FlowDiagram from '~/components/flow/FlowDiagram.vue'

definePageMeta({ title: '流程详情' })
defineOptions({ name: 'flow-detail' })

const route = useRoute()
const router = useRouter()
const api = useFlowApi()
const userStore = useUserStore()
const isAdmin = computed(() => userStore.isAdmin)

const instanceId = computed(() => Number(route.query.id || 0))
const detail = ref<FlowInstanceDetail | null>(null)
const loading = ref(false)
const comment = ref('')
const acting = ref(false)

const users = ref<FlowUserOption[]>([])

const fields = computed<FlowFormField[]>(() => parseSafe(detail.value?.formConf || '', []))
const formData = computed<Record<string, any>>(() => parseSafe(detail.value?.formData || '', {}))
const flowRoot = computed<FlowNode | null>(() => parseSafe<FlowNode | null>(detail.value?.flowConf || '', null))
const selfSelectNodes = computed(() => collectSelfSelectNodes(flowRoot.value))
const statusText = computed(() => flowInstStatusMap[detail.value?.instance.status || 0]?.text || '未知')
/** 最近一次驳回意见 (退回态提示) */
const lastRejectComment = computed(() => {
  const rs = detail.value?.records || []
  for (let i = rs.length - 1; i >= 0; i--) {
    if (rs[i].action === 'reject') return rs[i]
  }
  return null
})

/** 头图主题: 按实例状态着色 */
const heroTone = computed(() => {
  const s = detail.value?.instance.status || 0
  if (s === 1) return 'run'
  if (s === 2) return 'ok'
  if (s === 6 || s === 7) return 'back'
  if (s === 5) return 'stop'
  return 'off'
})

// ============================================================
// 流程进度: 只读流程图, 节点着色来自任务聚合 (公共辅助函数)
// ============================================================
const diagStates = computed(() => flowStatesFromTasks(detail.value?.tasks || []))
const diagEnd = computed(() => flowEndState(detail.value?.instance.status || 0))

// ============================================================
// 其余展示辅助
// ============================================================
function shortTime(s?: string | null) { return s ? s.slice(0, 16) : '' }

/** 返回: 详情页入口有多处 (我的审批/实例管理/流程定义/业务页), 有站内来路则回上一页;
 *  直接打开/新标签进入 (无来路) 时兜底回我的审批 */
function goBack() {
  if (window.history.state?.back) router.back()
  else navigateTo('/flow/center')
}

/** 表单只读展示值 */
function displayValue(key: string) {
  const v = formData.value?.[key]
  return v === undefined || v === null || v === '' ? '-' : String(v)
}

/** 图片/附件字段归一化值 */
function fileItems(key: string): FlowFileItem[] {
  return parseFileItems(formData.value?.[key])
}

/** 时间线头像首字 */
function avatarChar(name: string, operatorId: number) {
  return operatorId === 0 ? '系' : (name || '?').slice(0, 1)
}

/** 时间线动作配色 */
function recTone(a: string) {
  if (a === 'approve' || a === 'finish' || a === 'timeoutApprove') return 'ok'
  if (a === 'reject') return 'danger'
  if (a === 'resubmit' || a === 'back' || a === 'urge' || a === 'reduce' || a === 'withdraw' || a === 'timeoutRemind') return 'warn'
  if (a === 'cancel' || a === 'terminate') return 'off'
  if (a === 'transfer' || a === 'append' || a === 'delegate' || a === 'delegateResolve' || a === 'timeoutTransfer') return 'run'
  return 'run'
}

/** 重新提交卡片文案: 被驳回 / 已撤回 / 已撤销 三种可重提状态 */
const resubmitTitle = computed(() => {
  const s = detail.value?.instance.status
  if (s === 4) return '流程已撤销, 修改后可重新提交'
  if (s === 7) return '流程已撤回, 修改后可重新提交'
  return '被驳回, 修改后重新提交'
})

/** 重新提交表单 (退回态/已撤销态编辑) */
const resubmitForm = reactive({
  data: {} as Record<string, any>,
  selfSelects: {} as Record<string, number[]>,
})
const resubmitting = ref(false)

function parseSafe<T>(s: string, fallback: T): T {
  try { return s ? JSON.parse(s) as T : fallback } catch { return fallback }
}

async function load(autoRead = false) {
  if (!instanceId.value) return
  loading.value = true
  try {
    detail.value = await api.instDetail(instanceId.value)
    // 抄送待阅: 打开详情自动标记已读
    if (autoRead && detail.value.myCcTaskId) {
      try {
        await api.taskRead(detail.value.myCcTaskId)
        detail.value = await api.instDetail(instanceId.value)
      } catch {}
    }
    // 重提表单初始化: 表单数据 + 上轮自选审批人回填
    resubmitForm.data = { ...formData.value }
    resubmitForm.selfSelects = {}
    for (const [k, v] of Object.entries(detail.value.prevSelfSelects || {})) {
      resubmitForm.selfSelects[k] = [...(v || [])]
    }
  } finally { loading.value = false }
}

async function act(approve: boolean) {
  const taskId = detail.value?.myPendingTaskId
  if (!taskId) return
  if (!approve) { openReject(); return }
  await ElMessageBox.confirm('同意该申请?', '同意', { type: 'info' })
  acting.value = true
  try {
    await api.taskApprove(taskId, comment.value)
    ElMessage.success('已同意')
    comment.value = ''
    load()
  } catch {} finally { acting.value = false }
}

// ============================================================
// 驳回: 可选退回发起人 (修改后重提) 或退回到某个已审批节点
// ============================================================
const rejectVisible = ref(false)
const rejectTarget = ref('')
const rejectComment = ref('')

function openReject() {
  rejectTarget.value = ''
  // 顶部已填的审批意见带入驳回弹窗, 避免重复输入; 弹窗内可再修改
  rejectComment.value = comment.value.trim()
  rejectVisible.value = true
}

async function confirmReject() {
  const taskId = detail.value?.myPendingTaskId
  if (!taskId) { rejectVisible.value = false; return }
  if (!rejectComment.value.trim()) { ElMessage.warning('请填写驳回意见'); return }
  const target = detail.value?.rejectTargets?.find(t => t.nodeId === rejectTarget.value)
  acting.value = true
  try {
    await api.taskReject(taskId, rejectComment.value, rejectTarget.value || undefined)
    ElMessage.success(target ? `已驳回, 退回到「${target.nodeName}」` : '已驳回, 已退回发起人')
    rejectVisible.value = false
    comment.value = ''
    load()
  } catch {} finally { acting.value = false }
}

// ============================================================
// 转办: 将我的待办转给指定人处理 (原任务置已转出, 对方生成新待办)
// ============================================================
const transferVisible = ref(false)
const transferTarget = ref<number | undefined>(undefined)
const transferComment = ref('')

function openTransfer() {
  transferTarget.value = undefined
  transferComment.value = ''
  transferVisible.value = true
}

async function confirmTransfer() {
  const taskId = detail.value?.myPendingTaskId
  if (!taskId) { transferVisible.value = false; return }
  if (!transferTarget.value) { ElMessage.warning('请选择接办人'); return }
  acting.value = true
  try {
    await api.taskTransfer(taskId, transferTarget.value, transferComment.value.trim())
    ElMessage.success('已转办')
    transferVisible.value = false
    load()
  } catch {} finally { acting.value = false }
}

// ============================================================
// 委派: 我的待办交被委托人代办, 其提交意见后回到本人终审 (区别于转办换人)
// ============================================================
const delegateVisible = ref(false)
const delegateTarget = ref<number | undefined>(undefined)
const delegateComment = ref('')

function openDelegate() {
  delegateTarget.value = undefined
  delegateComment.value = ''
  delegateVisible.value = true
}

async function confirmDelegate() {
  const taskId = detail.value?.myPendingTaskId
  if (!taskId) { delegateVisible.value = false; return }
  if (!delegateTarget.value) { ElMessage.warning('请选择被委托人'); return }
  acting.value = true
  try {
    await api.taskDelegate(taskId, delegateTarget.value, delegateComment.value.trim())
    ElMessage.success('已委派, 待对方提交处理意见后回到你终审')
    delegateVisible.value = false
    load()
  } catch {} finally { acting.value = false }
}

/** 我的待办是否为被委派的代办任务 (只能提交处理意见, 不能直接表决) */
const isDelegatedTask = computed(() => (myTask.value?.delegateFromId || 0) > 0)
/** 委派发起人任务行 (代办横幅展示对方昵称) */
const delegatorTask = computed(() => {
  const from = myTask.value?.delegateFromId || 0
  return from ? (detail.value?.tasks || []).find(t => t.id === from) || null : null
})

async function confirmDelegateResolve() {
  const taskId = detail.value?.myPendingTaskId
  if (!taskId) return
  acting.value = true
  try {
    await api.taskDelegateResolve(taskId, comment.value.trim())
    ElMessage.success('已提交处理意见, 待原审批人终审')
    comment.value = ''
    load()
  } catch {} finally { acting.value = false }
}

// ============================================================
// 催办 (发起人, 运行中) / 终止 (管理员, 运行中)
// ============================================================
async function urge() {
  const r = await ElMessageBox.prompt('将向当前全部待办审批人发送催办提醒 (10 分钟内限一次)。', '催办', {
    confirmButtonText: '发送催办',
    cancelButtonText: '取消',
    inputType: 'textarea',
    inputPlaceholder: '催办说明 (选填)',
  }).catch(() => null)
  if (r === null) return
  try {
    await api.instUrge(instanceId.value, (r.value || '').trim())
    ElMessage.success('已发送催办提醒')
    load()
  } catch {}
}

async function terminate() {
  const r = await ElMessageBox.prompt('终止后待办全部作废, 流程立即结束且不可恢复。', '终止流程', {
    confirmButtonText: '确认终止',
    cancelButtonText: '取消',
    inputType: 'textarea',
    inputPlaceholder: '终止原因 (选填)',
  }).catch(() => null)
  if (r === null) return
  try {
    await api.instTerminate(instanceId.value, (r.value || '').trim())
    ElMessage.success('已终止流程')
    load()
  } catch {}
}

// ============================================================
// 加签/减签: 节点级人员调整 —— 我的待办为锚; 管理员可锚定当前节点任一待办
// ============================================================
const myTask = computed(() =>
  detail.value?.tasks?.find(t => t.id === detail.value?.myPendingTaskId) || null)
/** 加签/减签锚点任务: 我的待办优先; 否则管理员取当前节点任一待办 (后端同样放行) */
const signAnchorTask = computed(() => {
  if (myTask.value) return myTask.value
  if (!isAdmin.value || detail.value?.instance.status !== 1) return null
  return (detail.value?.tasks || []).find(t => t.nodeType === 1 && t.status === 1) || null
})
/** 锚点所在节点全部待办审批人 (减签候选) */
const nodePendingTasks = computed(() =>
  !signAnchorTask.value ? [] : (detail.value?.tasks || []).filter(
    t => t.nodeId === signAnchorTask.value!.nodeId && t.nodeType === 1 && t.status === 1))
/** 本轮已同意过该节点的人 (加签候选中禁用: 会签下重复加签会要求其二次同意, 后端同样拦截) */
const nodeApprovedIds = computed(() =>
  !signAnchorTask.value ? [] : (detail.value?.tasks || []).filter(
    t => t.nodeId === signAnchorTask.value!.nodeId && t.nodeType === 1 && t.status === 2).map(t => t.assigneeId))

const appendVisible = ref(false)
const appendUserIds = ref<number[]>([])
const appendComment = ref('')

function openAppend() {
  appendUserIds.value = []
  appendComment.value = ''
  appendVisible.value = true
}

async function confirmAppend() {
  const taskId = signAnchorTask.value?.id
  if (!taskId) { appendVisible.value = false; return }
  if (!appendUserIds.value.length) { ElMessage.warning('请选择加签人'); return }
  acting.value = true
  try {
    await api.taskAppend(taskId, appendUserIds.value, appendComment.value.trim())
    ElMessage.success('已加签, 当前节点转为会签')
    appendVisible.value = false
    load()
  } catch {} finally { acting.value = false }
}

const reduceVisible = ref(false)
const reduceUserIds = ref<number[]>([])
const reduceComment = ref('')

function openReduce() {
  reduceUserIds.value = []
  reduceComment.value = ''
  reduceVisible.value = true
}

async function confirmReduce() {
  const taskId = signAnchorTask.value?.id
  if (!taskId) { reduceVisible.value = false; return }
  if (!reduceUserIds.value.length) { ElMessage.warning('请选择要移除的审批人'); return }
  if (reduceUserIds.value.length >= nodePendingTasks.value.length) {
    ElMessage.warning('不能移除全部待办审批人, 至少保留一人')
    return
  }
  acting.value = true
  try {
    await api.taskReduce(taskId, reduceUserIds.value, reduceComment.value.trim())
    ElMessage.success('已减签')
    reduceVisible.value = false
    load()
  } catch {} finally { acting.value = false }
}

async function handleResubmit() {
  for (const n of selfSelectNodes.value) {
    if (!resubmitForm.selfSelects[n.id]?.length) {
      ElMessage.warning(`请在「${n.name}」中选择审批人`)
      return
    }
  }
  await ElMessageBox.confirm('重新提交流程将从头重走, 之前的审批记录会保留。', '重新提交', { type: 'info' })
  resubmitting.value = true
  try {
    await api.instResubmit(instanceId.value, {
      formData: resubmitForm.data,
      selfSelects: resubmitForm.selfSelects,
    })
    ElMessage.success('已重新提交')
    load()
  } catch {} finally { resubmitting.value = false }
}

async function cancel() {
  await ElMessageBox.confirm('撤销该流程? 未处理的待办将全部作废。', '撤销确认', { type: 'warning' })
  try {
    await api.instCancel(instanceId.value)
    ElMessage.success('已撤销')
    load()
  } catch {}
}

// 撤回: 尚无审批人同意时收回流程 (待修改重提, 区别于撤销终态)
async function withdraw() {
  await ElMessageBox.confirm(
    '撤回后当前待办作废, 流程回到你手中; 可修改表单后重新提交 (流程从头重走, 历史记录保留)。已有审批人同意时不可撤回。',
    '撤回确认', { type: 'warning', confirmButtonText: '确认撤回' },
  )
  try {
    await api.instWithdraw(instanceId.value)
    ElMessage.success('已撤回, 可修改后重新提交')
    load()
  } catch {}
}

function actionText(a: string) { return flowActionMap[a] || a }

onMounted(async () => {
  load(true)
  try {
    const opt = await api.options()
    users.value = opt.users || []
  } catch {}
})
</script>

<template>
  <div class="fd-page" v-loading="loading">
    <!-- 首次加载骨架 -->
    <div v-if="!detail" class="fd-skeleton">
      <el-skeleton :rows="4" animated />
    </div>

    <template v-if="detail">
      <!-- 状态头图 -->
      <div class="fd-hero" :class="`fd-hero--${heroTone}`">
        <div class="fd-hero__main">
          <div class="fd-hero__icon">
            <el-icon v-if="heroTone === 'run'"><Clock /></el-icon>
            <el-icon v-else-if="heroTone === 'ok'"><CircleCheck /></el-icon>
            <el-icon v-else-if="heroTone === 'back'"><EditPen /></el-icon>
            <el-icon v-else-if="heroTone === 'stop'"><Warning /></el-icon>
            <el-icon v-else><CircleClose /></el-icon>
          </div>
          <div class="fd-hero__body">
            <div class="fd-hero__title-row">
              <span class="fd-hero__title">{{ detail.instance.title }}</span>
              <span class="fd-hero__badge">{{ statusText }}</span>
            </div>
            <div class="fd-hero__meta">
              <div class="fd-hero__chip"><span class="k">流程</span><span class="v">{{ detail.instance.flowName }}</span></div>
              <div class="fd-hero__chip"><span class="k">发起人</span><span class="v">{{ detail.instance.startUserName }}</span></div>
              <div class="fd-hero__chip"><span class="k">发起时间</span><span class="v">{{ shortTime(detail.instance.createdAt) }}</span></div>
              <div v-if="detail.instance.currentNodes" class="fd-hero__chip">
                <span class="k">当前节点</span><span class="v">{{ detail.instance.currentNodes }}</span>
              </div>
              <div v-if="detail.instance.finishedAt" class="fd-hero__chip">
                <span class="k">结束时间</span><span class="v">{{ shortTime(detail.instance.finishedAt) }}</span>
              </div>
              <div class="fd-hero__chip"><span class="k">单号</span><span class="v">#{{ detail.instance.id }}</span></div>
            </div>
          </div>
        </div>
        <el-button round class="fd-hero__back" :icon="ArrowLeft" @click="goBack">返回</el-button>
      </div>

      <!-- 审批操作 -->
      <el-card v-if="detail.myPendingTaskId" class="fd-act fd-act--audit" shadow="never">
        <div class="fd-act__head">
          <el-icon><Stamp /></el-icon>
          <span>待你审批</span>
          <span class="fd-act__sub">该流程正在等待你的处理</span>
        </div>
        <!-- 被委派的代办任务: 只提交处理意见, 表决权在原审批人 -->
        <el-alert v-if="isDelegatedTask" type="warning" :closable="false" show-icon style="margin-bottom:12px"
          :title="`「${delegatorTask?.assigneeName || '原审批人'}」将该审批委派给你代办`"
          description="提交处理意见后任务会回到对方做最终同意/驳回; 你不能直接代替对方表决。" />
        <el-input v-model="comment" type="textarea" :rows="3" maxlength="500" show-word-limit
          :placeholder="isDelegatedTask ? '处理意见 (选填, 将随委派回执通知原审批人)' : '审批意见 (同意时选填, 点驳回时会带入驳回弹窗)'" />
        <div class="fd-act__btns">
          <template v-if="isDelegatedTask">
            <el-button v-permission="'flow:task:handle'" type="primary" :icon="Check" :loading="acting" @click="confirmDelegateResolve">委派处理</el-button>
          </template>
          <template v-else>
            <el-button v-permission="'flow:task:handle'" plain :icon="Share" @click="openTransfer">转办</el-button>
            <el-button v-permission="'flow:task:handle'" plain :icon="Connection" @click="openDelegate">委派</el-button>
            <el-button v-permission="'flow:task:handle'" plain :icon="Plus" @click="openAppend">加签</el-button>
            <el-button v-permission="'flow:task:handle'" plain :icon="Minus" @click="openReduce">减签</el-button>
            <el-button v-permission="'flow:task:handle'" type="danger" plain :icon="Close" :loading="acting" @click="openReject">驳回</el-button>
            <el-button v-permission="'flow:task:handle'" type="primary" :icon="Check" :loading="acting" @click="act(true)">同意</el-button>
          </template>
        </div>
      </el-card>

      <!-- 加签弹窗: 当前节点追加必要审批人 (转为会签) -->
      <el-dialog v-model="appendVisible" title="加签" width="480px" append-to-body destroy-on-close>
        <el-alert type="info" :closable="false" show-icon style="margin-bottom:14px"
          title="加签后当前节点转为会签: 原审批人与新加人均须同意, 节点才通过; 任一人驳回仍整单退回。" />
        <el-form label-width="90px">
          <el-form-item label="加签给" required>
            <el-select v-model="appendUserIds" multiple filterable placeholder="选择要加入的审批人" style="width:100%">
              <el-option v-for="u in users" :key="u.id" :label="u.nickname || u.username" :value="u.id"
                :disabled="!!nodePendingTasks.some(t => t.assigneeId === u.id) || nodeApprovedIds.includes(u.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="加签说明">
            <el-input v-model="appendComment" type="textarea" :rows="3" maxlength="500" show-word-limit
              placeholder="选填, 将记入流转记录" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="appendVisible = false">取消</el-button>
          <el-button type="primary" :loading="acting" @click="confirmAppend">确认加签</el-button>
        </template>
      </el-dialog>

      <!-- 减签弹窗: 移除当前节点待办审批人 (至少保留一人) -->
      <el-dialog v-model="reduceVisible" title="减签" width="480px" append-to-body destroy-on-close>
        <el-alert type="info" :closable="false" show-icon style="margin-bottom:14px"
          title="移除后其待办作废; 已同意的不回退。该节点至少保留一位待办审批人。" />
        <el-form label-width="90px">
          <el-form-item label="移除谁" required>
            <el-select v-model="reduceUserIds" multiple filterable placeholder="选择要移除的审批人" style="width:100%">
              <el-option v-for="t in nodePendingTasks" :key="t.assigneeId" :label="t.assigneeName" :value="t.assigneeId"
                :disabled="nodePendingTasks.length <= 1" />
            </el-select>
          </el-form-item>
          <el-form-item label="减签说明">
            <el-input v-model="reduceComment" type="textarea" :rows="3" maxlength="500" show-word-limit
              placeholder="选填, 将记入流转记录" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="reduceVisible = false">取消</el-button>
          <el-button type="danger" :loading="acting" @click="confirmReduce">确认减签</el-button>
        </template>
      </el-dialog>

      <!-- 转办弹窗: 选择接办人 -->
      <el-dialog v-model="transferVisible" title="转办" width="460px" append-to-body destroy-on-close>
        <el-form label-width="90px">
          <el-form-item label="转办给" required>
            <el-select v-model="transferTarget" filterable placeholder="选择接办人" style="width:100%">
              <el-option v-for="u in users" :key="u.id"
                :label="u.nickname || u.username" :value="u.id"
                :disabled="u.id === userStore.userInfo?.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="转办说明">
            <el-input v-model="transferComment" type="textarea" :rows="3" maxlength="500" show-word-limit
              placeholder="选填, 将随转办通知对方" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="transferVisible = false">取消</el-button>
          <el-button type="primary" :loading="acting" @click="confirmTransfer">确认转办</el-button>
        </template>
      </el-dialog>

      <!-- 委派弹窗: 交被委托人代办, 处理后回到本人终审 -->
      <el-dialog v-model="delegateVisible" title="委派" width="460px" append-to-body destroy-on-close>
        <el-alert type="info" :closable="false" show-icon style="margin-bottom:14px"
          title="委派与转办不同: 被委托人提交处理意见后, 任务回到你做最终同意/驳回; 对方不能代替你表决。" />
        <el-form label-width="90px">
          <el-form-item label="委派给" required>
            <el-select v-model="delegateTarget" filterable placeholder="选择被委托人" style="width:100%">
              <el-option v-for="u in users" :key="u.id"
                :label="u.nickname || u.username" :value="u.id"
                :disabled="u.id === userStore.userInfo?.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="委派说明">
            <el-input v-model="delegateComment" type="textarea" :rows="3" maxlength="500" show-word-limit
              placeholder="选填, 将随委派通知对方" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="delegateVisible = false">取消</el-button>
          <el-button type="primary" :loading="acting" @click="confirmDelegate">确认委派</el-button>
        </template>
      </el-dialog>

      <!-- 驳回弹窗: 退回发起人 / 退回到指定已审批节点 -->
      <el-dialog v-model="rejectVisible" title="驳回" width="500px" append-to-body destroy-on-close>
        <el-alert type="info" :closable="false" show-icon style="margin-bottom:14px"
          title="驳回仅作用于当前节点 (同节点其余待办一并作废)。退回到指定节点后实例保持运行, 从该节点重新处理; 退回发起人则修改表单后从头重走。" />
        <el-form label-width="90px">
          <el-form-item label="驳回到">
            <el-radio-group v-model="rejectTarget">
              <el-radio value="">退回发起人 (修改后重新提交)</el-radio>
              <el-radio v-for="t in detail?.rejectTargets || []" :key="t.nodeId" :value="t.nodeId">
                退回到「{{ t.nodeName }}」
              </el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="驳回意见" required>
            <el-input v-model="rejectComment" type="textarea" :rows="3" maxlength="500" show-word-limit
              placeholder="必填, 将通知发起人或目标节点处理人" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="rejectVisible = false">取消</el-button>
          <el-button type="danger" :loading="acting" @click="confirmReject">确认驳回</el-button>
        </template>
      </el-dialog>

      <!-- 驳回退回/已撤销: 发起人修改后重新提交 -->
      <el-card v-if="detail.canResubmit" class="fd-act fd-act--resubmit" shadow="never">
        <div class="fd-act__head">
          <el-icon><EditPen /></el-icon>
          <span>{{ resubmitTitle }}</span>
          <span class="fd-act__sub">流程将从头重走, 之前的审批记录保留</span>
        </div>
        <el-alert v-if="lastRejectComment && detail.instance.status !== 4 && detail.instance.status !== 7" type="warning" :closable="false" show-icon style="margin-bottom:12px"
          :title="`${lastRejectComment.operatorName} 驳回 (${lastRejectComment.nodeName})`"
          :description="lastRejectComment.comment || '未填写意见'" />
        <FormRender v-model="resubmitForm.data" :fields="fields" />
        <template v-if="selfSelectNodes.length">
          <el-divider content-position="left">选择审批人</el-divider>
          <el-form label-width="110px">
            <el-form-item v-for="n in selfSelectNodes" :key="n.id" :label="n.name" required>
              <el-select v-model="resubmitForm.selfSelects[n.id]" multiple filterable placeholder="选择审批人" style="width:100%">
                <el-option v-for="u in users" :key="u.id" :label="u.nickname || u.username" :value="u.id" />
              </el-select>
            </el-form-item>
          </el-form>
        </template>
        <div class="fd-act__btns">
          <el-button v-permission="'flow:instance:cancel'" plain :icon="RefreshLeft" @click="cancel">撤销流程</el-button>
          <el-button v-permission="'flow:instance:start'" type="primary" :icon="Position" :loading="resubmitting" @click="handleResubmit">重新提交</el-button>
        </div>
      </el-card>

      <!-- 运行中操作栏: 发起人可撤回/撤销/催办, 管理员可终止/加签/减签 -->
      <div v-if="detail.instance.status === 1 && (detail.canCancel || detail.canWithdraw || isAdmin)" class="fd-cancelbar">
        <span>
          流程运行中<template v-if="detail.canCancel || detail.canWithdraw">, 发起人可<template v-if="detail.canWithdraw">撤回(尚无审批时)或</template>撤销/催办</template><template v-if="isAdmin">, 管理员可终止/加签/减签</template>
        </span>
        <div style="display:flex;gap:8px">
          <el-button v-if="detail.canCancel" plain size="small" :icon="Bell" @click="urge">催办</el-button>
          <el-button v-if="detail.canWithdraw" v-permission="'flow:instance:cancel'" type="warning" plain size="small" :icon="RefreshLeft" @click="withdraw">撤回</el-button>
          <el-button v-if="detail.canCancel" v-permission="'flow:instance:cancel'" type="warning" plain size="small" :icon="RefreshLeft" @click="cancel">撤销流程</el-button>
          <!-- 管理员节点级人员调整 (锚定当前节点任一待办); 本人有待办时走上方"待你审批"卡片入口 -->
          <el-button v-if="isAdmin && !myTask && signAnchorTask" v-permission="'flow:task:handle'" plain size="small" :icon="Plus" @click="openAppend">加签</el-button>
          <el-button v-if="isAdmin && !myTask && signAnchorTask" v-permission="'flow:task:handle'" plain size="small" :icon="Minus" @click="openReduce">减签</el-button>
          <el-button v-if="isAdmin" type="danger" plain size="small" :icon="VideoPause" @click="terminate">终止流程</el-button>
        </div>
      </div>

      <!-- 流程进度 (流程图形态, 条件分支与实际走向一目了然) -->
      <el-card class="fd-card" shadow="never">
        <template #header><div class="fd-sec">流程进度</div></template>
        <FlowDiagram :root="flowRoot" :states="diagStates"
          :start-user="detail.instance.startUserName"
          :end-state="diagEnd.state" :end-text="diagEnd.text" />
      </el-card>

      <div class="fd-grid">
        <!-- 申请表单 (只读; 退回态由上方重提卡片编辑) -->
        <el-card class="fd-card" shadow="never">
          <template #header><div class="fd-sec">申请表单</div></template>
          <el-descriptions v-if="!detail.canResubmit && fields.length" class="fd-desc" :column="2" border>
            <el-descriptions-item v-for="f in fields" :key="f.key" :label="f.label"
              :span="f.type === 'textarea' || f.type === 'file' || f.type === 'image' ? 2 : 1">
              <!-- 图片: 缩略图可预览 -->
              <div v-if="f.type === 'image' && fileItems(f.key).length" class="fd-desc__imgs">
                <el-image v-for="(it, i) in fileItems(f.key)" :key="i" class="fd-desc__img" :src="it.url" fit="cover"
                  :preview-src-list="fileItems(f.key).map(x => x.url)" :initial-index="i" preview-teleported />
              </div>
              <!-- 附件: 链接列表 -->
              <div v-else-if="f.type === 'file' && fileItems(f.key).length" class="fd-desc__files">
                <el-link v-for="(it, i) in fileItems(f.key)" :key="i" type="primary" :href="it.url" target="_blank">
                  {{ it.name }}
                </el-link>
              </div>
              <span v-else class="fd-desc__val" :class="{ 'is-pre': f.type === 'textarea', 'is-empty': displayValue(f.key) === '-' }">
                {{ displayValue(f.key) }}
              </span>
            </el-descriptions-item>
          </el-descriptions>
          <el-empty v-else description="退回态: 请在上方修改表单" :image-size="60" />
        </el-card>

        <!-- 流转时间线 -->
        <el-card class="fd-card" shadow="never">
          <template #header><div class="fd-sec">流转记录</div></template>
          <el-timeline v-if="detail.records?.length" class="fd-tl">
            <el-timeline-item v-for="r in detail.records" :key="r.id">
              <template #dot>
                <div class="fd-tl__avatar" :class="`is-${recTone(r.action)}`">{{ avatarChar(r.operatorName, r.operatorId) }}</div>
              </template>
              <div class="fd-tl__card" :class="`is-${recTone(r.action)}`">
                <div class="fd-tl__head">
                  <span class="act">{{ actionText(r.action) }}</span>
                  <span class="op">{{ r.operatorId === 0 ? '系统' : r.operatorName }}</span>
                  <el-tag v-if="r.nodeName" size="small" effect="plain" type="info">{{ r.nodeName }}</el-tag>
                  <span class="time">{{ r.createdAt }}</span>
                </div>
                <div v-if="r.comment" class="fd-tl__comment">{{ r.comment }}</div>
              </div>
            </el-timeline-item>
          </el-timeline>
          <el-empty v-else description="暂无记录" :image-size="60" />
        </el-card>
      </div>

      <!-- 任务明细 -->
      <el-card class="fd-card" shadow="never">
        <template #header><div class="fd-sec">审批任务</div></template>
        <el-table :data="detail.tasks || []" border stripe size="small">
          <el-table-column prop="nodeName" label="节点" width="140" />
          <el-table-column label="类型" width="80">
            <template #default="{ row }">
              <el-tag size="small" effect="plain" :type="row.nodeType === 1 ? 'primary' : 'info'">
                {{ row.nodeType === 1 ? '审批' : '抄送' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="签核" width="80">
            <template #default="{ row }">{{ row.signType === 2 ? '会签' : '或签' }}</template>
          </el-table-column>
          <el-table-column label="处理人" width="130">
            <template #default="{ row }">
              <span>{{ row.assigneeName }}</span>
              <el-tag v-if="row.delegateFromId" size="small" effect="plain" type="warning" style="margin-left:4px">委办</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <el-tag :type="flowTaskStatusMap[row.status]?.tag || 'info'" size="small">
                {{ row.nodeType === 2 && row.status === 2 ? '已阅' : flowTaskStatusMap[row.status]?.text || '未知' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="comment" label="意见" min-width="140" show-overflow-tooltip />
          <el-table-column prop="receiveTime" label="到达时间" width="165" />
          <el-table-column label="办理期限" width="165">
            <template #default="{ row }">
              <span v-if="row.dueTime" :class="{ 'fd-overdue': row.status === 1 && new Date(row.dueTime) < new Date() }">
                {{ row.dueTime }}
              </span>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column prop="actedAt" label="处理时间" width="165" />
        </el-table>
      </el-card>
    </template>
  </div>
</template>

<style scoped>
.fd-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 1280px;
  margin: 0 auto;
}
.fd-skeleton {
  background: var(--el-bg-color);
  border-radius: 8px;
  padding: 24px;
}

/* ---------- 状态头图 ---------- */
.fd-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 22px 24px;
  border-radius: 8px;
  color: #fff;
  background: linear-gradient(120deg, var(--el-color-primary), #337ecc);
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.08);
}
.fd-hero--ok { background: linear-gradient(120deg, var(--el-color-success), #529b2e); }
.fd-hero--back { background: linear-gradient(120deg, var(--el-color-danger), #c45656); }
.fd-hero--stop { background: linear-gradient(120deg, var(--el-color-warning), #b88230); }
.fd-hero--off { background: linear-gradient(120deg, var(--el-text-color-secondary), #73767a); }
.fd-hero__main {
  display: flex;
  align-items: center;
  gap: 16px;
  min-width: 0;
}
.fd-hero__icon {
  flex: none;
  width: 52px;
  height: 52px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 26px;
  background: rgba(255, 255, 255, 0.18);
  border: 1px solid rgba(255, 255, 255, 0.35);
}
.fd-hero__body { min-width: 0; }
.fd-hero__title-row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.fd-hero__title {
  font-size: 19px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fd-hero__badge {
  flex: none;
  padding: 2px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  background: rgba(255, 255, 255, 0.22);
  border: 1px solid rgba(255, 255, 255, 0.4);
}
.fd-hero__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 18px;
  margin-top: 10px;
  font-size: 12px;
}
.fd-hero__chip { display: flex; gap: 6px; align-items: baseline; }
.fd-hero__chip .k { opacity: 0.75; }
.fd-hero__chip .v { font-weight: 600; }
.fd-hero__back {
  flex: none;
  color: #fff;
  background: rgba(255, 255, 255, 0.14);
  border-color: rgba(255, 255, 255, 0.45);
}
.fd-hero__back:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.26);
  border-color: #fff;
}

/* ---------- 区块卡片 ---------- */
.fd-card :deep(.el-card__header) {
  padding: 13px 16px;
}
.fd-sec {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 14px;
}
.fd-sec::before {
  content: '';
  width: 4px;
  height: 14px;
  border-radius: 2px;
  background: var(--el-color-primary);
}
.fd-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  align-items: stretch;
}
@media (max-width: 992px) {
  .fd-grid { grid-template-columns: 1fr; }
}

/* ---------- 操作区 ---------- */
.fd-act { border-left: 3px solid var(--el-color-primary); }
.fd-act--audit { background: linear-gradient(180deg, var(--el-color-primary-light-9), var(--el-bg-color) 60%); }
.fd-act--resubmit {
  border-left-color: var(--el-color-warning);
  background: linear-gradient(180deg, var(--el-color-warning-light-9), var(--el-bg-color) 60%);
}
.fd-act__head {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  margin-bottom: 12px;
  color: var(--el-text-color-primary);
}
.fd-act__head .el-icon { color: var(--el-color-primary); }
.fd-act--resubmit .fd-act__head .el-icon { color: var(--el-color-warning); }
.fd-act__sub { font-weight: 400; font-size: 12px; color: var(--el-text-color-secondary); }
.fd-act__btns {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 12px;
}
.fd-cancelbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-lighter);
  font-size: 13px;
}

/* ---------- 申请表单 (只读描述表) ---------- */
.fd-desc :deep(.el-descriptions__label) {
  width: 110px;
  min-width: 110px;
  color: var(--el-text-color-secondary);
  font-weight: 400;
  background: var(--el-fill-color-lighter);
}
.fd-desc__val {
  color: var(--el-text-color-primary);
  word-break: break-all;
}
.fd-desc__val.is-pre { white-space: pre-wrap; }
.fd-desc__val.is-empty { color: var(--el-text-color-disabled); }
.fd-desc__imgs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.fd-desc__img {
  width: 80px;
  height: 80px;
  border-radius: 6px;
  border: 1px solid var(--el-border-color-light);
  cursor: zoom-in;
}
.fd-desc__files {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: flex-start;
}

/* ---------- 流转时间线 ---------- */
.fd-tl { padding: 4px 4px 0 2px; }
.fd-tl :deep(.el-timeline-item) { padding-bottom: 18px; }
.fd-tl :deep(.el-timeline-item:last-child) { padding-bottom: 4px; }
/* EP 的竖线默认 left:4px 是按 10px 圆点定位的, 与 30px 头像圆错位 —— 重定到头像中心 (15px-1px 线宽) */
.fd-tl :deep(.el-timeline-item__tail) { left: 14px; }
.fd-tl :deep(.el-timeline-item:last-child .el-timeline-item__tail) { display: none; }
.fd-tl :deep(.el-timeline-item__wrapper) { padding-left: 44px; top: 0; }
.fd-tl__avatar {
  box-sizing: border-box;
  width: 30px;
  height: 30px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  background: var(--el-color-primary);
  border: 2px solid #fff;
  box-shadow: 0 0 0 1px var(--el-border-color-light);
}
.fd-tl__avatar.is-ok { background: var(--el-color-success); }
.fd-tl__avatar.is-danger { background: var(--el-color-danger); }
.fd-tl__avatar.is-warn { background: var(--el-color-warning); }
.fd-tl__avatar.is-off { background: var(--el-text-color-secondary); }
.fd-tl__card {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 10px 14px;
  transition: border-color .2s, box-shadow .2s;
}
.fd-tl__card:hover {
  border-color: var(--el-color-primary-light-8);
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.08);
}
.fd-tl__head {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  min-width: 0;
}
.fd-tl__head .act { font-weight: 600; font-size: 13px; color: var(--el-color-primary); }
.fd-tl__card.is-ok .act { color: var(--el-color-success); }
.fd-tl__card.is-danger .act { color: var(--el-color-danger); }
.fd-tl__card.is-warn .act { color: var(--el-color-warning); }
.fd-tl__card.is-off .act { color: var(--el-text-color-secondary); }
.fd-tl__head .op { color: var(--el-text-color-regular); font-size: 13px; }
.fd-tl__head .time {
  margin-left: auto;
  flex: none;
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}
.fd-tl__comment {
  margin-top: 8px;
  background: var(--el-fill-color-lighter);
  border-radius: 4px;
  padding: 6px 10px;
  font-size: 12px;
  color: var(--el-text-color-regular);
  white-space: pre-wrap;
  word-break: break-all;
}

/* 逾期待办的办理期限标红 */
.fd-overdue {
  color: var(--el-color-danger);
  font-weight: 600;
}
</style>
