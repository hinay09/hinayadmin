<script setup lang="ts">
/**
 * 租户套餐 (平台超管专属) —— docs/multi-tenant.md §7/§7.2。
 *
 * 应用授权集的模板复用层: 套餐 = 目录/页面级菜单勾选的命名快照,
 * 建同类租户时在创建向导选中预填 (免逐个勾选, 仍可微调);
 * 修改套餐不自动影响已绑定租户 —— 列表点「同步」显式物化
 * (走应用授权集同一路径: 重写 sys_tenant_menu + casbin 双向联动)。
 */
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { Plus, Edit, Delete, Refresh, Search } from '@element-plus/icons-vue'
import { useTenantPackageApi, type TenantPackageItem } from '~/composables/useApi'
import { useMenuApi } from '~/composables/useApi'
import { useUserStore } from '~/stores/user'
import { useConfigStore } from '~/stores/config'

definePageMeta({ title: '租户套餐' })
defineOptions({ name: 'system-tenant-packages' })

const pkgApi = useTenantPackageApi()
const menuApi = useMenuApi()
const userStore = useUserStore()
const configStore = useConfigStore()
const visible = computed(() => userStore.isPlatformAdmin && configStore.multiTenant)

const loading = ref(false)
const list = ref<TenantPackageItem[]>([])
const total = ref(0)
const query = reactive({ keyword: '', status: undefined as number | undefined, page: 1, pageSize: 10 })

async function loadList() {
  loading.value = true
  try {
    const res: any = await pkgApi.list(query)
    list.value = res?.list || []
    total.value = res?.total || 0
  } finally {
    loading.value = false
  }
}

onMounted(() => { if (visible.value) loadList() })

/* ---- 新增/编辑对话框 (共用菜单树) ---- */
const editVisible = ref(false)
const editing = ref(false)
const editRef = ref<FormInstance>()
const editForm = reactive({ id: 0, name: '', status: 1, remark: '', menuIds: [] as number[] })
const editRules = {
  name: [{ required: true, message: '请输入套餐名称', trigger: 'blur' }, { min: 2, max: 64, message: '长度 2-64', trigger: 'blur' }],
}
const saving = ref(false)

const menuTreeData = ref<any[]>([])
async function loadMenuTree() {
  if (menuTreeData.value.length) return
  const res: any = await menuApi.tree()
  menuTreeData.value = res?.tree || []
}
const treeRef = ref<any>()

async function openCreate() {
  Object.assign(editForm, { id: 0, name: '', status: 1, remark: '', menuIds: [] })
  editing.value = false
  editVisible.value = true
  await loadMenuTree()
  await nextTick()
  treeRef.value?.setCheckedKeys([], false)
}

async function openEdit(row: TenantPackageItem) {
  editing.value = true
  editVisible.value = true
  await loadMenuTree()
  const res: any = await pkgApi.detail(row.id)
  const pkg = res?.package || {}
  Object.assign(editForm, {
    id: pkg.id, name: pkg.name, status: pkg.status, remark: pkg.remark || '',
    menuIds: (pkg.menuIds || []).map(Number),
  })
  await nextTick()
  treeRef.value?.setCheckedKeys([], false)
  treeRef.value?.setCheckedKeys(editForm.menuIds, false)
}

async function submitEdit() {
  await editRef.value?.validate()
  // 勾选目录/页面级 (check-strictly 由父节点状态自行管理, 半选父节点随子生效)
  editForm.menuIds = treeRef.value.getCheckedKeys(false)
  if (!editForm.menuIds.length) {
    ElMessage.warning('请至少勾选一个应用模块')
    return
  }
  saving.value = true
  try {
    const data = { name: editForm.name, menuIds: editForm.menuIds, status: editForm.status, remark: editForm.remark }
    if (editing.value) {
      await pkgApi.update(editForm.id, data)
      ElMessage.success('套餐已保存; 绑定租户需点「同步」才会应用新授权集')
    } else {
      await pkgApi.create(data)
      ElMessage.success('套餐已创建, 创建租户时可在向导第三步选用')
    }
    editVisible.value = false
    loadList()
  } finally {
    saving.value = false
  }
}

async function handleDelete(row: TenantPackageItem) {
  await ElMessageBox.confirm(
    `确定删除套餐「${row.name}」? ${row.tenantCount > 0 ? '该套餐仍有租户绑定, 需先解绑。' : '软删保留数据, 套餐名将被改写防重建撞名。'}`,
    '删除套餐', { type: 'warning' },
  )
  await pkgApi.remove(row.id)
  ElMessage.success('已删除')
  loadList()
}

async function handleSync(row: TenantPackageItem) {
  await ElMessageBox.confirm(
    `将「${row.name}」的授权集同步到 ${row.tenantCount} 个绑定租户? 收缩即时清理越权策略, 扩张自动授予各租户管理员角色。`,
    '同步套餐', { type: 'warning' },
  )
  const res = await pkgApi.sync(row.id)
  ElMessage.success(`同步完成, 已更新 ${res?.count ?? 0} 个租户`)
  loadList()
}

function statusText(s: number) { return s === 1 ? '启用' : '停用' }
</script>

<template>
  <div v-if="visible" class="page">
    <el-card>
      <el-form inline @submit.prevent>
        <el-form-item label="关键词">
          <el-input v-model="query.keyword" placeholder="套餐名称" clearable :prefix-icon="Search" style="width: 200px" @keyup.enter="loadList" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="query.status" placeholder="全部" clearable style="width: 110px">
            <el-option label="启用" :value="1" />
            <el-option label="停用" :value="0" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="loadList">查询</el-button>
          <el-button type="success" :icon="Plus" @click="openCreate">新增套餐</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="list" border stripe>
        <el-table-column prop="id" label="ID" width="64" />
        <el-table-column prop="name" label="套餐名称" min-width="140" />
        <el-table-column label="应用模块" width="90">
          <template #default="{ row }">{{ row.menuCount }} 个</template>
        </el-table-column>
        <el-table-column label="绑定租户" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="row.tenantCount > 0 ? 'primary' : 'info'" effect="plain">{{ row.tenantCount }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="160" show-overflow-tooltip />
        <el-table-column label="操作" width="230" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" :icon="Edit" @click="openEdit(row)">编辑</el-button>
            <el-button link type="warning" :icon="Refresh" :disabled="row.tenantCount === 0" @click="handleSync(row)">同步</el-button>
            <el-button link type="danger" :icon="Delete" :disabled="row.tenantCount > 0" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="query.page"
        v-model:page-size="query.pageSize"
        :page-sizes="[10, 20, 50]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        style="margin-top: 14px; justify-content: flex-end"
        @current-change="loadList"
        @size-change="loadList"
      />
    </el-card>

    <!-- 新增/编辑对话框 -->
    <el-dialog v-model="editVisible" :title="editing ? '编辑套餐' : '新增套餐'" width="600px" :close-on-click-modal="false">
      <el-alert type="info" :closable="false" show-icon style="margin-bottom: 14px"
        title="勾选套餐包含的应用模块 (目录/页面级, 按钮随父页面闭包); 平台专属模块 (监控/工具/租户管理/租户套餐) 不可选" />
      <el-form ref="editRef" :model="editForm" :rules="editRules" label-width="90px">
        <el-form-item label="套餐名称" prop="name"><el-input v-model="editForm.name" placeholder="如 标准版 / 专业版" /></el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="editForm.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">停用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="editForm.remark" type="textarea" :rows="2" /></el-form-item>
        <el-form-item label="应用模块">
          <el-tree
            ref="treeRef"
            :data="menuTreeData"
            show-checkbox node-key="id" default-expand-all check-strictly
            :props="{ label: 'name', children: 'children' }"
            style="width: 100%; max-height: 320px; overflow: auto; border: 1px solid var(--el-border-color-lighter); border-radius: 6px; padding: 8px"
            @check="(_n: any, cs: any) => (editForm.menuIds = cs.checkedKeys)"
          >
            <template #default="{ data }">
              <span style="display: flex; align-items: center; gap: 6px">
                <span>{{ data.name }}</span>
                <el-tag v-if="data.type === 1" size="small" type="info">目录</el-tag>
                <el-tag v-else-if="data.type === 3" size="small" type="warning" effect="plain">按钮</el-tag>
              </span>
            </template>
          </el-tree>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitEdit">保存</el-button>
      </template>
    </el-dialog>
  </div>
  <el-empty v-else description="多租户功能未启用或无访问权限" />
</template>

<style scoped>
.page { display: flex; flex-direction: column; gap: 12px; }
</style>
