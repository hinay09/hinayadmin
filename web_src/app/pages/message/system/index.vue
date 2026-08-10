<script setup lang="ts">
/**
 * 消息中心 - 系统通知。
 * 普通用户: 查看收件箱中的系统通知, 支持标记已读 / 个人删除。
 * 管理员: 拥有发布(范围: 全员/角色/用户) + 物理删除按钮。
 */
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { Search, Plus, View, Check, Delete, Bell } from '@element-plus/icons-vue'
import { useMessageApi, useRoleApi, useUserApi, type MessageItem } from '~/composables/useApi'
import { useUserStore } from '~/stores/user'

definePageMeta({ title: '系统通知' })

const api = useMessageApi()
const roleApi = useRoleApi()
const userApi = useUserApi()
const userStore = useUserStore()
const route = useRoute()

const isAdmin = computed(() => userStore.isAdmin)

const loading = ref(false)
const list = ref<MessageItem[]>([])
const total = ref(0)
const query = reactive({
  keyword: '',
  isRead: undefined as number | undefined,
  page: 1,
  pageSize: 10,
})

// 角色 / 用户下拉数据
const roleOptions = ref<Array<{ id: number; name: string }>>([])
const userOptions = ref<Array<{ id: number; nickname: string; username: string }>>([])

async function loadRoles() {
  if (roleOptions.value.length) return
  const res: any = await roleApi.all()
  roleOptions.value = res?.list || []
}

async function loadUsers() {
  if (userOptions.value.length) return
  const res: any = await userApi.list({ page: 1, pageSize: 200 })
  userOptions.value = res?.list || []
}

const drawerVisible = ref(false)
const formRef = ref<FormInstance>()
const form = reactive({
  title: '',
  content: '',
  level: 1,
  status: 1,
  targetScope: 1,
  targetIds: [] as number[],
})
const rules = {
  title: [{ required: true, message: '请输入标题', trigger: 'blur' }],
  content: [{ required: true, message: '请输入内容', trigger: 'blur' }],
  targetScope: [{ required: true, message: '请选择发布范围', trigger: 'change' }],
}

// 详情抽屉
const detailVisible = ref(false)
const detail = ref<MessageItem | null>(null)

const levelText = (l: number) => (l === 3 ? '紧急' : l === 2 ? '重要' : '普通')
const levelTag = (l: number) => (l === 3 ? 'danger' : l === 2 ? 'warning' : 'info')
const scopeText = (m: MessageItem) => {
  if (m.targetScope === 1) return '全员'
  if (m.targetScope === 2) {
    const names = m.targetRoleNames?.length ? m.targetRoleNames.join(',') : `${m.targetRoleIds?.length || 0}个角色`
    return `角色: ${names}`
  }
  if (m.targetScope === 3) return `指定用户(${m.targetUserIds?.length || 0})`
  return '-'
}

async function loadList() {
  loading.value = true
  try {
    const params: any = { type: 1, page: query.page, pageSize: query.pageSize }
    if (query.keyword) params.keyword = query.keyword
    if (query.isRead !== undefined) params.isRead = query.isRead
    const res: any = await api.inbox(params)
    list.value = res?.list || []
    total.value = res?.total || 0
  }
  finally {
    loading.value = false
  }
}

function resetForm() {
  Object.assign(form, { title: '', content: '', level: 1, status: 1, targetScope: 1, targetIds: [] })
}

async function openCreate() {
  resetForm()
  await Promise.all([loadRoles(), loadUsers()])
  drawerVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  if ((form.targetScope === 2 || form.targetScope === 3) && form.targetIds.length === 0) {
    ElMessage.warning('请选择目标对象')
    return
  }
  await api.createSystem({
    title: form.title,
    content: form.content,
    level: form.level,
    status: form.status,
    targetScope: form.targetScope,
    targetIds: form.targetScope === 1 ? [] : form.targetIds,
  })
  ElMessage.success('发布成功')
  drawerVisible.value = false
  query.page = 1
  loadList()
}

async function openDetail(row: MessageItem) {
  const d = await api.inboxRead(row.id)
  detail.value = d
  detailVisible.value = true
  // 同步列表已读状态
  const target = list.value.find((x) => x.id === row.id)
  if (target) target.isRead = true
}

async function handleMarkRead(row: MessageItem) {
  if (row.isRead) return
  await api.markRead(row.id)
  row.isRead = true
  ElMessage.success('已标记为已读')
}

async function handleMarkReadAll() {
  const res: any = await api.markReadAll()
  ElMessage.success(`已标记 ${res?.affected || 0} 条为已读`)
  loadList()
}

async function handleAdminDelete(row: MessageItem) {
  await ElMessageBox.confirm(`确认彻底删除消息 "${row.title}"?`, '提示', { type: 'warning' })
  await api.remove(row.id)
  ElMessage.success('删除成功')
  loadList()
}

async function handleInboxDelete(row: MessageItem) {
  await ElMessageBox.confirm(`确认从收件箱中移除 "${row.title}"?`, '提示', { type: 'warning' })
  await api.inboxRemove(row.id)
  ElMessage.success('已删除')
  loadList()
}

onMounted(async () => {
  await loadList()
  // 从顶部铃铛点击跳转过来: ?id=xxx 自动打开详情
  const qid = Number(route.query.id)
  if (qid) {
    try {
      const d = await api.inboxRead(qid)
      detail.value = d
      detailVisible.value = true
      const target = list.value.find((x) => x.id === qid)
      if (target) target.isRead = true
    }
    catch {}
  }
})
</script>

<template>
  <div class="page">
    <el-card>
      <el-form inline @submit.prevent>
        <el-form-item label="关键词">
          <el-input v-model="query.keyword" placeholder="标题模糊" clearable />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="query.isRead" placeholder="全部" clearable style="width:120px">
            <el-option :value="0" label="未读" />
            <el-option :value="1" label="已读" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="() => { query.page = 1; loadList() }">查询</el-button>
          <el-button :icon="Check" @click="handleMarkReadAll">全部已读</el-button>
          <el-button v-permission="'message:system:send'" type="success" :icon="Plus" @click="openCreate">发布</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="list" border stripe style="margin-top:8px">
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="!row.isRead" type="danger" size="small" effect="dark">未读</el-tag>
            <el-tag v-else type="info" size="small">已读</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="标题" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <span :class="{ 'unread-title': !row.isRead }">{{ row.title }}</span>
          </template>
        </el-table-column>
        <el-table-column label="级别" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="levelTag(row.level)" size="small">{{ levelText(row.level) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="发送人" width="140" prop="senderName" show-overflow-tooltip />
        <el-table-column label="发布范围" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">{{ scopeText(row) }}</template>
        </el-table-column>
        <el-table-column prop="createdAt" label="发送时间" width="180" />
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" :icon="View" @click="openDetail(row)">查看</el-button>
            <el-button v-if="!row.isRead" link type="success" :icon="Check" @click="handleMarkRead(row)">已读</el-button>
            <el-button v-if="isAdmin" v-permission="'message:system:delete'" link type="danger" :icon="Delete" @click="handleAdminDelete(row)">彻底删除</el-button>
            <el-button v-else link type="danger" :icon="Delete" @click="handleInboxDelete(row)">删除</el-button>
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

    <!-- 发布系统通知抽屉 -->
    <el-drawer v-model="drawerVisible" title="发布系统通知" size="560px">
      <template #header>
        <div style="display:flex;align-items:center;gap:6px;font-weight:600">
          <el-icon><Bell /></el-icon>
          <span>发布系统通知</span>
        </div>
      </template>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="标题" prop="title">
          <el-input v-model="form.title" placeholder="请输入通知标题" maxlength="128" show-word-limit />
        </el-form-item>
        <el-form-item label="级别">
          <el-radio-group v-model="form.level">
            <el-radio :value="1">普通</el-radio>
            <el-radio :value="2">重要</el-radio>
            <el-radio :value="3">紧急</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio :value="0">草稿</el-radio>
            <el-radio :value="1">已发布</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="发布范围" prop="targetScope">
          <el-radio-group v-model="form.targetScope" @change="form.targetIds = []">
            <el-radio :value="1">全员</el-radio>
            <el-radio :value="2">指定角色</el-radio>
            <el-radio :value="3">指定用户</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="form.targetScope === 2" label="目标角色">
          <el-select v-model="form.targetIds" multiple filterable placeholder="选择角色" style="width:100%">
            <el-option v-for="r in roleOptions" :key="r.id" :value="r.id" :label="r.name" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.targetScope === 3" label="目标用户">
          <el-select v-model="form.targetIds" multiple filterable placeholder="选择用户" style="width:100%">
            <el-option v-for="u in userOptions" :key="u.id" :value="u.id" :label="`${u.nickname || u.username} (${u.username})`" />
          </el-select>
        </el-form-item>
        <el-form-item label="内容" prop="content">
          <el-input v-model="form.content" type="textarea" :rows="10" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="drawerVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">确定</el-button>
      </template>
    </el-drawer>

    <!-- 详情抽屉 -->
    <el-drawer v-model="detailVisible" title="消息详情" size="520px" destroy-on-close>
      <template #header>
        <div style="display:flex;align-items:center;gap:6px;font-weight:600">
          <el-icon><Bell /></el-icon>
          <span>{{ detail?.title || '消息详情' }}</span>
        </div>
      </template>
      <div v-if="detail" class="msg-detail">
        <el-descriptions :column="1" border size="small">
          <el-descriptions-item label="级别">
            <el-tag :type="levelTag(detail.level)" size="small">{{ levelText(detail.level) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="发送人">{{ detail.senderName }}</el-descriptions-item>
          <el-descriptions-item label="发布范围">{{ scopeText(detail) }}</el-descriptions-item>
          <el-descriptions-item label="发送时间">{{ detail.createdAt }}</el-descriptions-item>
        </el-descriptions>
        <div class="content">
          <div class="content-title">通知内容</div>
          <div class="content-body">{{ detail.content }}</div>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<style scoped>
.unread-title { font-weight: 600; color: #303133; }
.msg-detail { padding: 0 4px; }
.content { margin-top: 16px; }
.content-title { font-weight: 600; margin-bottom: 8px; color: #606266; }
.content-body { white-space: pre-wrap; line-height: 1.7; padding: 12px; background: #f5f7fa; border-radius: 4px; color: #303133; }
</style>
