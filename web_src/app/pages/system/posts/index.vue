<script setup lang="ts">
/**
 * 岗位管理: 岗位 CRUD + 岗位成员 (谁在哪个组织担任该岗位)。
 * 岗位是审批人解析依据: "指定岗位"按岗位找人; "部门主管"取组织内挂主管岗者。
 */
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { Search, Plus, Edit, Delete, User } from '@element-plus/icons-vue'
import { usePostApi, useUserApi, useOrgApi, type PostItem, type PostMemberItem } from '~/composables/useApi'

definePageMeta({ title: '岗位管理' })
defineOptions({ name: 'system-posts' })

const api = usePostApi()
const userApi = useUserApi()
const orgApi = useOrgApi()

// ============================================================
// 岗位列表
// ============================================================
const loading = ref(false)
const list = ref<PostItem[]>([])
const total = ref(0)
const query = reactive({ keyword: '', status: undefined as number | undefined, page: 1, pageSize: 10 })

async function load() {
  loading.value = true
  try {
    const res = await api.list({ ...query })
    list.value = res.list || []
    total.value = res.total || 0
  } finally { loading.value = false }
}

// ============================================================
// 新增/编辑
// ============================================================
const drawerVisible = ref(false)
const drawerTitle = ref('新增岗位')
const isEdit = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()
const form = reactive({ id: 0, postCode: '', postName: '', postKind: 1, sort: 0, status: 1, remark: '' })
const rules = {
  postCode: [{ required: true, message: '请输入岗位编码', trigger: 'blur' }],
  postName: [{ required: true, message: '请输入岗位名称', trigger: 'blur' }],
  postKind: [{ required: true, message: '请选择岗位类型', trigger: 'change' }],
}

function resetForm() {
  Object.assign(form, { id: 0, postCode: '', postName: '', postKind: 1, sort: 0, status: 1, remark: '' })
}

function openCreate() {
  resetForm()
  isEdit.value = false
  drawerTitle.value = '新增岗位'
  drawerVisible.value = true
}

function openEdit(row: PostItem) {
  resetForm()
  Object.assign(form, {
    id: row.id, postCode: row.postCode, postName: row.postName,
    postKind: row.postKind, sort: row.sort, status: row.status, remark: row.remark,
  })
  isEdit.value = true
  drawerTitle.value = '编辑岗位'
  drawerVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    const data = {
      postCode: form.postCode, postName: form.postName, postKind: form.postKind,
      sort: form.sort, status: form.status, remark: form.remark,
    }
    if (isEdit.value) {
      await api.update(form.id, data)
      ElMessage.success('更新成功')
    } else {
      await api.create(data)
      ElMessage.success('新增成功')
    }
    drawerVisible.value = false
    load()
  } catch {} finally { submitting.value = false }
}

async function handleDelete(row: PostItem) {
  await ElMessageBox.confirm(`删除岗位「${row.postName}」? 其挂岗关系将一并清除。`, '删除确认', { type: 'warning' })
  try {
    await api.remove(row.id)
    ElMessage.success('已删除')
    load()
  } catch {}
}

// ============================================================
// 岗位成员
// ============================================================
const memberVisible = ref(false)
const memberLoading = ref(false)
const members = ref<PostMemberItem[]>([])
const currentPost = ref<PostItem | null>(null)
const memberForm = reactive({ userId: undefined as number | undefined, orgId: undefined as number | undefined })
const memberSubmitting = ref(false)

const userOptions = ref<{ id: number, label: string }[]>([])
const orgTree = ref<any[]>([])

async function openMembers(row: PostItem) {
  currentPost.value = row
  memberVisible.value = true
  memberForm.userId = undefined
  memberForm.orgId = undefined
  loadMembers()
  if (!userOptions.value.length) {
    try {
      // 复用用户列表接口取选项 (仅取启用用户)
      const res = await userApi.list({ page: 1, pageSize: 500, status: 1 })
      userOptions.value = (res.list || []).map((u: any) => ({ id: u.id, label: u.nickname || u.username }))
    } catch {}
  }
  if (!orgTree.value.length) {
    try {
      const res = await orgApi.tree()
      orgTree.value = res || []
    } catch {}
  }
}

async function loadMembers() {
  if (!currentPost.value) return
  memberLoading.value = true
  try {
    const res = await api.members(currentPost.value.id)
    members.value = res.list || []
  } finally { memberLoading.value = false }
}

async function handleMemberAdd() {
  if (!currentPost.value || !memberForm.userId) { ElMessage.warning('请选择用户'); return }
  memberSubmitting.value = true
  try {
    await api.memberAdd(currentPost.value.id, { userId: memberForm.userId, orgId: memberForm.orgId || 0 })
    ElMessage.success('已添加')
    memberForm.userId = undefined
    loadMembers()
    load()
  } catch {} finally { memberSubmitting.value = false }
}

async function handleMemberRemove(row: PostMemberItem) {
  try {
    await api.memberRemove(row.id)
    ElMessage.success('已移除')
    loadMembers()
    load()
  } catch {}
}

onMounted(load)
</script>

<template>
  <div class="page">
    <el-card>
      <el-alert type="info" :closable="false" show-icon style="margin-bottom:12px"
        title="岗位用于审批人解析: 审批节点可选「指定岗位」; 标记为主管岗的岗位用于「部门主管」解析 (发起人组织起逐级向上)。" />

      <el-form inline @submit.prevent>
        <el-form-item label="搜索">
          <el-input v-model="query.keyword" placeholder="编码/名称" clearable
            @keyup.enter="() => { query.page = 1; load() }" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="query.status" clearable placeholder="全部" style="width:110px">
            <el-option label="启用" :value="1" />
            <el-option label="禁用" :value="0" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="() => { query.page = 1; load() }">查询</el-button>
          <el-button v-permission="'system:post:create'" type="success" :icon="Plus" @click="openCreate">新增岗位</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="list" border stripe style="margin-top:8px">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="postCode" label="岗位编码" width="160" />
        <el-table-column prop="postName" label="岗位名称" width="160" />
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="row.postKind === 2 ? 'warning' : 'info'" size="small">
              {{ row.postKind === 2 ? '主管岗' : '普通岗' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="members" label="成员数" width="80" align="center" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="70" />
        <el-table-column prop="remark" label="备注" min-width="140" show-overflow-tooltip />
        <el-table-column label="操作" width="250" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" :icon="User" @click="openMembers(row)">成员</el-button>
            <el-button v-permission="'system:post:update'" link type="primary" :icon="Edit" @click="openEdit(row)">编辑</el-button>
            <el-button v-permission="'system:post:delete'" link type="danger" :icon="Delete" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination v-model:current-page="query.page" v-model:page-size="query.pageSize"
        :total="total" :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next, jumper"
        style="margin-top:16px;justify-content:flex-end"
        @current-change="load" @size-change="load" />
    </el-card>

    <!-- 岗位编辑 -->
    <el-drawer v-model="drawerVisible" :title="drawerTitle" size="440px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="岗位编码" prop="postCode">
          <el-input v-model="form.postCode" placeholder="如 hr / dept_leader" maxlength="32" />
        </el-form-item>
        <el-form-item label="岗位名称" prop="postName">
          <el-input v-model="form.postName" maxlength="64" />
        </el-form-item>
        <el-form-item label="岗位类型" prop="postKind">
          <el-radio-group v-model="form.postKind">
            <el-radio :value="1">普通岗</el-radio>
            <el-radio :value="2">主管岗</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="2" maxlength="200" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="drawerVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-drawer>

    <!-- 成员管理 -->
    <el-dialog v-model="memberVisible" :title="`岗位成员 - ${currentPost?.postName || ''}`" width="640px" top="6vh">
      <el-form inline>
        <el-form-item label="用户">
          <el-select v-model="memberForm.userId" filterable placeholder="选择用户" style="width:180px">
            <el-option v-for="u in userOptions" :key="u.id" :label="u.label" :value="u.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="组织">
          <el-tree-select v-model="memberForm.orgId" :data="orgTree" node-key="id"
            :props="{ label: 'name', children: 'children' }" check-strictly clearable
            placeholder="不限定组织" style="width:180px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="memberSubmitting" @click="handleMemberAdd">添加</el-button>
        </el-form-item>
      </el-form>
      <el-table v-loading="memberLoading" :data="members" border size="small" max-height="380">
        <el-table-column prop="userName" label="用户" width="140" />
        <el-table-column prop="orgName" label="组织" min-width="160">
          <template #default="{ row }">{{ row.orgName || '不限定组织' }}</template>
        </el-table-column>
        <el-table-column prop="createAt" label="添加时间" width="170" />
        <el-table-column label="操作" width="90">
          <template #default="{ row }">
            <el-button link type="danger" size="small" @click="handleMemberRemove(row)">移除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>
