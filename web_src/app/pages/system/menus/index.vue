<script setup lang="ts">
/**
 * 菜单管理
 * 列表搜索: 纯前端按 名称/路径/权限标识 模糊过滤 (多词空格分隔逐词 AND, 与顶栏菜单搜索同口径),
 * 命中节点保留祖先链、收起子树; default-expand-all 下数据替换后自动全部展开, 无需手动展开。
 */
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { Plus, Edit, Delete, Refresh, Menu as MenuIcon, CirclePlus, Search } from '@element-plus/icons-vue'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import { useMenuApi } from '~/composables/useApi'

definePageMeta({ title: '菜单管理' })
defineOptions({ name: 'system-menus' })

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

/** 列表搜索: 关键词拆词 (小写、去空), 逐词 AND 命中 */
const keyword = ref('')
const terms = computed(() => keyword.value.trim().toLowerCase().split(/\s+/).filter(Boolean))
const searchActive = computed(() => terms.value.length > 0)

/** 过滤结果树: 自身命中的节点收起子树, 未命中但子孙命中的节点保留为祖先链 */
const searchResult = computed(() => {
  const ts = terms.value
  if (!ts.length) return { tree: treeData.value as any[], count: 0 }
  let count = 0
  const walk = (nodes: any[]): any[] => {
    const out: any[] = []
    for (const node of nodes) {
      const hay = `${node.name || ''} ${node.path || ''} ${node.permission || ''}`.toLowerCase()
      if (ts.every(t => hay.includes(t))) {
        count++
        out.push({ ...node, children: [] })
      }
      else {
        const children = node.children?.length ? walk(node.children) : []
        if (children.length) out.push({ ...node, children })
      }
    }
    return out
  }
  return { tree: walk(treeData.value), count }
})
const displayTree = computed(() => (searchActive.value ? searchResult.value.tree : treeData.value))
const emptyText = computed(() => (searchActive.value ? `未找到匹配「${keyword.value.trim()}」的菜单` : '暂无数据'))

/** 关键词高亮: 先转义再包 <mark>, 词与原文都按转义后形态匹配避免破坏标签 */
const escapeHtml = (s: string) =>
  s.replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', '\'': '&#39;' })[c] || c)
const escapeRe = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
function highlight(text: string) {
  const safe = escapeHtml(text || '')
  if (!searchActive.value) return safe
  const re = new RegExp(`(${terms.value.map(t => escapeRe(escapeHtml(t))).join('|')})`, 'gi')
  return safe.replace(re, '<mark class="menu-hit">$1</mark>')
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
      <div class="toolbar">
        <div>
          <el-button v-permission="'system:menu:create'" type="success" :icon="Plus" @click="openCreate()">新增根菜单</el-button>
          <el-button :icon="Refresh" @click="loadTree">刷新</el-button>
        </div>
        <div class="toolbar-search">
          <el-tag v-if="searchActive" type="info" size="small" effect="plain">命中 {{ searchResult.count }} 项</el-tag>
          <el-input
            v-model="keyword"
            :prefix-icon="Search"
            placeholder="搜索名称/路径/权限标识"
            clearable
            style="width:260px"
          />
        </div>
      </div>
      <el-table
        v-loading="loading"
        :data="displayTree"
        row-key="id"
        border
        stripe
        :tree-props="{ children: 'children' }"
        default-expand-all
        :empty-text="emptyText"
      >
        <el-table-column label="名称" min-width="180">
          <template #default="{ row }">
            <span v-html="highlight(row.name)" />
          </template>
        </el-table-column>
        <el-table-column label="类型" width="80">
          <template #default="{ row }">
            <el-tag :type="typeTag(row.type)" size="small">{{ typeText(row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="路径" min-width="160">
          <template #default="{ row }">
            <span v-html="highlight(row.path)" />
          </template>
        </el-table-column>
        <el-table-column label="权限标识" min-width="180">
          <template #default="{ row }">
            <span v-html="highlight(row.permission)" />
          </template>
        </el-table-column>
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
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px 12px;
  margin-bottom: 8px;
}
.toolbar-search {
  display: flex;
  align-items: center;
  gap: 8px;
}
:deep(mark.menu-hit) {
  padding: 0 2px;
  border-radius: 2px;
  background-color: var(--el-color-warning-light-8);
  color: var(--el-color-warning-dark-2);
}
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
  background-color: var(--el-color-primary-light-9);
}
.icon-picker-item.active {
  background-color: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}
.icon-picker-label {
  margin-top: 4px;
  font-size: 10px;
  color: var(--el-text-color-regular);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
  text-align: center;
}
.icon-picker-item.active .icon-picker-label {
  color: var(--el-color-primary);
}
</style>
