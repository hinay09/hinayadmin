<script setup lang="ts">
/**
 * CMS 文章管理: 列表 + 发布/下架动作; 编辑跳独立整页 (/cms/article/edit)。
 */
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Plus, Edit, Delete, Top } from '@element-plus/icons-vue'
import { useCmsArticleApi, useCmsCategoryApi, type CmsArticleItem, type CmsCategoryItem } from '~/composables/useApi'

definePageMeta({ title: '文章管理' })
defineOptions({ name: 'cms-article' })

const api = useCmsArticleApi()
const categoryApi = useCmsCategoryApi()
const router = useRouter()

// ============================================================
// 列表
// ============================================================
const loading = ref(false)
const list = ref<CmsArticleItem[]>([])
const total = ref(0)
const query = reactive({
  keyword: '',
  categoryId: undefined as number | undefined,
  status: undefined as number | undefined,
  page: 1,
  pageSize: 10,
})

const categories = ref<CmsCategoryItem[]>([])

async function load() {
  loading.value = true
  try {
    const res = await api.list({ ...query })
    list.value = res.list || []
    total.value = res.total || 0
  }
  finally { loading.value = false }
}

async function loadCategories() {
  try {
    const res = await categoryApi.all()
    categories.value = res.list || []
  }
  catch {}
}

// ============================================================
// 操作
// ============================================================
function openEdit(row?: CmsArticleItem) {
  router.push(row ? `/cms/article/edit?id=${row.id}` : '/cms/article/edit')
}

async function handlePublish(row: CmsArticleItem) {
  try {
    await api.publish(row.id)
    ElMessage.success('已发布')
    load()
  }
  catch {}
}

async function handleOffline(row: CmsArticleItem) {
  await ElMessageBox.confirm(`下架文章「${row.title}」? 下架后内容保留, 可再次发布。`, '下架确认', { type: 'warning' })
  try {
    await api.offline(row.id)
    ElMessage.success('已下架')
    load()
  }
  catch {}
}

async function handleDelete(row: CmsArticleItem) {
  await ElMessageBox.confirm(`删除文章「${row.title}」?`, '删除确认', { type: 'warning' })
  try {
    await api.remove(row.id)
    ElMessage.success('已删除')
    load()
  }
  catch {}
}

const statusTag = (s: number) => (s === 1 ? 'success' : s === 2 ? 'info' : 'warning')
const statusText = (s: number) => (s === 1 ? '已发布' : s === 2 ? '已下架' : '草稿')

onMounted(() => {
  load()
  loadCategories()
})
</script>

<template>
  <div class="page">
    <el-card>
      <el-form inline @submit.prevent>
        <el-form-item label="标题">
          <el-input v-model="query.keyword" placeholder="文章标题" clearable
            @keyup.enter="() => { query.page = 1; load() }" />
        </el-form-item>
        <el-form-item label="栏目">
          <el-select v-model="query.categoryId" clearable placeholder="全部" style="width:150px">
            <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="query.status" clearable placeholder="全部" style="width:110px">
            <el-option label="草稿" :value="0" />
            <el-option label="已发布" :value="1" />
            <el-option label="已下架" :value="2" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="() => { query.page = 1; load() }">查询</el-button>
          <el-button v-permission="'cms:article:create'" type="success" :icon="Plus" @click="openEdit()">写文章</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="list" border stripe style="margin-top:8px">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="封面" width="90">
          <template #default="{ row }">
            <el-image v-if="row.cover" :src="row.cover" fit="cover"
              :preview-src-list="[row.cover]" preview-teleported
              style="width:60px;height:40px;border-radius:4px" />
            <span v-else class="no-cover">—</span>
          </template>
        </el-table-column>
        <el-table-column label="标题" min-width="220">
          <template #default="{ row }">
            <span class="title-cell">
              <el-icon v-if="row.top === 1" color="var(--el-color-danger)"><Top /></el-icon>
              {{ row.title }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="栏目" width="120">
          <template #default="{ row }">{{ row.categoryName || '未分类' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="tags" label="标签" width="140" show-overflow-tooltip>
          <template #default="{ row }">{{ row.tags || '—' }}</template>
        </el-table-column>
        <el-table-column prop="publishAt" label="发布时间" width="170">
          <template #default="{ row }">{{ row.publishAt || '—' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="230" fixed="right">
          <template #default="{ row }">
            <el-button v-permission="'cms:article:update'" link type="primary" :icon="Edit" @click="openEdit(row)">编辑</el-button>
            <el-button v-if="row.status !== 1" v-permission="'cms:article:publish'" link type="success" @click="handlePublish(row)">发布</el-button>
            <el-button v-else v-permission="'cms:article:offline'" link type="warning" @click="handleOffline(row)">下架</el-button>
            <el-button v-permission="'cms:article:delete'" link type="danger" :icon="Delete" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination v-model:current-page="query.page" v-model:page-size="query.pageSize"
        :total="total" :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next, jumper"
        style="margin-top:16px;justify-content:flex-end"
        @current-change="load" @size-change="load" />
    </el-card>
  </div>
</template>

<style scoped>
.title-cell {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.no-cover {
  color: var(--el-text-color-placeholder);
}
</style>
