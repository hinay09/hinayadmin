<script setup lang="ts">
/**
 * CMS 栏目管理: 栏目 = 文章的分类 (扁平, 无层级)。
 * code 租户内唯一且创建后不可改 (语义标识, 供外部系统引用)。
 */
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { Search, Plus, Edit, Delete } from '@element-plus/icons-vue'
import { useCmsCategoryApi, type CmsCategoryItem } from '~/composables/useApi'

definePageMeta({ title: '栏目管理' })
defineOptions({ name: 'cms-category' })

const api = useCmsCategoryApi()

// ============================================================
// 列表
// ============================================================
const loading = ref(false)
const list = ref<CmsCategoryItem[]>([])
const total = ref(0)
const query = reactive({ keyword: '', status: undefined as number | undefined, page: 1, pageSize: 10 })

async function load() {
  loading.value = true
  try {
    const res = await api.list({ ...query })
    list.value = res.list || []
    total.value = res.total || 0
  }
  finally { loading.value = false }
}

// ============================================================
// 新增/编辑 (抽屉)
// ============================================================
const drawerVisible = ref(false)
const drawerTitle = ref('新增栏目')
const isEdit = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()
const form = reactive({ id: 0, name: '', code: '', sort: 0, status: 1, remark: '' })
const rules = {
  name: [{ required: true, message: '请输入栏目名称', trigger: 'blur' }],
  code: [
    { required: true, message: '请输入栏目编码', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_-]{1,64}$/, message: '1-64 位字母/数字/下划线/连字符', trigger: 'blur' },
  ],
}

function resetForm() {
  Object.assign(form, { id: 0, name: '', code: '', sort: 0, status: 1, remark: '' })
  formRef.value?.clearValidate()
}

function openCreate() {
  resetForm()
  isEdit.value = false
  drawerTitle.value = '新增栏目'
  drawerVisible.value = true
}

function openEdit(row: CmsCategoryItem) {
  resetForm()
  Object.assign(form, {
    id: row.id, name: row.name, code: row.code, sort: row.sort, status: row.status, remark: row.remark,
  })
  isEdit.value = true
  drawerTitle.value = '编辑栏目'
  drawerVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    if (isEdit.value) {
      // code 不可改: 不传给后端
      await api.update(form.id, { name: form.name, sort: form.sort, status: form.status, remark: form.remark })
      ElMessage.success('更新成功')
    }
    else {
      await api.create({ name: form.name, code: form.code, sort: form.sort, status: form.status, remark: form.remark })
      ElMessage.success('新增成功')
    }
    drawerVisible.value = false
    load()
  }
  catch {} finally { submitting.value = false }
}

async function handleDelete(row: CmsCategoryItem) {
  await ElMessageBox.confirm(`删除栏目「${row.name}」? 栏目下还有文章时将无法删除。`, '删除确认', { type: 'warning' })
  try {
    await api.remove(row.id)
    ElMessage.success('已删除')
    load()
  }
  catch {}
}

onMounted(load)
</script>

<template>
  <div class="page">
    <el-card>
      <el-form inline @submit.prevent>
        <el-form-item label="搜索">
          <el-input v-model="query.keyword" placeholder="名称/编码" clearable
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
          <el-button v-permission="'cms:category:create'" type="success" :icon="Plus" @click="openCreate">新增栏目</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="list" border stripe style="margin-top:8px">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="栏目名称" width="180" />
        <el-table-column prop="code" label="栏目编码" width="180" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="70" />
        <el-table-column prop="remark" label="备注" min-width="160" show-overflow-tooltip />
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button v-permission="'cms:category:update'" link type="primary" :icon="Edit" @click="openEdit(row)">编辑</el-button>
            <el-button v-permission="'cms:category:delete'" link type="danger" :icon="Delete" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination v-model:current-page="query.page" v-model:page-size="query.pageSize"
        :total="total" :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next, jumper"
        style="margin-top:16px;justify-content:flex-end"
        @current-change="load" @size-change="load" />
    </el-card>

    <!-- 栏目编辑 -->
    <el-drawer v-model="drawerVisible" :title="drawerTitle" size="440px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="栏目名称" prop="name">
          <el-input v-model="form.name" maxlength="64" placeholder="如 公司新闻 / 产品动态" />
        </el-form-item>
        <el-form-item label="栏目编码" prop="code">
          <el-input v-model="form.code" maxlength="64" placeholder="如 news / product"
            :disabled="isEdit" />
          <div v-if="isEdit" class="form-tip">编码创建后不可修改</div>
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
  </div>
</template>

<style scoped>
.form-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
  margin-top: 2px;
}
</style>
