<script setup lang="ts">
/**
 * 菜单管理
 */
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { Plus, Edit, Delete, Refresh, Menu as MenuIcon, CirclePlus, Search } from '@element-plus/icons-vue'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import { useMenuApi } from '~/composables/useApi'

definePageMeta({ title: '菜单管理' })

const menuApi = useMenuApi()
const loading = ref(false)
const treeData = ref<any[]>([])
const parentTreeOptions = computed(() => [
  { id: 0, name: '顶级菜单', children: treeData.value },
])

const drawerVisible = ref(false)
const drawerTitle = ref('新增菜单')
const formRef = ref<FormInstance>()
const isEdit = ref(false)
const form = reactive({
  id: 0,
  parentId: null as number | null,
  name: '',
  type: 2,
  path: '',
  icon: '',
  permission: '',
  sort: 0,
  visible: 1,
  status: 1,
})
const rules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
}

const iconNames = Object.keys(ElementPlusIconsVue).sort()
const iconSearch = ref('')
const filteredIcons = computed(() => {
  if (!iconSearch.value) return iconNames
  return iconNames.filter(n => n.toLowerCase().includes(iconSearch.value.toLowerCase()))
})

function selectIcon(name: string) {
  form.icon = name
  iconSearch.value = ''
}

async function loadTree() {
  loading.value = true
  try {
    const res = await menuApi.tree()
    treeData.value = res.tree || []
  }
  finally {
    loading.value = false
  }
}

function resetForm() {
  Object.assign(form, {
    id: 0, parentId: null, name: '', type: 2,
    path: '', icon: '',
    permission: '',
    sort: 0, visible: 1, status: 1,
  })
}

function openCreate(parent?: any) {
  resetForm()
  if (parent) form.parentId = parent.id
  drawerTitle.value = '新增菜单'
  isEdit.value = false
  drawerVisible.value = true
}

async function openEdit(row: any) {
  resetForm()
  const detail = await menuApi.detail(row.id)
  Object.assign(form, detail)
  drawerTitle.value = '编辑菜单'
  isEdit.value = true
  drawerVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  const payload = { ...form, parentId: form.parentId ?? 0 }
  if (isEdit.value) {
    await menuApi.update(form.id, payload)
    ElMessage.success('更新成功')
  }
  else {
    await menuApi.create(payload)
    ElMessage.success('新增成功')
  }
  drawerVisible.value = false
  loadTree()
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`确认删除菜单 ${row.name}?`, '提示', { type: 'warning' })
  await menuApi.remove(row.id)
  ElMessage.success('删除成功')
  loadTree()
}

const typeText = (t: number) => (t === 1 ? '目录' : t === 2 ? '菜单' : '按钮')
const typeTag = (t: number) => (t === 1 ? '' : t === 2 ? 'success' : 'warning')

onMounted(loadTree)
</script>

<template>
  <div class="page">
    <el-card>
      <div style="margin-bottom:8px">
        <el-button v-permission="'system:menu:create'" type="success" :icon="Plus" @click="openCreate()">新增根菜单</el-button>
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
        <el-table-column prop="name" label="名称" min-width="180" />
        <el-table-column label="类型" width="80">
          <template #default="{ row }">
            <el-tag :type="typeTag(row.type)" size="small">{{ typeText(row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="path" label="路径" min-width="160" />
        <el-table-column prop="permission" label="权限标识" min-width="180" />
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
            <el-button v-permission="'system:menu:create'" link type="success" :icon="CirclePlus" @click="openCreate(row)">添加子项</el-button>
            <el-button v-permission="'system:menu:update'" link type="primary" :icon="Edit" @click="openEdit(row)">编辑</el-button>
            <el-button v-permission="'system:menu:delete'" link type="danger" :icon="Delete" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-drawer v-model="drawerVisible" :title="drawerTitle" size="500px">
      <template #header>
        <div style="display:flex;align-items:center;gap:6px;font-weight:600">
          <el-icon><MenuIcon /></el-icon>
          <span>{{ drawerTitle }}</span>
        </div>
      </template>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="父节点">
          <el-tree-select
            v-model="form.parentId"
            :data="parentTreeOptions"
            :props="{ label: 'name', value: 'id', children: 'children' }"
            placeholder="顶级菜单"
            check-strictly
            clearable
            default-expand-all
            style="width:100%"
          />
        </el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model="form.type">
            <el-radio :value="1">目录</el-radio>
            <el-radio :value="2">菜单</el-radio>
            <el-radio :value="3">按钮</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item v-if="form.type !== 3" label="路径">
          <el-input v-model="form.path" placeholder="如 /system/users" />
        </el-form-item>
        <el-form-item v-if="form.type !== 3" label="图标">
          <el-popover placement="bottom" :width="420" trigger="click" popper-class="menu-icon-popper">
            <template #reference>
              <el-input
                :model-value="form.icon"
                placeholder="选择图标"
                readonly
                clearable
                @clear="form.icon = ''"
              >
                <template #prefix>
                  <el-icon v-if="form.icon"><component :is="form.icon" /></el-icon>
                </template>
                <template #suffix>
                  <el-icon class="el-input__icon"><Search /></el-icon>
                </template>
              </el-input>
            </template>
            <div class="icon-picker-body">
              <el-input v-model="iconSearch" placeholder="搜索图标..." clearable style="margin-bottom:8px" />
              <div class="icon-picker-grid">
                <div
                  v-for="name in filteredIcons"
                  :key="name"
                  class="icon-picker-item"
                  :class="{ active: form.icon === name }"
                  @click="selectIcon(name)"
                >
                  <el-icon :size="18"><component :is="name" /></el-icon>
                  <span class="icon-picker-label">{{ name }}</span>
                </div>
              </div>
            </div>
          </el-popover>
        </el-form-item>
        <el-form-item label="权限标识">
          <el-input v-model="form.permission" placeholder="如 system:user:create" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" />
        </el-form-item>
        <el-form-item label="可见">
          <el-switch v-model="form.visible" :active-value="1" :inactive-value="0" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="drawerVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">确定</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped>
.icon-picker-body {
  max-height: 420px;
}
.icon-picker-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 4px;
  max-height: 340px;
  overflow-y: auto;
}
.icon-picker-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 8px 4px;
  border-radius: 6px;
  cursor: pointer;
  transition: background-color 0.15s;
}
.icon-picker-item:hover {
  background-color: #f0f5ff;
}
.icon-picker-item.active {
  background-color: #ecf5ff;
  color: #409eff;
}
.icon-picker-label {
  margin-top: 4px;
  font-size: 10px;
  color: #606266;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
  text-align: center;
}
.icon-picker-item.active .icon-picker-label {
  color: #409eff;
}
</style>
