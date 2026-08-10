<script setup lang="ts">
/**
 * 用户管理
 */
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { Search, Plus, Edit, Delete, Key, User } from '@element-plus/icons-vue'
import { useUserApi, useRoleApi, useOrgApi } from '~/composables/useApi'

definePageMeta({ title: '用户管理' })

const userApi = useUserApi()
const roleApi = useRoleApi()
const orgApi = useOrgApi()

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const query = reactive({
  keyword: '',
  status: undefined as number | undefined,
  page: 1,
  pageSize: 10,
})
const roleOptions = ref<any[]>([])
const orgTree = ref<any[]>([])

const drawerVisible = ref(false)
const drawerTitle = ref('新增用户')
const formRef = ref<FormInstance>()
const form = reactive({
  id: 0 as number,
  username: '',
  password: '',
  nickname: '',
  email: '',
  phone: '',
  orgId: null as number | null,
  status: 1,
  remark: '',
  roleIds: [] as number[],
})
const isEdit = ref(false)
const submitting = ref(false)
const rules = computed(() => ({
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 32, message: '长度 2-32', trigger: 'blur' },
  ],
  password: isEdit.value
    ? []
    : [
        { required: true, message: '请输入密码', trigger: 'blur' },
        { min: 6, max: 32, message: '长度 6-32', trigger: 'blur' },
      ],
  nickname: [{ required: true, message: '请输入昵称', trigger: 'blur' }],
  email: [{ type: 'email', message: '邮箱格式不正确', trigger: 'blur' }],
  phone: [{
    pattern: /^1[3-9]\d{9}$/, message: '手机号格式不正确', trigger: 'blur',
  }],
  roleIds: [{
    type: 'array', required: true, min: 1, message: '请至少选择一个角色', trigger: 'change',
  }],
}))

async function loadList() {
  loading.value = true
  try {
    const res = await userApi.list({ ...query })
    list.value = res.list || []
    total.value = res.total || 0
  }
  finally {
    loading.value = false
  }
}

async function loadRoles() {
  const res = await roleApi.all()
  roleOptions.value = res.list || []
}

async function loadOrgTree() {
  const res = await orgApi.tree()
  orgTree.value = res.tree || []
}

function resetForm() {
  Object.assign(form, {
    id: 0,
    username: '',
    password: '',
    nickname: '',
    email: '',
    phone: '',
    orgId: null,
    status: 1,
    remark: '',
    roleIds: [],
  })
}

function openCreate() {
  resetForm()
  drawerTitle.value = '新增用户'
  isEdit.value = false
  drawerVisible.value = true
}

async function openEdit(row: any) {
  resetForm()
  drawerTitle.value = '编辑用户'
  isEdit.value = true
  const detail = await userApi.detail(row.id)
  Object.assign(form, {
    id: detail.id,
    username: detail.username,
    nickname: detail.nickname,
    email: detail.email,
    phone: detail.phone,
    orgId: detail.orgId || null,
    status: detail.status,
    remark: detail.remark,
    roleIds: roleOptions.value
      .filter((r: any) => detail.roles?.includes(r.code))
      .map((r: any) => r.id),
  })
  drawerVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    if (isEdit.value) {
      await userApi.update(form.id, {
        nickname: form.nickname,
        email: form.email,
        phone: form.phone,
        orgId: form.orgId || 0,
        status: form.status,
        remark: form.remark,
        roleIds: form.roleIds,
      })
      ElMessage.success('更新成功')
    }
    else {
      await userApi.create({ ...form })
      ElMessage.success('新增成功')
    }
    drawerVisible.value = false
    loadList()
  }
  catch (e: any) {
    // 响应拦截已提示, 这里保留开面让用户修改后重试
    ElMessage.error(e?.message || '提交失败')
  }
  finally {
    submitting.value = false
  }
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`确认删除用户 ${row.username}?`, '提示', { type: 'warning' })
  try {
    await userApi.remove(row.id)
    ElMessage.success('删除成功')
    loadList()
  }
  catch (e: any) {
    ElMessage.error(e?.message || '删除失败')
  }
}

async function handleResetPwd(row: any) {
  const { value } = await ElMessageBox.prompt('请输入新密码', '重置密码', {
    inputPattern: /^.{6,32}$/,
    inputErrorMessage: '密码长度 6-32',
  })
  try {
    await userApi.resetPwd(row.id, value)
    ElMessage.success('重置成功')
  }
  catch (e: any) {
    ElMessage.error(e?.message || '重置失败')
  }
}

onMounted(() => {
  loadRoles()
  loadOrgTree()
  loadList()
})
</script>

<template>
  <div class="page">
    <el-card>
      <el-form inline @submit.prevent>
        <el-form-item label="关键词">
          <el-input v-model="query.keyword" placeholder="账号/昵称" clearable @keyup.enter="() => { query.page = 1; loadList() }" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="query.status" placeholder="全部" clearable style="width:120px">
            <el-option :value="1" label="启用" />
            <el-option :value="0" label="禁用" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="() => { query.page = 1; loadList() }">查询</el-button>
          <el-button v-permission="'system:user:create'" type="success" :icon="Plus" @click="openCreate">新增</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="list" border stripe style="margin-top:8px">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="username" label="账号" />
        <el-table-column prop="nickname" label="昵称" />
        <el-table-column prop="email" label="邮箱" />
        <el-table-column prop="phone" label="手机" />
        <el-table-column prop="orgName" label="所属组织" min-width="120" />
        <el-table-column label="角色">
          <template #default="{ row }">
            <el-tag v-for="r in row.roles" :key="r" size="small" style="margin-right:4px">{{ roleOptions.find(ro => ro.code === r)?.name || r }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="180" />
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <el-button v-permission="'system:user:update'" link type="primary" :icon="Edit" @click="openEdit(row)">编辑</el-button>
            <el-button v-permission="'system:user:update'" link type="warning" :icon="Key" @click="handleResetPwd(row)">重置密码</el-button>
            <el-button v-permission="'system:user:delete'" link type="danger" :icon="Delete" @click="handleDelete(row)">删除</el-button>
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
          <el-icon><User /></el-icon>
          <span>{{ drawerTitle }}</span>
        </div>
      </template>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" :disabled="isEdit" placeholder="用户名为唯一标识, 创建后不可修改" />
        </el-form-item>
        <el-form-item v-if="!isEdit" label="密码" prop="password">
          <el-input v-model="form.password" type="password" show-password />
        </el-form-item>
        <el-form-item label="昵称" prop="nickname">
          <el-input v-model="form.nickname" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" />
        </el-form-item>
        <el-form-item label="手机" prop="phone">
          <el-input v-model="form.phone" />
        </el-form-item>
        <el-form-item label="所属组织">
          <el-tree-select
            v-model="form.orgId"
            :data="orgTree"
            :props="{ label: 'name', value: 'id', children: 'children' }"
            placeholder="请选择组织"
            check-strictly
            clearable
            default-expand-all
            style="width:100%"
          />
        </el-form-item>
        <el-form-item label="角色" prop="roleIds">
          <el-select v-model="form.roleIds" multiple style="width:100%">
            <el-option v-for="r in roleOptions" :key="r.id" :value="r.id" :label="r.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status" :disabled="isEdit && form.id === 1">
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
