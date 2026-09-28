<script setup lang="ts">
/**
 * 代码生成 (gencode): 选表 -> 配置模块/标题/列勾选 -> 预览 / zip 下载 / 直写源码树。
 * 直写仅开发环境可用 (需后端从源码目录启动且 gencode.enable=true)。
 */
import { ref, reactive, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, View, Download, DocumentChecked } from '@element-plus/icons-vue'
import { useGencodeApi, type GencodeColumnConf } from '~/composables/useApi'

definePageMeta({ title: '代码生成' })
defineOptions({ name: 'system-gencode' })

const api = useGencodeApi()

// ---- 表清单 ----
const tableLoading = ref(false)
const tables = ref<Array<{ tableName: string; tableComment: string }>>([])
const keyword = ref('')
const selectedTable = ref('')

const filteredTables = computed(() => {
  if (!keyword.value) return tables.value
  const k = keyword.value.toLowerCase()
  return tables.value.filter(t =>
    t.tableName.toLowerCase().includes(k) || (t.tableComment || '').toLowerCase().includes(k))
})

async function loadTables() {
  tableLoading.value = true
  try {
    const res = await api.tables()
    tables.value = res.list || []
  }
  catch (e: any) {
    // 开关未开启等场景给出明确提示
    tables.value = []
  }
  finally {
    tableLoading.value = false
  }
}
loadTables()

// ---- 列配置 ----
interface ColRow {
  name: string
  dataType: string
  comment: string
  goType: string
  kind: string
  skip: boolean
  inList: boolean
  inForm: boolean
  inSearch: boolean
}
const colLoading = ref(false)
const colRows = ref<ColRow[]>([])
const form = reactive({ mod: '', title: '' })

async function selectTable(name: string) {
  selectedTable.value = name
  colLoading.value = true
  try {
    const res = await api.columns(name)
    colRows.value = (res.list || []).filter(c => !c.skip)
    // 默认模块名: 表名去前缀去复数
    form.mod = name.replace(/^(biz_|sys_)/, '').replace(/s$/, '')
    form.title = ''
  }
  finally {
    colLoading.value = false
  }
}

// 提交用的列勾选 (仅传显式关闭项, 减少参数体积)
function columnConf(): GencodeColumnConf[] {
  return colRows.value.map(c => ({
    name: c.name,
    inList: c.inList,
    inForm: c.inForm,
    inSearch: c.inSearch,
  }))
}

const activeName = computed(() =>
  selectedTable.value ? `${selectedTable.value}${colRows.value.find(c => c.comment)?.comment || ''}` : '')

// ---- 预览 ----
const previewVisible = ref(false)
const previewLoading = ref(false)
const previewFiles = ref<Array<{ path: string; content: string }>>([])
const activeFile = ref('')

async function handlePreview() {
  previewLoading.value = true
  try {
    const res = await api.preview({
      table: selectedTable.value,
      mod: form.mod || undefined,
      title: form.title || undefined,
      columns: columnConf(),
    })
    previewFiles.value = res.files || []
    activeFile.value = previewFiles.value[0]?.path || ''
    previewVisible.value = true
  }
  finally {
    previewLoading.value = false
  }
}

// ---- 下载 / 直写 ----
async function handleDownload() {
  await api.download(selectedTable.value, form.mod || '', form.title || '', columnConf())
  ElMessage.success('已开始下载')
}

async function handleWrite() {
  await ElMessageBox.confirm(
    '将把生成文件直接写入前后端源码目录并自动接线 (已存在的文件会中止防止覆盖), 继续吗?',
    '写入源码',
    { type: 'warning', confirmButtonText: '写入' },
  )
  const res = await api.write({
    table: selectedTable.value,
    mod: form.mod || undefined,
    title: form.title || undefined,
    columns: columnConf(),
  })
  ElMessage.success(`已写入 ${res.written?.length || 0} 个文件, 重启后端生效 (接线已自动完成)`)
}

const ready = computed(() => !!selectedTable.value && !colLoading.value)
</script>

<template>
  <div class="page gencode-page">
    <el-row :gutter="12">
      <!-- 左: 表清单 -->
      <el-col :xs="24" :sm="24" :md="8" :lg="7">
        <el-card>
          <template #header>
            <span>数据表</span>
          </template>
          <el-input v-model="keyword" placeholder="表名/注释过滤" clearable :prefix-icon="Search" style="margin-bottom:8px" />
          <div v-loading="tableLoading" class="table-list">
            <div
              v-for="t in filteredTables"
              :key="t.tableName"
              class="table-item"
              :class="{ active: selectedTable === t.tableName }"
              @click="selectTable(t.tableName)"
            >
              <span class="tname">{{ t.tableName }}</span>
              <span v-if="t.tableComment" class="tcomment">{{ t.tableComment }}</span>
            </div>
            <el-empty v-if="!tableLoading && filteredTables.length === 0" description="无表 (或功能未启用)" :image-size="60" />
          </div>
        </el-card>
      </el-col>

      <!-- 右: 配置 + 列 -->
      <el-col :xs="24" :sm="24" :md="16" :lg="17">
        <el-card>
          <template #header>
            <span>生成配置{{ activeName ? ` - ${activeName}` : '' }}</span>
          </template>

          <el-empty v-if="!selectedTable" description="请先在左侧选择数据表" :image-size="80" />
          <template v-else>
            <el-form inline @submit.prevent>
              <el-form-item label="模块名">
                <el-input v-model="form.mod" placeholder="如 article" style="width:160px" />
              </el-form-item>
              <el-form-item label="中文标题">
                <el-input v-model="form.title" placeholder="如 文章管理" style="width:160px" />
              </el-form-item>
              <el-form-item>
                <el-button v-permission="'system:gencode:preview'" type="primary" :icon="View" :disabled="!ready" :loading="previewLoading" @click="handlePreview">预览</el-button>
                <el-button v-permission="'system:gencode:download'" type="success" :icon="Download" :disabled="!ready" @click="handleDownload">下载 zip</el-button>
                <el-button v-permission="'system:gencode:write'" type="warning" :icon="DocumentChecked" :disabled="!ready" @click="handleWrite">写入源码</el-button>
              </el-form-item>
            </el-form>

            <el-table v-loading="colLoading" :data="colRows" border stripe size="small" max-height="480">
              <el-table-column prop="name" label="列名" min-width="140" />
              <el-table-column prop="comment" label="注释" min-width="110" show-overflow-tooltip />
              <el-table-column prop="dataType" label="类型" width="100" />
              <el-table-column prop="goType" label="Go 类型" width="90" />
              <el-table-column label="控件" width="90">
                <template #default="{ row }">
                  <el-tag size="small" type="info">{{ row.kind }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="列表" width="70" align="center">
                <template #default="{ row }">
                  <el-checkbox v-model="row.inList" />
                </template>
              </el-table-column>
              <el-table-column label="表单" width="70" align="center">
                <template #default="{ row }">
                  <el-checkbox v-model="row.inForm" />
                </template>
              </el-table-column>
              <el-table-column label="搜索" width="70" align="center">
                <template #default="{ row }">
                  <el-checkbox v-model="row.inSearch" :disabled="row.goType !== 'string'" />
                </template>
              </el-table-column>
            </el-table>
          </template>
        </el-card>
      </el-col>
    </el-row>

    <!-- 预览抽屉 -->
    <el-drawer v-model="previewVisible" :title="`生成文件预览 (${previewFiles.length})`" size="760px">
      <el-tabs v-model="activeFile" tab-position="left" class="preview-tabs">
        <el-tab-pane v-for="f in previewFiles" :key="f.path" :name="f.path">
          <template #label>
            <span class="file-tab-label" :title="f.path">{{ f.path.split('/').pop() }}</span>
          </template>
          <div class="file-path">{{ f.path }}</div>
          <pre class="code-block">{{ f.content }}</pre>
        </el-tab-pane>
      </el-tabs>
    </el-drawer>
  </div>
</template>

<style scoped>
.gencode-page {
  padding: 0;
}

.table-list {
  max-height: 560px;
  overflow-y: auto;
}

.table-item {
  padding: 7px 10px;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}
.table-item:hover {
  background: var(--el-fill-color-light);
}
.table-item.active {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}
.tname {
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  font-size: 13px;
}
.tcomment {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 120px;
}

.preview-tabs {
  height: calc(100vh - 130px);
  display: flex;
}
.preview-tabs :deep(.el-tabs__content) {
  overflow: auto;
  flex: 1;
}
.file-tab-label {
  font-size: 12px;
  max-width: 140px;
  display: inline-block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: bottom;
}
.file-path {
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 6px;
  word-break: break-all;
}
.code-block {
  background: var(--el-fill-color-darker);
  color: var(--el-color-info-light-3);
  padding: 12px;
  border-radius: 6px;
  font-size: 12px;
  line-height: 1.6;
  overflow: auto;
  max-height: calc(100vh - 190px);
  margin: 0;
}
</style>
