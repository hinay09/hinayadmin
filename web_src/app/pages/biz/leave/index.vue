<script setup lang="ts">
/**
 * 请假申请 —— 业务型流程审批 Demo。
 *
 * 业务与流程分离: 单据增删改是业务自己的事 (草稿态, 发起前自由操作);
 * 「提交审批」走公共弹窗 <FlowSubmitDialog :show-form=false> —— 流程确认与
 * 自选审批人由公共组件负责, 表单快照由后端从业务表组装 (StartForBiz 发起);
 * 「审批状态」列的每次变化均由引擎回调写回 flow_status; 审批/驳回统一在
 * 「审批中心 → 我的审批」详情页处理。
 */
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { Search, Plus, Edit, Delete, Promotion, RefreshLeft, View } from '@element-plus/icons-vue'
import FlowHistoryDialog from '~/components/flow/FlowHistoryDialog.vue'
import FlowSubmitDialog from '~/components/flow/FlowSubmitDialog.vue'
import { useUserStore } from '~/stores/user'
import type { FlowSubmitPayload } from '~/composables/useApi/flow'
import {
  useLeaveApi,
  leaveTypeMap, bizFlowStatusMap, leaveEditable, leaveCancelable,
  type LeaveItem, type LeaveFormData,
} from '~/composables/useApi'

definePageMeta({ title: '请假申请' })
defineOptions({ name: 'biz-leave' })

const api = useLeaveApi()
const userStore = useUserStore()
const isAdmin = computed(() => userStore.isAdmin)

// ============================================================
// 列表
// ============================================================
const loading = ref(false)
const list = ref<LeaveItem[]>([])
const total = ref(0)
const query = reactive({
  leaveType: undefined as number | undefined,
  flowStatus: undefined as number | undefined,
  mine: undefined as number | undefined,
  page: 1,
  pageSize: 10,
})

async function load() {
  loading.value = true
  try {
    const res = await api.list({ ...query })
    list.value = res.list || []
    total.value = res.total || 0
  } finally { loading.value = false }
}

// ============================================================
// 新增/编辑 (被退回/已撤销的单子编辑后可再次提交)
// ============================================================
const drawerVisible = ref(false)
const drawerTitle = ref('新增请假申请')
const isEdit = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()
const form = reactive<LeaveFormData>({ leaveType: 1, startDate: '', endDate: '', days: 1, reason: '' })
const rules = {
  leaveType: [{ required: true, message: '请选择请假类型', trigger: 'change' }],
  startDate: [{ required: true, message: '请选择开始日期', trigger: 'change' }],
  endDate: [{ required: true, message: '请选择结束日期', trigger: 'change' }],
  days: [{ required: true, message: '请填写请假天数', trigger: 'blur' }],
  reason: [{ required: true, message: '请填写请假事由', trigger: 'blur' }],
}
const editingId = ref(0)

function resetForm() {
  Object.assign(form, { leaveType: 1, startDate: '', endDate: '', days: 1, reason: '' })
  editingId.value = 0
}

function openCreate() {
  resetForm()
  isEdit.value = false
  drawerTitle.value = '新增请假申请'
  drawerVisible.value = true
}

function openEdit(row: LeaveItem) {
  resetForm()
  Object.assign(form, {
    leaveType: row.leaveType, startDate: row.startDate, endDate: row.endDate,
    days: row.days, reason: row.reason,
  })
  editingId.value = row.id
  isEdit.value = true
  drawerTitle.value = '修改请假申请'
  drawerVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    if (isEdit.value) {
      await api.update(editingId.value, { ...form })
      ElMessage.success('已保存, 可重新提交审批')
    } else {
      await api.create({ ...form })
      ElMessage.success('已保存为草稿')
    }
    drawerVisible.value = false
    load()
  } catch {} finally { submitting.value = false }
}

async function handleDelete(row: LeaveItem) {
  await ElMessageBox.confirm('删除该请假申请?', '删除确认', { type: 'warning' })
  try {
    await api.remove(row.id)
    ElMessage.success('已删除')
    load()
  } catch {}
}

// ============================================================
// 流程动作: 公共弹窗提交 (业务与流程分离) / 撤销 / 查看审批
// ============================================================

/** 提交审批走公共弹窗 FlowSubmitDialog: 流程确认/自选审批人由弹窗负责,
 *  业务表单数据不重复录入 (后端 submit 时从业务表组装快照) */
const submitVisible = ref(false)
const submitRow = ref<LeaveItem | null>(null)
const flowSubmitting = ref(0)

function openSubmit(row: LeaveItem) {
  submitRow.value = row
  submitVisible.value = true
}

/** FlowSubmitDialog 的业务提交钩子: 弹窗校验通过后把产出交给业务 API */
async function onFlowSubmit(payload: FlowSubmitPayload): Promise<number | void> {
  if (!submitRow.value) return
  const res = await api.submit(submitRow.value.id, payload.selfSelects)
  return res.flowInstance
}

/** 提交成功: 刷新列表并跳流程详情页看进度 */
function onSubmitted(instanceId: number) {
  load()
  if (instanceId) {
    navigateTo({ path: '/flow/detail', query: { id: String(instanceId) } })
  }
}

async function handleCancelFlow(row: LeaveItem) {
  await ElMessageBox.confirm('撤销后流程立即结束 (可在详情页重新提交), 确认撤销?', '撤销审批', { type: 'warning' })
  flowSubmitting.value = row.id
  try {
    await api.cancel(row.id)
    ElMessage.success('已撤销')
    load()
  } catch {} finally { flowSubmitting.value = 0 }
}

// 审批历史弹窗 (只需实例 ID)
const historyVisible = ref(false)
const historyId = ref(0)

onMounted(load)
</script>

<template>
  <div class="page">
    <el-card>
      <el-alert type="info" :closable="false" show-icon style="margin-bottom:12px"
        title="业务型审批 Demo: 业务与流程分离 —— 单据增删改在发起审批前自由操作; 「提交审批」走公共提交弹窗, 表单快照由后端从业务表组装; 状态列每次变化均由审批引擎回调写回; 审批/驳回请在「审批中心-我的审批」处理。" />

      <el-form inline @submit.prevent>
        <el-form-item label="类型">
          <el-select v-model="query.leaveType" clearable placeholder="全部" style="width:110px">
            <el-option v-for="(name, t) in leaveTypeMap" :key="t" :label="name" :value="Number(t)" />
          </el-select>
        </el-form-item>
        <el-form-item label="审批状态">
          <el-select v-model="query.flowStatus" clearable placeholder="全部" style="width:110px">
            <el-option v-for="(s, k) in bizFlowStatusMap" :key="k" :label="s.text" :value="Number(k)" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="isAdmin" label="范围">
          <el-switch v-model="query.mine" :active-value="1" :inactive-value="undefined" active-text="只看我的" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="() => { query.page = 1; load() }">查询</el-button>
          <el-button v-permission="'biz:leave:create'" type="success" :icon="Plus" @click="openCreate">申请请假</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="list" border stripe style="margin-top:8px">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column v-if="isAdmin" prop="createName" label="申请人" width="110" />
        <el-table-column label="类型" width="80">
          <template #default="{ row }">
            <el-tag size="small">{{ leaveTypeMap[row.leaveType] || '其他' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="请假时间" width="200">
          <template #default="{ row }">{{ row.startDate }} ~ {{ row.endDate }}</template>
        </el-table-column>
        <el-table-column prop="days" label="天数" width="70" align="center" />
        <el-table-column prop="reason" label="事由" min-width="150" show-overflow-tooltip />
        <el-table-column label="审批状态" width="110">
          <template #default="{ row }">
            <template v-if="row.flowInstance">
              <el-tag :type="bizFlowStatusMap[row.flowStatus]?.tag || 'info'">
                {{ bizFlowStatusMap[row.flowStatus]?.text || '未知' }}
              </el-tag>
            </template>
            <el-tag v-else type="info" effect="plain">草稿</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="申请时间" width="170" />
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <el-button v-if="leaveEditable(row)" v-permission="'biz:leave:submit'" link type="primary"
              :icon="Promotion" @click="openSubmit(row)">
              {{ row.flowInstance ? '重新提交' : '提交审批' }}
            </el-button>
            <el-button v-if="leaveCancelable(row)" v-permission="'biz:leave:cancel'" link type="warning"
              :icon="RefreshLeft" :loading="flowSubmitting === row.id" @click="handleCancelFlow(row)">撤销</el-button>
            <el-button v-if="leaveEditable(row)" v-permission="'biz:leave:update'" link type="primary"
              :icon="Edit" @click="openEdit(row)">编辑</el-button>
            <el-button v-if="leaveEditable(row)" v-permission="'biz:leave:delete'" link type="danger"
              :icon="Delete" @click="handleDelete(row)">删除</el-button>
            <el-button v-if="row.flowInstance" link type="primary" :icon="View"
              @click="navigateTo({ path: '/flow/detail', query: { id: String(row.flowInstance) } })">
              {{ row.flowStatus === 2 ? '去处理' : '查看审批' }}
            </el-button>
            <el-button v-if="row.flowInstance" link @click="historyId = row.flowInstance; historyVisible = true">历史</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination v-model:current-page="query.page" v-model:page-size="query.pageSize"
        :total="total" :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next, jumper"
        style="margin-top:16px;justify-content:flex-end"
        @current-change="load" @size-change="load" />
    </el-card>

    <!-- 新增/编辑 -->
    <el-drawer v-model="drawerVisible" :title="drawerTitle" size="440px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="请假类型" prop="leaveType">
          <el-radio-group v-model="form.leaveType">
            <el-radio v-for="(name, t) in leaveTypeMap" :key="t" :value="Number(t)">{{ name }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="开始日期" prop="startDate">
          <el-date-picker v-model="form.startDate" type="date" value-format="YYYY-MM-DD"
            placeholder="选择日期" style="width:100%" />
        </el-form-item>
        <el-form-item label="结束日期" prop="endDate">
          <el-date-picker v-model="form.endDate" type="date" value-format="YYYY-MM-DD"
            :disabled-date="(d: Date) => form.startDate && d.getTime() < new Date(form.startDate).getTime()"
            placeholder="选择日期" style="width:100%" />
        </el-form-item>
        <el-form-item label="请假天数" prop="days">
          <el-input-number v-model="form.days" :min="0.5" :max="99.5" :step="0.5" step-strictly />
        </el-form-item>
        <el-form-item label="请假事由" prop="reason">
          <el-input v-model="form.reason" type="textarea" :rows="4" maxlength="200" show-word-limit />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="drawerVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-drawer>

    <!-- 审批历史 -->
    <FlowHistoryDialog v-model="historyVisible" :instance-id="historyId" />

    <!-- 提交审批公共弹窗: 业务模式 (:show-form=false, 表单数据由后端从业务表组装);
         流程定义将来加"发起人自选"节点时, 弹窗自动渲染选人 UI, 本页零改动 -->
    <FlowSubmitDialog v-model="submitVisible" flow-key="biz_leave"
      :show-form="false" :show-title="false"
      :handler="onFlowSubmit" @submitted="onSubmitted" />
  </div>
</template>
