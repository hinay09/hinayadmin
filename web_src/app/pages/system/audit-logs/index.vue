<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Search, Refresh, View, Document as DocumentIcon } from '@element-plus/icons-vue'
import { useAuditLogApi } from '~/composables/useApi'

definePageMeta({ title: '操作日志' })

const api = useAuditLogApi()
const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const query = reactive({
  keyword: '',
  username: '',
  action: '',
  result: '',
  startAt: '',
  endAt: '',
  page: 1,
  pageSize: 10,
})

const actionMap: Record<string, string> = {
  create: '新增',
  update: '修改',
  delete: '删除',
  upload: '上传',
  login: '登录',
  logout: '登出',
  refresh: '续签',
  create_roles: '设置角色',
  update_password: '改密',
  update_status: '改状态',
}

const actionTagType = (action: string): string => {
  if (action === 'delete') return 'danger'
  if (action === 'create' || action === 'login') return 'success'
  return 'warning'
}

const methodTagType = (method: string): string => {
  switch (method) {
    case 'GET': return 'success'
    case 'POST': return 'primary'
    case 'PUT': return 'warning'
    case 'DELETE': return 'danger'
    default: return 'info'
  }
}

const isSuccess = (row: any): boolean => row.code === 0 && row.statusCode < 400

// ---- 详情抽屉 ----
const drawerVisible = ref(false)
const current = ref<any>(null)

function openDetail(row: any) {
  current.value = row
  drawerVisible.value = true
}

/** 请求体详情格式化为可读 JSON */
function prettyDetail(detail: string): string {
  if (!detail) return '（无请求体）'
  try {
    return JSON.stringify(JSON.parse(detail), null, 2)
  }
  catch {
    return detail
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
  query.username = ''
  query.action = ''
  query.result = ''
  query.startAt = ''
  query.endAt = ''
  query.page = 1
  loadList()
}

onMounted(loadList)
</script>

<template>
  <div class="page">
    <el-card>
      <el-form inline @submit.prevent>
        <el-form-item label="搜索">
          <el-input
            v-model="query.keyword"
            placeholder="用户名/路径/资源/消息/RequestId"
            clearable
            style="width:220px"
            @keyup.enter="() => { query.page = 1; loadList() }"
          />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input
            v-model="query.username"
            placeholder="精确匹配"
            clearable
            style="width:120px"
            @keyup.enter="() => { query.page = 1; loadList() }"
          />
        </el-form-item>
        <el-form-item label="操作类型">
          <el-select v-model="query.action" placeholder="全部" clearable style="width:110px">
            <el-option v-for="(label, key) in actionMap" :key="key" :value="key" :label="label" />
          </el-select>
        </el-form-item>
        <el-form-item label="结果">
          <el-select v-model="query.result" placeholder="全部" clearable style="width:100px">
            <el-option value="success" label="成功" />
            <el-option value="fail" label="失败" />
          </el-select>
        </el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker
            v-model="query.startAt"
            type="datetime"
            placeholder="开始时间"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width:180px"
          />
          <span style="margin:0 8px">~</span>
          <el-date-picker
            v-model="query.endAt"
            type="datetime"
            placeholder="结束时间"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width:180px"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="() => { query.page = 1; loadList() }">查询</el-button>
          <el-button :icon="Refresh" @click="resetQuery">重置</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="list" border stripe style="margin-top:8px">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="用户名" width="110">
          <template #default="{ row }">
            {{ row.username || '匿名' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="actionTagType(row.action)">
              {{ actionMap[row.action] || row.action }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="资源" width="130">
          <template #default="{ row }">
            {{ row.resource }}<span v-if="row.resourceId"> #{{ row.resourceId }}</span>
          </template>
        </el-table-column>
        <el-table-column label="请求" min-width="220">
          <template #default="{ row }">
            <el-tag size="small" :type="methodTagType(row.method)" class="method-tag">{{ row.method || '-' }}</el-tag>
            <span class="req-path">{{ row.path }}</span>
          </template>
        </el-table-column>
        <el-table-column label="结果" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="isSuccess(row) ? 'success' : 'danger'">
              {{ isSuccess(row) ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="耗时" width="80">
          <template #default="{ row }">
            <span :class="{ 'slow-warn': row.durationMs > 500 }">{{ row.durationMs }}ms</span>
          </template>
        </el-table-column>
        <el-table-column prop="ip" label="IP" width="130" />
        <el-table-column prop="createdAt" label="操作时间" width="170" />
        <el-table-column label="操作" width="80" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" :icon="View" @click="openDetail(row)">详情</el-button>
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

    <!-- 详情抽屉 -->
    <el-drawer v-model="drawerVisible" :title="`日志详情 #${current?.id ?? ''}`" size="620px">
      <template #header>
        <div class="drawer-title">
          <el-icon><DocumentIcon /></el-icon>
          <span>日志详情 #{{ current?.id ?? '' }}</span>
          <el-tag v-if="current" size="small" :type="isSuccess(current) ? 'success' : 'danger'" style="margin-left:8px">
            {{ isSuccess(current) ? '成功' : '失败' }}
          </el-tag>
        </div>
      </template>

      <el-descriptions v-if="current" :column="2" border size="small">
        <el-descriptions-item label="操作人" :span="2">
          {{ current.username || '匿名' }}<span v-if="current.userId">（ID: {{ current.userId }}）</span>
        </el-descriptions-item>
        <el-descriptions-item label="操作">
          <el-tag size="small" :type="actionTagType(current.action)">{{ actionMap[current.action] || current.action }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="资源">
          {{ current.resource }}<span v-if="current.resourceId"> #{{ current.resourceId }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="HTTP 状态">
          <el-tag size="small" :type="current.statusCode >= 400 ? 'danger' : 'success'">{{ current.statusCode || '-' }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="业务码">{{ current.code }}</el-descriptions-item>
        <el-descriptions-item label="耗时">
          <span :class="{ 'slow-warn': current.durationMs > 500 }">{{ current.durationMs }}ms</span>
        </el-descriptions-item>
        <el-descriptions-item label="操作时间">{{ current.createdAt || '-' }}</el-descriptions-item>
        <el-descriptions-item label="请求路径" :span="2">
          <el-tag size="small" :type="methodTagType(current.method)" class="method-tag">{{ current.method || '-' }}</el-tag>
          <span class="req-path">{{ current.path || '-' }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="失败原因 / 消息" :span="2">
          <span :class="{ 'fail-msg': !isSuccess(current) }">{{ current.message || '-' }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="请求 ID (RequestId)" :span="2">
          <span class="req-path">{{ current.requestId || '-' }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="IP 地址" :span="2">{{ current.ip || '-' }}</el-descriptions-item>
        <el-descriptions-item label="User-Agent" :span="2">
          <span class="ua-text">{{ current.userAgent || '-' }}</span>
        </el-descriptions-item>
      </el-descriptions>

      <div v-if="current" class="detail-section">
        <div class="detail-section-title">请求体（已脱敏）</div>
        <pre class="detail-json">{{ prettyDetail(current.detail) }}</pre>
      </div>

      <template #footer>
        <el-button @click="drawerVisible = false">关闭</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped>
.method-tag {
  margin-right: 6px;
  font-family: monospace;
}
.req-path {
  font-family: monospace;
  font-size: 12px;
  word-break: break-all;
}
.slow-warn {
  color: #e6a23c;
  font-weight: 600;
}
.fail-msg {
  color: #f56c6c;
}
.ua-text {
  font-size: 12px;
  color: #606266;
  word-break: break-all;
}
.drawer-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
}
.detail-section {
  margin-top: 16px;
}
.detail-section-title {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 8px;
}
.detail-json {
  margin: 0;
  padding: 12px;
  background: #f5f7fa;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
  max-height: 360px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
