<script setup lang="ts">
/**
 * 角色管理
 */
import { ref, reactive, computed, onMounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { Search, Plus, Edit, Delete, Key, UserFilled } from '@element-plus/icons-vue'
import { useRoleApi, useMenuApi, useApiResourceApi } from '~/composables/useApi'

definePageMeta({ title: '角色管理' })

const roleApi = useRoleApi()
const menuApi = useMenuApi()
const apiResourceApi = useApiResourceApi()

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const query = reactive({
  keyword: '',
  status: undefined as number | undefined,
  page: 1,
  pageSize: 10,
})

const drawerVisible = ref(false)
const drawerTitle = ref('新增角色')
const formRef = ref<FormInstance>()
const isEdit = ref(false)
const form = reactive({
  id: 0,
  name: '',
  code: '',
  sort: 0,
  status: 1,
  remark: '',
})
const rules = {
  name: [{ required: true, message: '请输入角色名', trigger: 'blur' }],
  code: [{ required: true, message: '请输入编码', trigger: 'blur' }],
}

/* ---- 权限分配对话框 ---- */
const permDialogVisible = ref(false)
const permDialogTitle = ref('')
const currentRole = ref<any>(null)
const activeTab = ref('menu')

// 菜单权限
const menuTreeRef = ref()
const menuTreeData = ref<any[]>([])
const checkedMenuIds = ref<number[]>([])

// 接口权限
const apiList = ref<any[]>([])
const checkedApis = ref<Array<{ path: string; method: string }>>([])
const apiListLoading = ref(false)

const apiGrouped = computed(() => {
  const map = new Map<string, any[]>()
  for (const api of apiList.value) {
    const key = api.groupName || '未分组'
    if (!map.has(key)) map.set(key, [])
    map.get(key)!.push(api)
  }
  return Array.from(map.entries()).map(([name, apis]) => ({ name, apis }))
})

const methodTagType: Record<string, string> = {
  GET: 'success',
  POST: 'primary',
  PUT: 'warning',
  DELETE: 'danger',
}

function isApiChecked(api: any): boolean {
  return checkedApis.value.some(a => a.path === api.path && a.method === api.method)
}

function toggleApi(api: any, checked: boolean) {
  if (checked) {
    if (!isApiChecked(api)) {
      checkedApis.value.push({ path: api.path, method: api.method })
    }
  }
  else {
    checkedApis.value = checkedApis.value.filter(a => !(a.path === api.path && a.method === api.method))
  }
}

function toggleGroup(apis: any[], checked: boolean) {
  for (const api of apis) {
    toggleApi(api, checked)
  }
}

function isGroupAllChecked(apis: any[]): boolean {
  return apis.every(api => isApiChecked(api))
}

function isGroupIndeterminate(apis: any[]): boolean {
  const checkedCount = apis.filter(api => isApiChecked(api)).length
  return checkedCount > 0 && checkedCount < apis.length
}

async function loadList() {
  loading.value = true
  try {
    const res = await roleApi.list({ ...query })
    list.value = res.list || []
    total.value = res.total || 0
  }
  finally {
    loading.value = false
  }
}

function resetForm() {
  Object.assign(form, { id: 0, name: '', code: '', sort: 0, status: 1, remark: '' })
}

function openCreate() {
  resetForm()
  drawerTitle.value = '新增角色'
  isEdit.value = false
  drawerVisible.value = true
}

function openEdit(row: any) {
  resetForm()
  Object.assign(form, row)
  drawerTitle.value = '编辑角色'
  isEdit.value = true
  drawerVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  if (isEdit.value) {
    await roleApi.update(form.id, {
      name: form.name,
      sort: form.sort,
      status: form.status,
      remark: form.remark,
    })
    ElMessage.success('更新成功')
  }
  else {
    await roleApi.create({ ...form })
    ElMessage.success('新增成功')
  }
  drawerVisible.value = false
  loadList()
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`确认删除角色 ${row.name}?`, '提示', { type: 'warning' })
  await roleApi.remove(row.id)
  ElMessage.success('删除成功')
  loadList()
}

async function openAssignPerm(row: any) {
  currentRole.value = row
  permDialogTitle.value = `权限分配 - ${row.name}`
  activeTab.value = 'menu'

  // 先清空旧的勾选状态, 防止上一次的残留
  if (menuTreeRef.value) {
    menuTreeRef.value.setCheckedKeys([], false)
  }
  menuTreeData.value = []
  checkedMenuIds.value = []
  apiList.value = []
  checkedApis.value = []

  apiListLoading.value = true
  try {
    // 并行加载菜单树、角色已绑定菜单、API 列表、角色已绑定 API
    const [treeRes, menuRes, apis, assignedApis] = await Promise.all([
      menuApi.tree(),
      roleApi.getMenus(row.id),
      apiResourceApi.all(),
      roleApi.getApis(row.id),
    ])
    menuTreeData.value = treeRes.tree || []
    checkedMenuIds.value = (menuRes.menuIds || []).map((v: any) => Number(v))
    apiList.value = apis.list || []
    checkedApis.value = assignedApis.list || []
  }
  finally {
    apiListLoading.value = false
  }

  // 数据就绪后再打开 dialog, 由 @opened 回调完成菜单回显
  permDialogVisible.value = true
}

// el-dialog 完全打开后再设置树勾选, 确保 el-tree DOM 已挂载
async function handlePermDialogOpened() {
  await nextTick()
  if (menuTreeRef.value) {
    menuTreeRef.value.setCheckedKeys(checkedMenuIds.value, false)
  }
}

async function handlePermSubmit() {
  if (!currentRole.value) return

  // 菜单权限: 仅提交 checked 节点, 半选父节点由 el-tree 自动推导, 避免回显时过度勾选
  if (menuTreeRef.value) {
    const menuIds = menuTreeRef.value.getCheckedKeys() as number[]
    await roleApi.assignMenus(currentRole.value.id, menuIds)
  }

  // 接口权限
  // eslint-disable-next-line no-console
  console.log('[perm] submit assignApis payload =', {
    roleId: currentRole.value.id,
    apis: JSON.parse(JSON.stringify(checkedApis.value)),
  })
  await roleApi.assignApis(currentRole.value.id, checkedApis.value)

  ElMessage.success('保存成功')
  permDialogVisible.value = false
}

onMounted(loadList)
</script>

<template>
  <div class="page">
    <el-card>
      <el-form inline @submit.prevent>
        <el-form-item label="关键词">
          <el-input v-model="query.keyword" placeholder="名称/编码" clearable />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="() => { query.page = 1; loadList() }">查询</el-button>
          <el-button v-permission="'system:role:create'" type="success" :icon="Plus" @click="openCreate">新增</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="list" border stripe style="margin-top:8px">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="code" label="编码" />
        <el-table-column prop="sort" label="排序" width="80" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" />
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button v-permission="'system:role:update'" link type="primary" :icon="Edit" @click="openEdit(row)">编辑</el-button>
            <el-button v-permission="'system:role:assign'" link type="warning" :icon="Key" @click="openAssignPerm(row)">分配权限</el-button>
            <el-button v-permission="'system:role:delete'" link type="danger" :icon="Delete" @click="handleDelete(row)">删除</el-button>
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

    <el-drawer v-model="drawerVisible" :title="drawerTitle" size="420px">
      <template #header>
        <div style="display:flex;align-items:center;gap:6px;font-weight:600">
          <el-icon><UserFilled /></el-icon>
          <span>{{ drawerTitle }}</span>
        </div>
      </template>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="编码" prop="code">
          <el-input v-model="form.code" :disabled="isEdit" />
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
        <el-button type="primary" @click="handleSubmit">确定</el-button>
      </template>
    </el-drawer>

    <el-dialog v-model="permDialogVisible" :title="permDialogTitle" width="680px" top="5vh" @opened="handlePermDialogOpened">
      <el-tabs v-model="activeTab">
        <!-- 菜单权限 Tab -->
        <el-tab-pane label="菜单权限" name="menu">
          <el-tree
            ref="menuTreeRef"
            :data="menuTreeData"
            show-checkbox
            node-key="id"
            :props="{ label: 'name', children: 'children' }"
            default-expand-all
            check-strictly
          >
            <template #default="{ data }">
              <span style="display: flex; align-items: center; gap: 6px">
                <span>{{ data.name }}</span>
                <el-tag v-if="data.type === 1" size="small" type="info">目录</el-tag>
                <el-tag v-else-if="data.type === 2" size="small" type="warning">菜单</el-tag>
                <el-tag v-else-if="data.type === 3" size="small" type="danger">按钮</el-tag>
              </span>
            </template>
          </el-tree>
        </el-tab-pane>

        <!-- 接口权限 Tab -->
        <el-tab-pane label="接口权限" name="api">
          <div v-loading="apiListLoading" style="max-height: 500px; overflow-y: auto">
            <div v-for="group in apiGrouped" :key="group.name" style="margin-bottom: 16px">
              <div style="margin-bottom: 8px; border-bottom: 1px solid #ebeef5; padding-bottom: 6px">
                <el-checkbox
                  :model-value="isGroupAllChecked(group.apis)"
                  :indeterminate="isGroupIndeterminate(group.apis)"
                  @change="(val: boolean) => toggleGroup(group.apis, val)"
                >
                  <span style="font-weight: 600; font-size: 14px">{{ group.name }}</span>
                </el-checkbox>
              </div>
              <el-row :gutter="8">
                <el-col v-for="api in group.apis" :key="api.id" :span="24" style="margin-bottom: 6px">
                  <el-checkbox
                    :model-value="isApiChecked(api)"
                    @change="(val: boolean) => toggleApi(api, val)"
                  >
                    <el-tag :type="methodTagType[api.method] || 'info'" size="small" style="margin-right: 4px">
                      {{ api.method }}
                    </el-tag>
                    <span>{{ api.path }}</span>
                    <span v-if="api.description" style="color: #909399; margin-left: 8px">
                      {{ api.description }}
                    </span>
                  </el-checkbox>
                </el-col>
              </el-row>
            </div>
            <el-empty v-if="apiGrouped.length === 0" description="暂无 API 资源" />
          </div>
        </el-tab-pane>
      </el-tabs>
      <template #footer>
        <el-button @click="permDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handlePermSubmit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>
