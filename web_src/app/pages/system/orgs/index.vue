<script setup lang="ts">
/**
 * 组织机构管理
 */
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { Plus, Edit, Delete, Refresh, OfficeBuilding, CirclePlus } from '@element-plus/icons-vue'
import { useOrgApi } from '~/composables/useApi'

definePageMeta({ title: '组织机构管理' })

const orgApi = useOrgApi()
const loading = ref(false)
const treeData = ref<any[]>([])
const parentTreeOptions = computed(() => [
  { id: 0, name: '顶级组织', children: treeData.value },
])

const drawerVisible = ref(false)
const drawerTitle = ref('新增组织')
const formRef = ref<FormInstance>()
const isEdit = ref(false)
const form = reactive({
  id: 0,
  parentId: null as number | null,
  name: '',
  leader: '',
  phone: '',
  email: '',
  sort: 0,
  status: 1,
  remark: '',
})
const rules = {
  name: [{ required: true, message: '请输入组织名称', trigger: 'blur' }],
  email: [{ type: 'email', message: '邮箱格式不正确', trigger: 'blur' }],
  phone: [{
    pattern: /^1[3-9]\d{9}$/, message: '手机号格式不正确', trigger: 'blur',
  }],
}

async function loadTree() {
  loading.value = true
  try {
    const res = await orgApi.tree()
    treeData.value = res.tree || []
  }
  finally {
    loading.value = false
  }
}

function resetForm() {
  Object.assign(form, {
    id: 0, parentId: null, name: '', leader: '',
    phone: '', email: '', sort: 0, status: 1, remark: '',
  })
}

function openCreate(parent?: any) {
  resetForm()
  if (parent) form.parentId = parent.id
  drawerTitle.value = '新增组织'
  isEdit.value = false
  drawerVisible.value = true
}

async function openEdit(row: any) {
  resetForm()
  const detail = await orgApi.detail(row.id)
  Object.assign(form, detail)
  drawerTitle.value = '编辑组织'
  isEdit.value = true
  drawerVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  const payload = { ...form, parentId: form.parentId ?? 0 }
  if (isEdit.value) {
    await orgApi.update(form.id, payload)
    ElMessage.success('更新成功')
  }
  else {
    await orgApi.create(payload)
    ElMessage.success('新增成功')
  }
  drawerVisible.value = false
  loadTree()
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`确认删除组织 ${row.name}?`, '提示', { type: 'warning' })
  await orgApi.remove(row.id)
  ElMessage.success('删除成功')
  loadTree()
}

onMounted(loadTree)
</script>

<template>
  <div class="page">
    <el-card>
      <div style="margin-bottom:8px">
        <el-button v-permission="'system:org:create'" type="success" :icon="Plus" @click="openCreate()">新增根组织</el-button>
        <el-button :icon="Refresh" @click="loadTree">刷新</el-button>
      </div>
      <el-table
        v-loading="loading"
        :data="treeData"
        row-key="id"
        border
        stripe
        :tree-props="{ children: 'children' }"
        default-expand-all
      >
        <el-table-column prop="name" label="组织名称" min-width="180" />
        <el-table-column prop="leader" label="负责人" width="120" />
        <el-table-column prop="phone" label="联系电话" width="140" />
        <el-table-column prop="email" label="邮箱" min-width="180" />
        <el-table-column prop="sort" label="排序" width="80" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <el-button v-permission="'system:org:create'" link type="success" :icon="CirclePlus" @click="openCreate(row)">添加子组织</el-button>
            <el-button v-permission="'system:org:update'" link type="primary" :icon="Edit" @click="openEdit(row)">编辑</el-button>
            <el-button v-permission="'system:org:delete'" link type="danger" :icon="Delete" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-drawer v-model="drawerVisible" :title="drawerTitle" size="500px">
      <template #header>
        <div style="display:flex;align-items:center;gap:6px;font-weight:600">
          <el-icon><OfficeBuilding /></el-icon>
          <span>{{ drawerTitle }}</span>
        </div>
      </template>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="父节点">
          <el-tree-select
            v-model="form.parentId"
            :data="parentTreeOptions"
            :props="{ label: 'name', value: 'id', children: 'children' }"
            placeholder="顶级组织"
            check-strictly
            clearable
            default-expand-all
            style="width:100%"
          />
        </el-form-item>
        <el-form-item label="组织名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入组织名称" />
        </el-form-item>
        <el-form-item label="负责人">
          <el-input v-model="form.leader" placeholder="请输入负责人" />
        </el-form-item>
        <el-form-item label="联系电话" prop="phone">
          <el-input v-model="form.phone" placeholder="请输入联系电话" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" placeholder="请输入邮箱" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="drawerVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">确定</el-button>
      </template>
    </el-drawer>
  </div>
</template>
