<script setup lang="ts">
/**
 * 流程定义管理: 列表 + 设计器 (表单设计/流程设计) + 发布版本管理 + 发起流程入口。
 */
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Plus, Edit, Delete, Promotion, VideoPause, Position, FullScreen } from '@element-plus/icons-vue'
import { useFlowApi, type FlowDefinitionItem, type FlowFormField, type FlowNode, type FlowRoleOption, type FlowUserOption, type FlowPostOption } from '~/composables/useApi/flow'
import FormDesigner from '~/components/flow/FormDesigner.vue'
import FlowDesigner from '~/components/flow/FlowDesigner.vue'
import FlowSubmitDialog from '~/components/flow/FlowSubmitDialog.vue'

definePageMeta({ title: '流程定义' })
defineOptions({ name: 'flow-definitions' })

const api = useFlowApi()

// ============================================================
// 列表
// ============================================================
const loading = ref(false)
const list = ref<FlowDefinitionItem[]>([])
const total = ref(0)
const query = reactive({ keyword: '', status: undefined as number | undefined, version: undefined as number | null | undefined, page: 1, pageSize: 10 })

async function load() {
  loading.value = true
  try {
    // el-input-number 清空时置 null, ufo 序列化只跳过 undefined, 需归一化避免发出 version=
    const res = await api.defList({ ...query, version: query.version ?? undefined })
    list.value = res.list || []
    total.value = res.total || 0
  } finally { loading.value = false }
}

function statusInfo(row: FlowDefinitionItem) {
  if (row.version === 0) return { text: '草稿', tag: 'info' as const }
  if (row.status === 2) return { text: `v${row.version} 已停用`, tag: 'warning' as const }
  return { text: `v${row.version} 已发布`, tag: 'success' as const }
}

// ============================================================
// 设计器 (新增/编辑/查看)
// ============================================================
const users = ref<FlowUserOption[]>([])
const roles = ref<FlowRoleOption[]>([])
const posts = ref<FlowPostOption[]>([])

const editVisible = ref(false)
const editMode = ref<'create' | 'edit' | 'view'>('create')
const editId = ref(0)
const submitting = ref(false)
const editForm = reactive({
  name: '', flowKey: '', remark: '',
  fields: [] as FlowFormField[],
  root: null as FlowNode | null,
})

const editTitle = computed(() => ({ create: '新增流程', edit: '编辑流程', view: '查看流程' })[editMode.value])
const editorFullscreen = ref(false)
const editIdPublished = ref(false)

function parseSafe<T>(s: string, fallback: T): T {
  try { return s ? JSON.parse(s) as T : fallback } catch { return fallback }
}

function openCreate() {
  editMode.value = 'create'
  editId.value = 0
  editIdPublished.value = false
  editorFullscreen.value = false
  editForm.name = ''
  editForm.flowKey = ''
  editForm.remark = ''
  editForm.fields = [{ key: 'reason', label: '申请事由', type: 'textarea', required: true }]
  editForm.root = { id: 'start', type: 'start', name: '发起人', child: { id: 'n1', type: 'approver', name: '审批人', approverType: 'selfSelect', approverIds: [], signType: 'any' } }
  editVisible.value = true
}

function openEdit(row: FlowDefinitionItem, mode: 'edit' | 'view') {
  editMode.value = mode
  editId.value = row.id
  editIdPublished.value = row.status === 1 && row.version > 0
  editorFullscreen.value = false
  editForm.name = row.name
  editForm.flowKey = row.flowKey
  editForm.remark = row.remark
  editForm.fields = parseSafe<FlowFormField[]>(row.formConf, [])
  editForm.root = parseSafe<FlowNode | null>(row.flowConf, null)
  if (!editForm.root) editForm.root = { id: 'start', type: 'start', name: '发起人' }
  editVisible.value = true
}

async function handleSave() {
  if (!editForm.name) { ElMessage.warning('请输入流程名称'); return }
  submitting.value = true
  try {
    const payload = {
      name: editForm.name,
      flowKey: editForm.flowKey,
      remark: editForm.remark,
      formConf: JSON.stringify(editForm.fields),
      flowConf: JSON.stringify(editForm.root),
    }
    if (editMode.value === 'create') {
      const res = await api.defCreate(payload)
      editId.value = res.id
      ElMessage.success('已保存草稿, 发布后可发起')
    } else {
      await api.defUpdate(editId.value, payload)
      ElMessage.success(editIdPublished.value ? '已保存, 对新发起即时生效' : '已保存')
    }
    editMode.value = 'edit'
    editVisible.value = false
    load()
  } catch {} finally { submitting.value = false }
}

async function handlePublish(row: FlowDefinitionItem) {
  await ElMessageBox.confirm(`发布「${row.name}」? 版本号将 +1 并立即可发起; 之后仍可继续编辑, 修改对新发起即时生效。`, '发布确认', { type: 'info' })
  try {
    const res = await api.defPublish(row.id)
    ElMessage.success(`已发布 v${res.version}`)
    load()
  } catch {}
}

async function handleDisable(row: FlowDefinitionItem) {
  await ElMessageBox.confirm(`停用「${row.name}」v${row.version}? 停用后不可再发起, 在途流程不受影响。`, '停用确认', { type: 'warning' })
  try {
    await api.defDisable(row.id)
    ElMessage.success('已停用')
    load()
  } catch {}
}

async function handleDelete(row: FlowDefinitionItem) {
  if (row.version > 0) { ElMessage.warning('已发布过的流程不允许删除 (仅草稿可删除)'); return }
  await ElMessageBox.confirm(`删除草稿「${row.name}」? 删除后不可恢复。`, '删除确认', { type: 'warning' })
  try {
    await api.defRemove(row.id)
    ElMessage.success('已删除')
    load()
  } catch {}
}

// ============================================================
// 发起流程 (公共弹窗 FlowSubmitDialog: 选流程/表单/自选审批人/必填校验统一在这里)
// ============================================================
const startVisible = ref(false)

function onSubmitted(id: number) {
  navigateTo({ path: '/flow/detail', query: { id: String(id) } })
}

onMounted(async () => {
  load()
  try {
    const opt = await api.options()
    users.value = opt.users || []
    roles.value = opt.roles || []
    posts.value = opt.posts || []
  } catch {}
})
</script>

<template>
  <div class="page">
    <el-card>
      <el-form inline @submit.prevent>
        <el-form-item label="搜索">
          <el-input v-model="query.keyword" placeholder="名称/标识" clearable
            @keyup.enter="() => { query.page = 1; load() }" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="query.status" clearable placeholder="全部" style="width:130px">
            <el-option label="草稿" :value="0" />
            <el-option label="已发布" :value="1" />
            <el-option label="已停用" :value="2" />
          </el-select>
        </el-form-item>
        <el-form-item label="版本">
          <el-input-number v-model="query.version" :min="0" :max="999" step-strictly controls-position="right"
            :value-on-clear="null" placeholder="0" style="width:120px"
            @keyup.enter="() => { query.page = 1; load() }" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="() => { query.page = 1; load() }">查询</el-button>
          <el-button v-permission="'flow:definition:create'" type="success" :icon="Plus" @click="openCreate">新增流程</el-button>
          <el-button v-permission="'flow:instance:start'" type="warning" plain :icon="Position" @click="startVisible = true">发起流程</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="list" border stripe style="margin-top:8px">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="流程名称" min-width="150" />
        <el-table-column prop="flowKey" label="流程标识" width="150" show-overflow-tooltip />
        <el-table-column label="版本" width="120">
          <template #default="{ row }">
            <el-tag :type="statusInfo(row).tag">{{ statusInfo(row).text }}</el-tag>
          </template>

        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="140" show-overflow-tooltip />
        <el-table-column prop="updatedAt" label="更新时间" width="170" />
        <el-table-column label="操作" width="330" fixed="right">
          <template #default="{ row }">
            <el-button v-permission="'flow:definition:update'" link type="primary" :icon="Edit"
              @click="openEdit(row, 'edit')">编辑</el-button>
            <el-button v-if="row.version === 0 || row.status === 2" v-permission="'flow:definition:publish'" link type="success" :icon="Promotion"
              @click="handlePublish(row)">发布</el-button>
            <el-button v-if="row.status === 1" v-permission="'flow:definition:publish'" link type="warning" :icon="VideoPause"
              @click="handleDisable(row)">停用</el-button>
            <el-tooltip v-if="row.version > 0" content="已发布过的流程不允许删除" placement="top">
              <el-button link type="danger" :icon="Delete" disabled>删除</el-button>
            </el-tooltip>
            <el-button v-else v-permission="'flow:definition:delete'" link type="danger" :icon="Delete"
              @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination v-model:current-page="query.page" v-model:page-size="query.pageSize"
        :total="total" :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next, jumper"
        style="margin-top:16px;justify-content:flex-end"
        @current-change="load" @size-change="load" />
    </el-card>

    <!-- 流程设计器 -->
    <el-dialog v-model="editVisible" :fullscreen="editorFullscreen" :width="editorFullscreen ? '100%' : '94%'"
      :top="editorFullscreen ? '0' : '3vh'" destroy-on-close class="flow-editor-dialog">
      <template #header>
        <div class="fe-header">
          <span style="font-weight:600">{{ editTitle }}</span>
          <el-tooltip :content="editorFullscreen ? '还原窗口' : '全屏编辑'">
            <el-button link :icon="FullScreen" @click="editorFullscreen = !editorFullscreen" />
          </el-tooltip>
        </div>
      </template>
      <el-alert v-if="editMode === 'edit' && editIdPublished" type="warning" :closable="false" show-icon style="margin-bottom:10px"
        title="该流程已发布: 保存后对新发起的流程即时生效; 在途流程使用发起时的快照, 不受影响。" />
      <el-tabs>
        <el-tab-pane label="基本信息">
          <el-form label-width="90px" :disabled="editMode === 'view'">
            <el-form-item label="流程名称" required>
              <el-input v-model="editForm.name" maxlength="60" style="max-width:360px" />
            </el-form-item>
            <el-form-item label="流程标识">
              <el-input v-model="editForm.flowKey" maxlength="60" placeholder="如 leave (同标识共用版本组, 可空自动生成)" style="max-width:360px" />
            </el-form-item>
            <el-form-item label="备注">
              <el-input v-model="editForm.remark" type="textarea" :rows="2" style="max-width:480px" />
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="表单设计">
          <FormDesigner v-model="editForm.fields" />
        </el-tab-pane>
        <el-tab-pane label="流程设计">
          <FlowDesigner :model-value="editForm.root!" :users="users" :roles="roles" :posts="posts" :fields="editForm.fields" />
        </el-tab-pane>
      </el-tabs>
      <template #footer>
        <el-button @click="editVisible = false">关闭</el-button>
        <el-button v-if="editMode !== 'view'" type="primary" :loading="submitting" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>

    <!-- 发起流程 (公共弹窗) -->
    <FlowSubmitDialog v-model="startVisible" @submitted="onSubmitted" />
  </div>
</template>

<style scoped>
.fe-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-right: 24px;
}
</style>
