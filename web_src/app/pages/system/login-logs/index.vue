<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Refresh, Delete } from '@element-plus/icons-vue'
import { useLoginLogApi } from '~/composables/useApi'

definePageMeta({ title: '登录日志' })
defineOptions({ name: 'system-login-logs' })

const api = useLoginLogApi()
const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const query = reactive({
  username: '',
  ip: '',
  status: '',
  startAt: '',
  endAt: '',
  page: 1,
  pageSize: 10,
})

const statusTagType = (status: number): string => (status === 1 ? 'success' : 'danger')

// 截断过长的 UA, 完整内容放 tooltip
const shortUa = (ua: string): string => (ua && ua.length > 46 ? `${ua.slice(0, 46)}…` : (ua || '-'))

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
  query.username = ''
  query.ip = ''
  query.status = ''
  query.startAt = ''
  query.endAt = ''
  query.page = 1
  loadList()
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`确认删除日志 #${row.id} 吗?`, '提示', { type: 'warning' })
  await api.delete(row.id)
  ElMessage.success('已删除')
  loadList()
}


onMounted(loadList)
</script>

<template>
  <div class="page">
    <el-card>
      <el-form inline @submit.prevent>
        <el-form-item label="用户名">
          <el-input
            v-model="query.username"
            placeholder="模糊匹配"
            clearable
            style="width:150px"
            @keyup.enter="() => { query.page = 1; loadList() }"
          />
        </el-form-item>
        <el-form-item label="IP">
          <el-input
            v-model="query.ip"
            placeholder="模糊匹配"
            clearable
            style="width:140px"
            @keyup.enter="() => { query.page = 1; loadList() }"
          />
        </el-form-item>
        <el-form-item label="结果">
          <el-select v-model="query.status" placeholder="全部" clearable style="width:100px">
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
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="结果" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="statusTagType(row.status)">
              {{ row.status === 1 ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="username" label="用户名" width="130">
          <template #default="{ row }">
            {{ row.username || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="失败原因" min-width="150">
          <template #default="{ row }">
            <span :class="{ 'fail-msg': row.status === 0 }">{{ row.message || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="ip" label="IP" width="140" />
        <el-table-column label="User-Agent" min-width="200">
          <template #default="{ row }">
            <el-tooltip v-if="row.userAgent" :content="row.userAgent" placement="top">
              <span class="ua-text">{{ shortUa(row.userAgent) }}</span>
            </el-tooltip>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="登录时间" width="170" />
        <el-table-column label="操作" width="80" fixed="right">
          <template #default="{ row }">
            <el-button v-permission="'system:login-log:delete'" link type="danger" :icon="Delete" @click="handleDelete(row)">删除</el-button>
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
  </div>
</template>

<style scoped>
.page {
  padding: 0;
}

.table-actions {
  margin-top: 4px;
}

.fail-msg {
  color: var(--el-color-danger);
}

.ua-text {
  cursor: default;
  word-break: break-all;
}
</style>
