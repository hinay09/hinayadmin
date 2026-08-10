<script setup lang="ts">
/**
 * 全局配置管理
 * 支持按类型筛选、新增/编辑配置项
 */
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { Search, Plus, Edit, Delete } from '@element-plus/icons-vue'
import { useConfigApi, type ConfigItem } from '~/composables/useApi'

definePageMeta({ title: '全局配置' })

const configApi = useConfigApi()

// ============================================================
// 列表
// ============================================================
const loading = ref(false)
const list = ref<ConfigItem[]>([])
const total = ref(0)
const query = reactive({
  keyword: '',
  configType: undefined as number | undefined,
  page: 1,
  pageSize: 10,
})

const configTypeOptions = [
  { label: '文本', value: 0 },
  { label: '数字', value: 1 },
  { label: '布尔', value: 2 },
  { label: 'JSON', value: 3 },
]

function typeLabel(t: number) {
  return configTypeOptions.find(o => o.value === t)?.label ?? '文本'
}

function typeTagType(t: number) {
  const map: Record<number, string> = { 0: '', 1: 'warning', 2: 'success', 3: 'info' }
  return map[t] ?? ''
}

async function loadList() {
  loading.value = true
  try {
    const params: any = { ...query }
    if (params.configType === undefined) delete params.configType
    const res = await configApi.list(params)
    list.value = res.list || []
    total.value = res.total || 0
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  query.page = 1
  loadList()
}

function resetQuery() {
  query.keyword = ''
  query.configType = undefined
  query.page = 1
  query.pageSize = 10
  loadList()
}

// ============================================================
// 新增 / 编辑 Drawer
// ============================================================
const drawerVisible = ref(false)
const drawerTitle = ref('新增配置')
const formRef = ref<FormInstance>()
const form = reactive({
  id: 0,
  configKey: '',
  configValue: '',
  configType: 0,
  name: '',
  remark: '',
  status: 1,
  sort: 0,
})
const isEdit = ref(false)
const submitting = ref(false)

const rules = computed(() => ({
  configKey: [{ required: true, message: '请输入配置键', trigger: 'blur' }],
  name: [{ required: true, message: '请输入配置名称', trigger: 'blur' }],
}))

function resetForm() {
  Object.assign(form, {
    id: 0, configKey: '', configValue: '', configType: 0,
    name: '', remark: '', status: 1, sort: 0,
  })
}

function openCreate() {
  resetForm()
  drawerTitle.value = '新增配置'
  isEdit.value = false
  drawerVisible.value = true
}

function openEdit(row: ConfigItem) {
  resetForm()
  drawerTitle.value = '编辑配置'
  isEdit.value = true
  Object.assign(form, {
    id: row.id,
    configKey: row.configKey,
    configValue: row.configValue,
    configType: row.configType,
    name: row.name,
    remark: row.remark,
    status: row.status,
    sort: row.sort,
  })
  drawerVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    const data = {
      configKey: form.configKey,
      configValue: form.configValue,
      configType: form.configType,
      name: form.name,
      remark: form.remark,
      status: form.status,
      sort: form.sort,
    }
    if (isEdit.value) {
      await configApi.update(form.id, data)
      ElMessage.success('更新成功')
    } else {
      await configApi.create(data)
      ElMessage.success('新增成功')
    }
    drawerVisible.value = false
    loadList()
  } catch {
  } finally {
    submitting.value = false
  }
}

async function handleDelete(row: ConfigItem) {
  await ElMessageBox.confirm(`确认删除配置「${row.name}」?`, '提示', { type: 'warning' })
  try {
    await configApi.remove(row.id)
    ElMessage.success('删除成功')
    loadList()
  } catch {
  }
}

onMounted(() => loadList())
</script>

<template>
  <div class="page">
    <el-card>
      <!-- 搜索栏 -->
      <el-form inline @submit.prevent>
        <el-form-item label="关键词">
          <el-input v-model="query.keyword" placeholder="配置键/名称" clearable
            @keyup.enter="() => { query.page = 1; loadList() }" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="query.configType" placeholder="全部" clearable style="width:140px"
            @change="handleSearch">
            <el-option v-for="o in configTypeOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
          <el-button v-permission="'system:config:create'" type="success" :icon="Plus" @click="openCreate">
            新增配置
          </el-button>
        </el-form-item>
      </el-form>

      <!-- 列表 -->
      <el-table v-loading="loading" :data="list" border stripe style="margin-top:8px">
        <el-table-column prop="configKey" label="配置键" min-width="180" show-overflow-tooltip />
        <el-table-column prop="name" label="配置名称" min-width="150" show-overflow-tooltip />
        <el-table-column label="配置类型" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="typeTagType(row.configType)" size="small">{{ typeLabel(row.configType) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="configValue" label="配置值" min-width="200" show-overflow-tooltip />
        <el-table-column prop="sort" label="排序" width="70" align="center" />
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="140" show-overflow-tooltip />
        <el-table-column label="操作" width="180" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-permission="'system:config:update'" link type="primary" :icon="Edit"
              @click="openEdit(row)">编辑</el-button>
            <el-button v-permission="'system:config:delete'" link type="danger" :icon="Delete"
              @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
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

    <!-- 新增/编辑 Drawer -->
    <el-drawer v-model="drawerVisible" :title="drawerTitle" size="500px">
      <template #header>
        <div style="display:flex;align-items:center;gap:6px;font-weight:600">
          <span>{{ drawerTitle }}</span>
        </div>
      </template>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="配置键" prop="configKey">
          <el-input v-model="form.configKey" :disabled="isEdit" placeholder="如 sys.site_name" />
        </el-form-item>
        <el-form-item label="配置名称" prop="name">
          <el-input v-model="form.name" placeholder="如 站点名称" />
        </el-form-item>
        <el-form-item label="配置类型">
          <el-select v-model="form.configType" style="width:100%">
            <el-option v-for="o in configTypeOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="配置值">
          <el-input
            v-model="form.configValue"
            :type="form.configType === 3 ? 'textarea' : 'text'"
            :rows="form.configType === 3 ? 4 : 1"
            :placeholder="form.configType === 2 ? 'true / false' : '请输入配置值'"
          />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" />
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="drawerVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-drawer>
  </div>
</template>
