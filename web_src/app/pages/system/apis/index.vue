<script setup lang="ts">
/**
 * API 资源管理
 */
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { Search, Plus, Edit, Delete, Connection } from '@element-plus/icons-vue'
import { useApiResourceApi } from '~/composables/useApi'

definePageMeta({ title: 'API管理' })

const apiResourceApi = useApiResourceApi()

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const query = reactive({
  groupName: '',
  path: '',
  page: 1,
  pageSize: 10,
})

const drawerVisible = ref(false)
const drawerTitle = ref('新增 API')
const formRef = ref<FormInstance>()
const isEdit = ref(false)
const form = reactive({
  id: 0,
  path: '',
  method: 'GET',
  groupName: '',
  description: '',
})
const rules = {
  path: [{ required: true, message: '请输入路径', trigger: 'blur' }],
  method: [{ required: true, message: '请选择方法', trigger: 'change' }],
}
const methodOptions = [
  { value: 'GET', label: 'GET' },
  { value: 'POST', label: 'POST' },
  { value: 'PUT', label: 'PUT' },
  { value: 'DELETE', label: 'DELETE' },
]

const methodTagType: Record<string, string> = {
  GET: 'success',
  POST: 'primary',
  PUT: 'warning',
  DELETE: 'danger',
}

async function loadList() {
  loading.value = true
  try {
    const res = await apiResourceApi.list({ ...query })
    list.value = res.list || []
    total.value = res.total || 0
  }
  finally {
    loading.value = false
  }
}

function resetForm() {
  Object.assign(form, { id: 0, path: '', method: 'GET', groupName: '', description: '' })
}

function openCreate() {
  resetForm()
  drawerTitle.value = '新增 API'
  isEdit.value = false
  drawerVisible.value = true
}

async function openEdit(row: any) {
  resetForm()
  Object.assign(form, row)
  drawerTitle.value = '编辑 API'
  isEdit.value = true
  drawerVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  if (isEdit.value) {
    await apiResourceApi.update(form.id, {
      path: form.path,
      method: form.method,
      groupName: form.groupName,
      description: form.description,
    })
    ElMessage.success('更新成功')
  }
  else {
    await apiResourceApi.create({
      path: form.path,
      method: form.method,
      groupName: form.groupName,
      description: form.description,
    })
    ElMessage.success('新增成功')
  }
  drawerVisible.value = false
  loadList()
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`确认删除 API ${row.path}?`, '提示', { type: 'warning' })
  await apiResourceApi.remove(row.id)
  ElMessage.success('删除成功')
  loadList()
}

onMounted(loadList)
</script>

<template>
  <div class="page">
    <el-card>
      <el-form inline @submit.prevent>
        <el-form-item label="分组">
          <el-input v-model="query.groupName" placeholder="分组名称" clearable />
        </el-form-item>
        <el-form-item label="路径">
          <el-input v-model="query.path" placeholder="API 路径" clearable />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="() => { query.page = 1; loadList() }">查询</el-button>
          <el-button v-permission="'system:api:create'" type="success" :icon="Plus" @click="openCreate">新增</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="list" border stripe style="margin-top:8px">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="groupName" label="分组" min-width="120" />
        <el-table-column prop="path" label="路径" min-width="200" />
        <el-table-column label="方法" width="100">
          <template #default="{ row }">
            <el-tag :type="methodTagType[row.method] || 'info'" size="small">
              {{ row.method }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="180" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button v-permission="'system:api:update'" link type="primary" :icon="Edit" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm
              title="确认删除该 API?"
              confirm-button-text="确认"
              cancel-button-text="取消"
              @confirm="handleDelete(row)"
            >
              <template #reference>
                <el-button v-permission="'system:api:delete'" link type="danger" :icon="Delete">删除</el-button>
              </template>
            </el-popconfirm>
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

    <el-drawer v-model="drawerVisible" :title="drawerTitle" size="480px">
      <template #header>
        <div style="display:flex;align-items:center;gap:6px;font-weight:600">
          <el-icon><Connection /></el-icon>
          <span>{{ drawerTitle }}</span>
        </div>
      </template>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-form-item label="路径" prop="path">
          <el-input v-model="form.path" placeholder="如 /system/users" />
        </el-form-item>
        <el-form-item label="方法" prop="method">
          <el-select v-model="form.method" style="width:100%">
            <el-option
              v-for="opt in methodOptions"
              :key="opt.value"
              :value="opt.value"
              :label="opt.label"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="分组">
          <el-input v-model="form.groupName" placeholder="如 用户管理" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="接口描述" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="drawerVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">确定</el-button>
      </template>
    </el-drawer>
  </div>
</template>
