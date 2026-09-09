<script setup lang="ts">
/**
 * 文件管理
 */
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Upload, Delete, Download } from '@element-plus/icons-vue'
import { useFileApi } from '~/composables/useApi'

definePageMeta({ title: '文件管理' })

const api = useFileApi()

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const query = reactive({
  keyword: '',
  mimeType: '',
  page: 1,
  pageSize: 10,
})

const uploadVisible = ref(false)
const uploading = ref(false)

// 格式化文件大小
function formatSize(bytes: number): string {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${size.toFixed(2)} ${units[i]}`
}

// 仅放行 http(s) 与站内绝对路径, 拦截 javascript:/data: 等危险 scheme
function fileUrl(url: string): string {
  if (!url) return ''
  if (/^https?:\/\//i.test(url)) return url
  if (url.startsWith('/')) return url
  return ''
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

async function handleUpload() {
  const input = document.createElement('input')
  input.type = 'file'
  input.onchange = async () => {
    const file = input.files?.[0]
    if (!file) return

    uploading.value = true
    try {
      const res = await api.upload(file)
      ElMessage.success(`上传成功: ${res.originalName}`)
      loadList()
    }
    catch {}
    finally {
      uploading.value = false
    }
  }
  input.click()
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`确认删除文件 ${row.originalName}?`, '提示', { type: 'warning' })
  try {
    await api.remove(row.id)
    ElMessage.success('删除成功')
    loadList()
  }
  catch {}
}

function handleDownload(row: any) {
  window.open(fileUrl(row.url), '_blank')
}

function handlePreview(row: any) {
  window.open(fileUrl(row.url), '_blank')
}

// 根据 MIME 类型返回图标
function getFileIcon(mimeType: string): string {
  if (mimeType.startsWith('image/')) return '📷'
  if (mimeType.includes('pdf')) return '📄'
  if (mimeType.includes('word') || mimeType.includes('document')) return '📝'
  if (mimeType.includes('sheet') || mimeType.includes('excel') || mimeType.includes('spreadsheet')) return '📊'
  if (mimeType.startsWith('video/')) return '🎬'
  if (mimeType.startsWith('audio/')) return '🎵'
  if (mimeType.includes('zip') || mimeType.includes('rar') || mimeType.includes('tar') || mimeType.includes('7z')) return '🗜️'
  return '📁'
}

onMounted(loadList)
</script>

<template>
  <div class="page">
    <el-card>
      <div style="display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:12px">
        <el-form inline @submit.prevent style="flex:1">
          <el-form-item label="搜索">
            <el-input v-model="query.keyword" placeholder="文件名" clearable @keyup.enter="() => { query.page = 1; loadList() }" />
          </el-form-item>
          <el-form-item label="类型">
            <el-select v-model="query.mimeType" placeholder="全部" clearable style="width:120px">
              <el-option value="image" label="图片" />
              <el-option value="application" label="文档" />
              <el-option value="video" label="视频" />
              <el-option value="audio" label="音频" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :icon="Search" @click="() => { query.page = 1; loadList() }">查询</el-button>
            <el-button type="success" :icon="Upload" :loading="uploading" @click="handleUpload">上传</el-button>
          </el-form-item>
        </el-form>
      </div>

      <el-table v-loading="loading" :data="list" border stripe style="margin-top:8px">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="文件" width="60">
          <template #default="{ row }">
            <span style="font-size:20px">{{ getFileIcon(row.mimeType) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="originalName" label="文件名" min-width="200" show-overflow-tooltip />
        <el-table-column label="大小" width="100">
          <template #default="{ row }">
            {{ formatSize(row.size) }}
          </template>
        </el-table-column>
        <el-table-column prop="mimeType" label="类型" width="100" show-overflow-tooltip />
        <el-table-column prop="extension" label="后缀" width="70" />
        <el-table-column prop="createdAt" label="上传时间" width="180" />
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.mimeType?.startsWith('image/')" link type="primary" @click="handlePreview(row)">
              预览
            </el-button>
            <el-button link type="primary" :icon="Download" @click="handleDownload(row)">下载</el-button>
            <el-button v-permission="'system:file:delete'" link type="danger" :icon="Delete" @click="handleDelete(row)">删除</el-button>
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
