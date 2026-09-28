<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Search, Refresh, Plus, Edit, Delete, VideoPlay, Tickets } from '@element-plus/icons-vue'
import { useJobApi } from '~/composables/useApi'
import { useUserStore } from '~/stores/user'

definePageMeta({ title: '定时任务' })
defineOptions({ name: 'system-jobs' })

const api = useJobApi()
const userStore = useUserStore()
const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const handlers = ref<string[]>([])
const query = reactive({ keyword: '', page: 1, pageSize: 10 })

// gcron 表达式为 6 位 (秒 分 时 日 月 周), 常用预设降低手写门槛
const cronPresets = [
  { label: '每30秒', value: '*/30 * * * * *' },
  { label: '每分钟', value: '0 * * * * *' },
  { label: '每5分钟', value: '0 */5 * * * *' },
  { label: '每小时', value: '0 0 * * * *' },
  { label: '每天 02:00', value: '0 0 2 * * *' },
  { label: '每周一 09:00', value: '0 0 9 * * 1' },
]

// ---- 任务表单 ----
const dialogVisible = ref(false)
const submitting = ref(false)
const editingId = ref(0)
const formRef = ref<FormInstance>()
const form = reactive({
  name: '',
  handler: '',
  cronExpr: '',
  params: '',
  remark: '',
})
const formRules: FormRules = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  handler: [{ required: true, message: '请选择处理器', trigger: 'change' }],
  cronExpr: [{ required: true, message: '请输入 cron 表达式', trigger: 'blur' }],
}

function openCreate() {
  editingId.value = 0
  Object.assign(form, { name: '', handler: '', cronExpr: '', params: '', remark: '' })
  dialogVisible.value = true
}

function openEdit(row: any) {
  editingId.value = row.id
  Object.assign(form, {
    name: row.name,
    handler: row.handler,
    cronExpr: row.cronExpr,
    params: row.params || '',
    remark: row.remark || '',
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    if (editingId.value) {
      await api.update(editingId.value, { ...form })
      ElMessage.success('修改成功')
    }
    else {
      await api.create({ ...form })
      ElMessage.success('新增成功, 任务默认为暂停状态')
    }
    dialogVisible.value = false
    loadList()
  }
  catch (e: any) {
    ElMessage.error(e?.message || '提交失败')
  }
  finally {
    submitting.value = false
  }
}

async function loadList() {
  loading.value = true
  try {
    const res = await api.list({ ...query })
    list.value = res.list || []
    total.value = res.total || 0
  }
  finally {
    loading.value = false
  }
}

function resetQuery() {
  query.keyword = ''
  query.page = 1
  loadList()
}

// ---- 一键启停 / 立即执行 / 删除 ----
const canStatus = userStore.hasPermission('system:job:status')

async function handleStatusChange(row: any) {
  try {
    await api.changeStatus(row.id, row.status)
    ElMessage.success(row.status === 1 ? '已启动' : '已暂停')
  }
  catch (e: any) {
    row.status = row.status === 1 ? 0 : 1 // 失败回滚开关
    ElMessage.error(e?.message || '操作失败')
  }
}

async function handleRun(row: any) {
  await api.run(row.id)
  ElMessage.success('已触发执行, 结果请查看执行日志')
  setTimeout(() => openLogs(row), 800)
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`确认删除任务「${row.name}」吗? 执行日志将保留。`, '提示', { type: 'warning' })
  await api.delete(row.id)
  ElMessage.success('已删除')
  loadList()
}

// ---- 执行日志抽屉 ----
const logVisible = ref(false)
const logLoading = ref(false)
const logList = ref<any[]>([])
const logTotal = ref(0)
const logJob = ref<any>(null)
const logQuery = reactive({ jobId: 0, status: '', page: 1, pageSize: 10 })

function openLogs(row: any) {
  logJob.value = row
  logQuery.jobId = row.id
  logQuery.status = ''
  logQuery.page = 1
  logVisible.value = true
  loadLogs()
}

async function loadLogs() {
  logLoading.value = true
  try {
    const res = await api.logs({ ...logQuery })
    logList.value = res.list || []
    logTotal.value = res.total || 0
  }
  finally {
    logLoading.value = false
  }
}

const durationText = (ms: number): string => (ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${ms}ms`)

onMounted(async () => {
  loadList()
  try {
    const res = await api.handlers()
    handlers.value = res.handlers || []
  }
  catch {
    handlers.value = []
  }
})
</script>

<template>
  <div class="page">
    <el-card>
      <el-form inline @submit.prevent>
        <el-form-item label="搜索">
          <el-input
            v-model="query.keyword"
            placeholder="任务名/处理器"
            clearable
            style="width:200px"
            @keyup.enter="() => { query.page = 1; loadList() }"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="() => { query.page = 1; loadList() }">查询</el-button>
          <el-button :icon="Refresh" @click="resetQuery">重置</el-button>
        </el-form-item>
      </el-form>

      <div class="table-actions">
        <el-button v-permission="'system:job:create'" type="success" :icon="Plus" @click="openCreate">新增</el-button>
      </div>

      <el-table v-loading="loading" :data="list" border stripe style="margin-top:8px">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="任务名称" min-width="140" />
        <el-table-column prop="handler" label="处理器" min-width="150" />
        <el-table-column label="cron 表达式" width="160">
          <template #default="{ row }">
            <code class="cron-code">{{ row.cronExpr }}</code>
          </template>
        </el-table-column>
        <el-table-column label="参数" min-width="120">
          <template #default="{ row }">
            <code v-if="row.params" class="cron-code">{{ row.params }}</code>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-switch
              v-model="row.status"
              :active-value="1"
              :inactive-value="0"
              :disabled="!canStatus"
              active-text="启动"
              inactive-text="暂停"
              inline-prompt
              @change="handleStatusChange(row)"
            />
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="120" show-overflow-tooltip />
        <el-table-column prop="updatedAt" label="更新时间" width="170" />
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button v-permission="'system:job:run'" link type="success" :icon="VideoPlay" @click="handleRun(row)">执行</el-button>
            <el-button v-permission="'system:job:log'" link type="primary" :icon="Tickets" @click="openLogs(row)">日志</el-button>
            <el-button v-permission="'system:job:update'" link type="warning" :icon="Edit" @click="openEdit(row)">编辑</el-button>
            <el-button v-permission="'system:job:delete'" link type="danger" :icon="Delete" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="query.page"
        v-model:page-size="query.pageSize"
        :total="total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next, jumper"
        style="margin-top:16px;justify-content:flex-end"
        @current-change="loadList"
        @size-change="loadList"
      />
    </el-card>

    <!-- 新增/编辑弹窗 -->
    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑任务' : '新增任务'" width="560px">
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="100px">
        <el-form-item label="任务名称" prop="name">
          <el-input v-model="form.name" placeholder="如: 清理登录日志" />
        </el-form-item>
        <el-form-item label="处理器" prop="handler">
          <el-select v-model="form.handler" placeholder="选择已注册的处理器" style="width:100%">
            <el-option v-for="h in handlers" :key="h" :value="h" :label="h" />
          </el-select>
        </el-form-item>
        <el-form-item label="cron 表达式" prop="cronExpr">
          <el-input v-model="form.cronExpr" placeholder="6位: 秒 分 时 日 月 周" />
          <div class="preset-row">
            <el-tag
              v-for="p in cronPresets"
              :key="p.value"
              class="preset-tag"
              effect="plain"
              @click="form.cronExpr = p.value"
            >{{ p.label }}</el-tag>
          </div>
        </el-form-item>
        <el-form-item label="任务参数">
          <el-input v-model="form.params" type="textarea" :rows="2" placeholder='JSON 参数, 如 {"days": 90} (可空)' />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="2" placeholder="任务用途说明 (可空)" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- 执行日志抽屉 -->
    <el-drawer v-model="logVisible" size="720px">
      <template #header>
        <span>执行日志{{ logJob ? ` - ${logJob.name}` : '' }}</span>
      </template>
      <el-form inline @submit.prevent>
        <el-form-item label="结果">
          <el-select v-model="logQuery.status" placeholder="全部" clearable style="width:100px" @change="() => { logQuery.page = 1; loadLogs() }">
            <el-option value="success" label="成功" />
            <el-option value="fail" label="失败" />
          </el-select>
        </el-form-item>
      </el-form>
      <el-table v-loading="logLoading" :data="logList" border stripe size="small">
        <el-table-column label="结果" width="70">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="输出/原因" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <span :class="{ 'fail-msg': row.status === 0 }">{{ row.output || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="耗时" width="90">
          <template #default="{ row }">{{ durationText(row.durationMs) }}</template>
        </el-table-column>
        <el-table-column prop="createdAt" label="执行时间" width="170" />
      </el-table>
      <el-pagination
        v-model:current-page="logQuery.page"
        v-model:page-size="logQuery.pageSize"
        :total="logTotal"
        :page-sizes="[10, 20, 50]"
        layout="total, prev, pager, next"
        style="margin-top:12px;justify-content:flex-end"
        @current-change="loadLogs"
        @size-change="loadLogs"
      />
    </el-drawer>
  </div>
</template>

<style scoped>
.page {
  padding: 0;
}

.table-actions {
  margin-top: 4px;
}

.cron-code {
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  font-size: 12px;
  background: var(--el-fill-color-light);
  padding: 2px 6px;
  border-radius: 4px;
}

.preset-row {
  margin-top: 6px;
}

.preset-tag {
  cursor: pointer;
  margin-right: 6px;
}

.fail-msg {
  color: var(--el-color-danger);
}
</style>
