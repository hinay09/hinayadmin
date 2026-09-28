<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Refresh, SwitchButton } from '@element-plus/icons-vue'
import { useOnlineApi } from '~/composables/useApi'

definePageMeta({ title: '在线用户' })
defineOptions({ name: 'system-online' })

const api = useOnlineApi()
const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const query = reactive({
  username: '',
  page: 1,
  pageSize: 10,
})

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
  query.page = 1
  loadList()
}

async function handleKick(row: any) {
  await ElMessageBox.confirm(
    `确认将用户「${row.username}」的此会话强制下线吗?`,
    '强制下线',
    { type: 'warning', confirmButtonText: '下线' },
  )
  await api.kick(row.sessionId)
  ElMessage.success('已强制下线')
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
            style="width:180px"
            @keyup.enter="() => { query.page = 1; loadList() }"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="() => { query.page = 1; loadList() }">查询</el-button>
          <el-button :icon="Refresh" @click="resetQuery">重置</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="list" border stripe style="margin-top:8px">
        <el-table-column label="用户名" width="140">
          <template #default="{ row }">
            {{ row.username }}
            <el-tag v-if="row.nickname" size="small" type="info" effect="plain" style="margin-left:4px">{{ row.nickname }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="ip" label="登录 IP" width="140" />
        <el-table-column label="User-Agent" min-width="200">
          <template #default="{ row }">
            <el-tooltip v-if="row.userAgent" :content="row.userAgent" placement="top">
              <span class="ua-text">{{ shortUa(row.userAgent) }}</span>
            </el-tooltip>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="loginAt" label="登录时间" width="170" />
        <el-table-column prop="lastActiveAt" label="最近活跃" width="170" />
        <el-table-column label="操作" width="110" fixed="right">
          <template #default="{ row }">
            <el-button v-permission="'system:online:kick'" link type="danger" :icon="SwitchButton" @click="handleKick(row)">下线</el-button>
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

.ua-text {
  cursor: default;
  word-break: break-all;
}
</style>
