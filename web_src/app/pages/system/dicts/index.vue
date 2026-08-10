<script setup lang="ts">
/**
 * 字典管理 - 主从表结构
 * 主界面: 字典类型列表
 * 设置弹窗: 字典数据项管理 (支持上下移动排序)
 */
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { Search, Plus, Edit, Delete, Setting, Top, Bottom } from '@element-plus/icons-vue'
import { useDictTypeApi, useDictApi } from '~/composables/useApi'

definePageMeta({ title: '字典管理' })

const typeApi = useDictTypeApi()
const itemApi = useDictApi()

// ============================================================
// 字典类型列表
// ============================================================
const loading = ref(false)
const typeList = ref<any[]>([])
const typeTotal = ref(0)
const typeQuery = reactive({ keyword: '', page: 1, pageSize: 10 })

const drawerVisible = ref(false)
const drawerTitle = ref('新增字典类型')
const typeFormRef = ref<FormInstance>()
const typeForm = reactive({ id: 0, typeCode: '', typeName: '', status: 1, remark: '' })
const isTypeEdit = ref(false)
const submitting = ref(false)

const typeRules = computed(() => ({
  typeCode: [{ required: true, message: '请输入类型编码', trigger: 'blur' }],
  typeName: [{ required: true, message: '请输入类型名称', trigger: 'blur' }],
}))

async function loadTypes() {
  loading.value = true
  try {
    const res = await typeApi.list({ ...typeQuery })
    typeList.value = res.list || []
    typeTotal.value = res.total || 0
  } finally { loading.value = false }
}

function resetTypeForm() {
  Object.assign(typeForm, { id: 0, typeCode: '', typeName: '', status: 1, remark: '' })
}

function openCreateType() {
  resetTypeForm()
  drawerTitle.value = '新增字典类型'
  isTypeEdit.value = false
  drawerVisible.value = true
}

function openEditType(row: any) {
  resetTypeForm()
  drawerTitle.value = '编辑字典类型'
  isTypeEdit.value = true
  Object.assign(typeForm, {
    id: row.id, typeCode: row.typeCode, typeName: row.typeName,
    status: row.status, remark: row.remark,
  })
  drawerVisible.value = true
}

async function handleTypeSubmit() {
  if (!typeFormRef.value) return
  const valid = await typeFormRef.value.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    if (isTypeEdit.value) {
      await typeApi.update(typeForm.id, {
        typeCode: typeForm.typeCode, typeName: typeForm.typeName,
        status: typeForm.status, remark: typeForm.remark,
      })
      ElMessage.success('更新成功')
    } else {
      await typeApi.create({
        typeCode: typeForm.typeCode, typeName: typeForm.typeName,
        status: typeForm.status, remark: typeForm.remark,
      })
      ElMessage.success('新增成功')
    }
    drawerVisible.value = false
    loadTypes()
  } catch {} finally { submitting.value = false }
}

async function handleTypeDelete(row: any) {
  await ElMessageBox.confirm(`确认删除字典类型「${row.typeName}」?`, '提示', { type: 'warning' })
  try {
    await typeApi.remove(row.id)
    ElMessage.success('删除成功')
    loadTypes()
  } catch {}
}

// ============================================================
// 字典数据项管理 (设置弹窗)
// ============================================================
const itemsDialogVisible = ref(false)
const currentType = ref<any>(null)
const itemsLoading = ref(false)
const itemList = ref<any[]>([])
const itemTotal = ref(0)

// 数据项编辑
const itemDrawerVisible = ref(false)
const itemDrawerTitle = ref('新增字典项')
const itemFormRef = ref<FormInstance>()
const itemForm = reactive({ id: 0, dictLabel: '', dictValue: '', sort: 0, status: 1, remark: '' })
const isItemEdit = ref(false)
const itemSubmitting = ref(false)

const itemRules = computed(() => ({
  dictLabel: [{ required: true, message: '请输入字典标签', trigger: 'blur' }],
}))

async function openItemsDialog(row: any) {
  currentType.value = row
  itemsDialogVisible.value = true
  await loadItems()
}

async function loadItems() {
  if (!currentType.value) return
  itemsLoading.value = true
  try {
    const res = await itemApi.listByType(currentType.value.id, { pageSize: 200 })
    itemList.value = res.list || []
    itemTotal.value = res.total || 0
  } finally { itemsLoading.value = false }
}

function resetItemForm() {
  const maxSort = itemList.value.reduce((m, i) => Math.max(m, i.sort || 0), 0)
  Object.assign(itemForm, { id: 0, dictLabel: '', dictValue: '', sort: maxSort + 1, status: 1, remark: '' })
}

function openCreateItem() {
  resetItemForm()
  itemDrawerTitle.value = '新增字典项'
  isItemEdit.value = false
  itemDrawerVisible.value = true
}

function openEditItem(row: any) {
  itemDrawerTitle.value = '编辑字典项'
  isItemEdit.value = true
  Object.assign(itemForm, {
    id: row.id, dictLabel: row.dictLabel, dictValue: row.dictValue,
    sort: row.sort, status: row.status, remark: row.remark,
  })
  itemDrawerVisible.value = true
}

async function handleItemSubmit() {
  if (!itemFormRef.value || !currentType.value) return
  const valid = await itemFormRef.value.validate().catch(() => false)
  if (!valid) return
  itemSubmitting.value = true
  try {
    const typeId = currentType.value.id
    if (isItemEdit.value) {
      await itemApi.update(typeId, itemForm.id, {
        dictLabel: itemForm.dictLabel, dictValue: itemForm.dictValue,
        sort: itemForm.sort, status: itemForm.status, remark: itemForm.remark,
      })
      ElMessage.success('更新成功')
    } else {
      await itemApi.create(typeId, {
        dictLabel: itemForm.dictLabel, dictValue: itemForm.dictValue,
        sort: itemForm.sort, status: itemForm.status, remark: itemForm.remark,
      })
      ElMessage.success('新增成功')
    }
    itemDrawerVisible.value = false
    loadItems()
  } catch {} finally { itemSubmitting.value = false }
}

async function handleItemDelete(row: any) {
  if (!currentType.value) return
  await ElMessageBox.confirm(`确认删除字典项「${row.dictLabel}」?`, '提示', { type: 'warning' })
  try {
    await itemApi.remove(currentType.value.id, row.id)
    ElMessage.success('删除成功')
    loadItems()
  } catch {}
}

// 上下移动排序
async function moveItem(index: number, direction: 'up' | 'down') {
  const targetIdx = direction === 'up' ? index - 1 : index + 1
  if (targetIdx < 0 || targetIdx >= itemList.value.length) return
  // 交换 sort 值
  const a = itemList.value[index]
  const b = itemList.value[targetIdx]
  const sortA = a.sort ?? index
  const sortB = b.sort ?? targetIdx
  try {
    if (!currentType.value) return
    await itemApi.sort(currentType.value.id, [
      { id: a.id, sort: sortB },
      { id: b.id, sort: sortA },
    ])
    loadItems()
  } catch {}
}

onMounted(() => { loadTypes() })
</script>

<template>
  <div class="page">
    <!-- 字典类型列表 -->
    <el-card>
      <el-form inline @submit.prevent>
        <el-form-item label="搜索">
          <el-input v-model="typeQuery.keyword" placeholder="编码/名称" clearable
            @keyup.enter="() => { typeQuery.page = 1; loadTypes() }" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="() => { typeQuery.page = 1; loadTypes() }">查询</el-button>
          <el-button v-permission="'system:dict:create'" type="success" :icon="Plus" @click="openCreateType">新增类型</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="typeList" border stripe style="margin-top:8px">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="typeCode" label="类型编码" width="200" />
        <el-table-column prop="typeName" label="类型名称" width="200" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="160" show-overflow-tooltip />
        <el-table-column prop="createdAt" label="创建时间" width="180" />
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button v-permission="'system:dict:update'" link type="primary" :icon="Setting"
              @click="openItemsDialog(row)">设置</el-button>
            <el-button v-permission="'system:dict:update'" link type="primary" :icon="Edit"
              @click="openEditType(row)">编辑</el-button>
            <el-button v-permission="'system:dict:delete'" link type="danger" :icon="Delete"
              @click="handleTypeDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination v-model:current-page="typeQuery.page" v-model:page-size="typeQuery.pageSize"
        :total="typeTotal" :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next, jumper"
        style="margin-top:16px;justify-content:flex-end"
        @current-change="loadTypes" @size-change="loadTypes" />
    </el-card>

    <!-- 字典类型 新增/编辑 Drawer -->
    <el-drawer v-model="drawerVisible" :title="drawerTitle" size="480px">
      <template #header>
        <div style="display:flex;align-items:center;gap:6px;font-weight:600">
          <span>{{ drawerTitle }}</span>
        </div>
      </template>
      <el-form ref="typeFormRef" :model="typeForm" :rules="typeRules" label-width="90px">
        <el-form-item label="类型编码" prop="typeCode">
          <el-input v-model="typeForm.typeCode" placeholder="如 sys_status" />
        </el-form-item>
        <el-form-item label="类型名称" prop="typeName">
          <el-input v-model="typeForm.typeName" placeholder="如 系统状态" />
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="typeForm.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="typeForm.remark" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="drawerVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleTypeSubmit">确定</el-button>
      </template>
    </el-drawer>

    <!-- 字典数据项管理 Dialog -->
    <el-dialog v-model="itemsDialogVisible"
      :title="`字典项管理 - ${currentType?.typeName || ''}`"
      width="800px" top="5vh" destroy-on-close>
      <div style="margin-bottom:12px;display:flex;justify-content:flex-end">
        <el-button v-permission="'system:dict:create'" type="primary" :icon="Plus" size="small"
          @click="openCreateItem">新增字典项</el-button>
      </div>
      <el-table v-loading="itemsLoading" :data="itemList" border stripe size="small">
        <el-table-column prop="sort" label="排序" width="70" align="center" />
        <el-table-column prop="dictLabel" label="字典标签" min-width="140" />
        <el-table-column prop="dictValue" label="字典值" min-width="140" show-overflow-tooltip />
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="120" show-overflow-tooltip />
        <el-table-column label="操作" width="160" fixed="right" align="center">
          <template #default="{ row, $index }">
            <el-button link type="info" :icon="Top" :disabled="$index === 0"
              @click="moveItem($index, 'up')" />
            <el-button link type="info" :icon="Bottom" :disabled="$index === itemList.length - 1"
              @click="moveItem($index, 'down')" />
            <el-button v-permission="'system:dict:update'" link type="primary" :icon="Edit"
              @click="openEditItem(row)" />
            <el-button v-permission="'system:dict:delete'" link type="danger" :icon="Delete"
              @click="handleItemDelete(row)" />
          </template>
        </el-table-column>
      </el-table>
      <div style="margin-top:8px;color:#999;font-size:12px">共 {{ itemTotal }} 条数据项</div>
    </el-dialog>

    <!-- 字典数据项 新增/编辑 Drawer -->
    <el-drawer v-model="itemDrawerVisible" :title="itemDrawerTitle" size="420px" append-to-body>
      <el-form ref="itemFormRef" :model="itemForm" :rules="itemRules" label-width="80px">
        <el-form-item label="字典标签" prop="dictLabel">
          <el-input v-model="itemForm.dictLabel" placeholder="展示名称" />
        </el-form-item>
        <el-form-item label="字典值" prop="dictValue">
          <el-input v-model="itemForm.dictValue" placeholder="存储值" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="itemForm.sort" :min="0" />
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="itemForm.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="itemForm.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="itemDrawerVisible = false">取消</el-button>
        <el-button type="primary" :loading="itemSubmitting" @click="handleItemSubmit">确定</el-button>
      </template>
    </el-drawer>
  </div>
</template>
