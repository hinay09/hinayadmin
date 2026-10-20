<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Search, Refresh, View, Document as DocumentIcon } from '@element-plus/icons-vue'
import { useAccessLogApi } from '~/composables/useApi'

definePageMeta({ title: '接口日志' })
defineOptions({ name: 'system-access-logs' })

const api = useAccessLogApi()

/* ---- 多租户: 平台超管混排视角展示所属租户列 (§8.6) ---- */
const userStore = useUserStore()
const configStore = useConfigStore()
const showTenantCol = computed(() => userStore.isPlatformAdmin && configStore.multiTenant)

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const query = reactive({
  keyword: '',
  username: '',
  method: '',
  slow: false,
  result: '',
  minDuration: undefined as number | undefined,
  startAt: '',
  endAt: '',
  page: 1,
  pageSize: 10,
})

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
const statusTagType = (code: number): string => (code >= 400 ? 'danger' : code >= 300 ? 'warning' : 'success')

/** 截断过长的查询串用于表格内展示 */
function shortQuery(q: string): string {
  if (!q) return ''
  return q.length > 60 ? `${q.slice(0, 60)}…` : q
}

async function loadList() {
  loading.value = true
  try {
    const res = await api.list({
      ...query,
      slow: query.slow ? 1 : 0,
      minDuration: query.minDuration || 0,
    })
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
  query.method = ''
  query.slow = false
  query.result = ''
  query.minDuration = undefined
  query.startAt = ''
  query.endAt = ''
  query.page = 1
  loadList()
}

onMounted(loadList)

// ---- 详情抽屉 ----
const drawerVisible = ref(false)
const current = ref<any>(null)

function openDetail(row: any) {
  current.value = row
  drawerVisible.value = true
}
</script>

<template>
  <div class="page">
    <el-card>
      <el-form inline @submit.prevent>
        <el-form-item label="搜索">
          <el-input
            v-model="query.keyword"
            placeholder="路径/用户名/RequestId"
            clearable
            style="width:200px"
            @keyup.enter="() => { query.page = 1; loadList() }"
          />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input
            v-model="query.username"
            placeholder="精确匹配"
            clearable
            style="width:110px"
            @keyup.enter="() => { query.page = 1; loadList() }"
          />
        </el-form-item>
        <el-form-item label="方法">
          <el-select v-model="query.method" placeholder="全部" clearable style="width:100px">
            <el-option value="GET" label="GET" />
            <el-option value="POST" label="POST" />
            <el-option value="PUT" label="PUT" />
            <el-option value="DELETE" label="DELETE" />
            <el-option value="PATCH" label="PATCH" />
          </el-select>
        </el-form-item>
        <el-form-item label="结果">
          <el-select v-model="query.result" placeholder="全部" clearable style="width:90px">
            <el-option value="success" label="成功" />
            <el-option value="fail" label="失败" />
          </el-select>
        </el-form-item>
        <el-form-item label="仅慢接口">
          <el-switch v-model="query.slow" @change="() => { query.page = 1; loadList() }" />
        </el-form-item>
        <el-form-item label="耗时≥">
          <el-input-number
            v-model="query.minDuration"
            :min="0"
            :step="100"
            :controls="false"
            placeholder="毫秒"
            style="width:100px"
            @keyup.enter="() => { query.page = 1; loadList() }"
          />
          <span style="margin-left:4px">ms</span>
        </el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker
            v-model="query.startAt"
            type="datetime"
            placeholder="开始时间"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width:175px"
          />
          <span style="margin:0 8px">~</span>
          <el-date-picker
            v-model="query.endAt"
            type="datetime"
            placeholder="结束时间"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width:175px"
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
        <el-table-column v-if="showTenantCol" prop="tenantName" label="所属租户" min-width="110" />
        <el-table-column label="请求" min-width="280">
          <template #default="{ row }">
            <div class="req-line">
              <el-tag size="small" :type="methodTagType(row.method)" class="method-tag">{{ row.method || '-' }}</el-tag>
              <span class="req-path">{{ row.path }}</span>
            </div>
            <div v-if="row.query" class="req-query" :title="row.query">?{{ shortQuery(row.query) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="HTTP" width="80">
          <template #default="{ row }">
            <el-tag size="small" :type="statusTagType(row.statusCode)">{{ row.statusCode || '-' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="业务码" width="80">
          <template #default="{ row }">
            <span :class="{ 'fail-msg': row.code !== 0 }">{{ row.code }}</span>
          </template>
        </el-table-column>
        <el-table-column label="耗时" width="120">
          <template #default="{ row }">
            <el-tag v-if="row.slow === 1" size="small" type="danger" effect="dark" class="slow-tag">慢</el-tag>
            <span :class="{ 'slow-warn': row.slow === 1 || row.durationMs > 500 }">{{ row.durationMs }}ms</span>
          </template>
        </el-table-column>
        <el-table-column prop="ip" label="IP" width="130" />
        <el-table-column prop="createdAt" label="访问时间" width="170" />
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
    <el-drawer v-model="drawerVisible" size="600px">
      <template #header>
        <div class="drawer-title">
          <el-icon><DocumentIcon /></el-icon>
          <span>访问详情 #{{ current?.id ?? '' }}</span>
          <el-tag v-if="current" size="small" :type="isSuccess(current) ? 'success' : 'danger'" style="margin-left:8px">
            {{ isSuccess(current) ? '成功' : '失败' }}
          </el-tag>
          <el-tag v-if="current?.slow === 1" size="small" type="danger" effect="dark" style="margin-left:4px">慢接口</el-tag>
        </div>
      </template>

      <el-descriptions v-if="current" :column="2" border size="small">
        <el-descriptions-item label="访问人" :span="2">
          {{ current.username || '匿名' }}<span v-if="current.userId">（ID: {{ current.userId }}）</span>
        </el-descriptions-item>
        <el-descriptions-item label="请求路径" :span="2">
          <el-tag size="small" :type="methodTagType(current.method)" class="method-tag">{{ current.method || '-' }}</el-tag>
          <span class="req-path">{{ current.path || '-' }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="查询串" :span="2">
          <span class="req-path">{{ current.query ? `?${current.query}` : '（无）' }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="HTTP 状态">
          <el-tag size="small" :type="statusTagType(current.statusCode)">{{ current.statusCode || '-' }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="业务码">
          <span :class="{ 'fail-msg': current.code !== 0 }">{{ current.code }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="耗时">
          <span :class="{ 'slow-warn': current.slow === 1 || current.durationMs > 500 }">{{ current.durationMs }}ms</span>
        </el-descriptions-item>
        <el-descriptions-item label="慢接口">
          <el-tag size="small" :type="current.slow === 1 ? 'danger' : 'info'">{{ current.slow === 1 ? '是' : '否' }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="访问时间" :span="2">{{ current.createdAt || '-' }}</el-descriptions-item>
        <el-descriptions-item label="请求 ID (RequestId)" :span="2">
          <span class="req-path">{{ current.requestId || '-' }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="IP 地址" :span="2">{{ current.ip || '-' }}</el-descriptions-item>
        <el-descriptions-item label="User-Agent" :span="2">
          <span class="ua-text">{{ current.userAgent || '-' }}</span>
        </el-descriptions-item>
      </el-descriptions>

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
.req-line {
  display: flex;
  align-items: center;
  min-width: 0;
}
.req-path {
  font-family: monospace;
  font-size: 12px;
  word-break: break-all;
}
.req-query {
  margin-top: 2px;
  padding-left: 46px;
  font-family: monospace;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}
.slow-tag {
  margin-right: 6px;
}
.slow-warn {
  color: var(--el-color-warning);
  font-weight: 600;
}
.fail-msg {
  color: var(--el-color-danger);
}
.ua-text {
  font-size: 12px;
  color: var(--el-text-color-regular);
  word-break: break-all;
}
.drawer-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
}
</style>
