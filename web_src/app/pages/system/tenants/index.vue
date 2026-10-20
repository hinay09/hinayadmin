<script setup lang="ts">
/**
 * 租户管理 (平台超管专属) —— docs/multi-tenant.md §7/§7.1/§8.2。
 *
 * - 列表: 全租户混排, 禁用/到期状态可视;
 * - 创建向导 (三步): 基本信息 → 管理员账号 → 应用授权预览;
 * - 应用授权抽屉: 菜单树勾选 (复用角色管理 el-tree 模式), PUT 全量覆盖,
 *   后端双向联动 casbin (收缩清策略/扩张给 tenant_admin 追加);
 * - 平台干预: 重置租户覆盖配置; 删除租户 (软删, code 原子改写防撞 uk)。
 */
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { Plus, Edit, Delete, Key, RefreshLeft, Search } from '@element-plus/icons-vue'
import { useTenantApi, useTenantPackageApi, type TenantItem, type TenantPackageLite } from '~/composables/useApi'
import { useMenuApi } from '~/composables/useApi'
import { useUserStore } from '~/stores/user'
import { useConfigStore } from '~/stores/config'

definePageMeta({ title: '租户管理' })
defineOptions({ name: 'system-tenants' })

const tenantApi = useTenantApi()
const pkgApi = useTenantPackageApi()
const menuApi = useMenuApi()
const userStore = useUserStore()
const configStore = useConfigStore()
const visible = computed(() => userStore.isPlatformAdmin && configStore.multiTenant)

const loading = ref(false)
const list = ref<TenantItem[]>([])
const total = ref(0)
const query = reactive({ keyword: '', status: undefined as number | undefined, page: 1, pageSize: 10 })

async function loadList() {
  loading.value = true
  try {
    const res: any = await tenantApi.list(query)
    list.value = res?.list || []
    total.value = res?.total || 0
  } finally {
    loading.value = false
  }
}

/* ---- 启用套餐下拉 (向导预填/编辑绑定/授权导入共用) ---- */
const packages = ref<TenantPackageLite[]>([])
async function loadPackages() {
  const res: any = await pkgApi.listAll()
  packages.value = (res?.list || []).map((p: any) => ({ ...p, menuIds: (p.menuIds || []).map(Number) }))
}

onMounted(() => { if (visible.value) { loadList(); loadPackages() } })

/* ---- 编辑抽屉 ---- */
const editRef = ref<FormInstance>()
const editVisible = ref(false)
const editForm = reactive({ id: 0, name: '', contact: '', plan: 'standard', packageId: 0, expireAt: '', status: 1, remark: '', code: '' })

function openEdit(row: TenantItem) {
  Object.assign(editForm, {
    id: row.id, name: row.name, contact: row.contact, plan: row.plan || 'standard',
    packageId: row.packageId || 0,
    expireAt: row.expireAt ? String(row.expireAt).replace('T', ' ').slice(0, 16) : '',
    status: row.status, remark: row.remark || '', code: row.code,
  })
  if (!packages.value.length) loadPackages()
  editVisible.value = true
}

async function submitEdit() {
  await tenantApi.update(editForm.id, {
    name: editForm.name, contact: editForm.contact, plan: editForm.plan,
    packageId: editForm.packageId || 0,
    expireAt: editForm.expireAt, status: editForm.status, remark: editForm.remark,
  })
  ElMessage.success('已保存 (套餐绑定变更不自动调整授权集, 需生效请在租户套餐页同步)')
  editVisible.value = false
  loadList()
}

async function handleDelete(row: TenantItem) {
  await ElMessageBox.confirm(
    `确定删除租户「${row.name}(${row.code})」? 软删保留数据, 租户码将被改写防重建撞号, 该操作不可恢复。`,
    '删除租户', { type: 'warning' },
  )
  await tenantApi.remove(row.id)
  ElMessage.success('已删除')
  loadList()
}

async function handleResetConfigs(row: TenantItem) {
  await ElMessageBox.confirm(
    `重置「${row.name}」的全部覆盖配置? 该租户的站点名/Logo/版权/密码策略等将回归平台默认值。`,
    '重置租户配置', { type: 'warning' },
  )
  await tenantApi.resetConfigs(row.id)
  ElMessage.success('已重置')
}

/* ---- 创建向导 (三步) ---- */
const wizardVisible = ref(false)
const wizardStep = ref(0)
const wizardRef = ref<FormInstance>()
const wizardTreeRef = ref<any>()
const wizard = reactive({
  name: '', code: '', contact: '', plan: 'standard', packageId: undefined as number | undefined,
  expireAt: '', remark: '',
  adminUsername: '', adminNickname: '', adminPassword: '',
  menuIds: [] as number[],
})
const wizardRules = {
  name: [{ required: true, message: '请输入租户名称', trigger: 'blur' }, { min: 2, max: 64, message: '长度 2-64', trigger: 'blur' }],
  code: [
    { required: true, message: '请输入租户标识', trigger: 'blur' },
    { pattern: /^[a-z][a-z0-9-]*$/, message: '小写字母开头, 仅小写字母/数字/连字符', trigger: 'blur' },
  ],
  adminUsername: [{ required: true, message: '请输入管理员账号', trigger: 'blur' }, { min: 2, max: 32, message: '长度 2-32', trigger: 'blur' }],
}
const creating = ref(false)

// 应用授权树数据 (创建向导第三步 + 授权抽屉共用)
const menuTreeData = ref<any[]>([])
async function loadMenuTree() {
  if (menuTreeData.value.length) return
  const res: any = await menuApi.tree()
  menuTreeData.value = res?.tree || []
}
const treeProps = { label: 'name', children: 'children' }

function openWizard() {
  Object.assign(wizard, {
    name: '', code: '', contact: '', plan: 'standard', packageId: undefined,
    expireAt: '', remark: '',
    adminUsername: '', adminNickname: '', adminPassword: '', menuIds: [],
  })
  wizardStep.value = 0
  wizardVisible.value = true
  loadMenuTree()
  if (!packages.value.length) loadPackages()
}

// 选中套餐 → 预填勾选树与套餐展示标签 (仍可在树上继续微调)
function applyPackageToWizard(pkgId?: number) {
  const pkg = packages.value.find(p => p.id === pkgId)
  if (!pkg) return
  wizard.menuIds = [...pkg.menuIds]
  wizard.plan = pkg.name
  wizardTreeRef.value?.setCheckedKeys([], false)
  wizardTreeRef.value?.setCheckedKeys(pkg.menuIds, false)
}

async function nextStep() {
  if (wizardStep.value === 0) {
    await wizardRef.value?.validate()
  }
  if (wizardStep.value === 1 && !wizard.adminPassword && wizard.adminPassword !== '') {
    // 密码可空 (默认 123456 + 强制改密)
  }
  wizardStep.value++
  if (wizardStep.value === 2) await nextTick()
}

async function submitWizard() {
  creating.value = true
  try {
    const res = await tenantApi.create({
      name: wizard.name, code: wizard.code, contact: wizard.contact, plan: wizard.plan,
      packageId: wizard.packageId || undefined,
      expireAt: wizard.expireAt, remark: wizard.remark,
      adminUsername: wizard.adminUsername, adminNickname: wizard.adminNickname || undefined,
      adminPassword: wizard.adminPassword || undefined,
      menuIds: wizard.menuIds,
    })
    ElMessage.success(`租户创建成功 (id=${res.id}), 管理员初始密码${wizard.adminPassword ? '已设置' : '为 123456'}, 首次登录强制改密`)
    wizardVisible.value = false
    loadList()
  } finally {
    creating.value = false
  }
}

/* ---- 应用授权抽屉 (§7.1) ---- */
const authVisible = ref(false)
const authTreeRef = ref<any>()
const authTenant = ref<TenantItem | null>(null)
const authSaving = ref(false)
const authImportPkg = ref<number | undefined>(undefined)

async function openAuth(row: TenantItem) {
  authTenant.value = row
  authImportPkg.value = undefined
  authVisible.value = true
  await loadMenuTree()
  if (!packages.value.length) loadPackages()
  const [res] = await Promise.all([tenantApi.getMenus(row.id)])
  await nextTick()
  authTreeRef.value?.setCheckedKeys([], false)
  authTreeRef.value?.setCheckedKeys((res?.menuIds || []).map(Number), false)
}

// 从套餐导入授权集到勾选树 (预填, 仍需手动保存)
function importPackageToAuth(pkgId?: number) {
  const pkg = packages.value.find(p => p.id === pkgId)
  if (!pkg) return
  authTreeRef.value?.setCheckedKeys([], false)
  authTreeRef.value?.setCheckedKeys(pkg.menuIds, false)
}

async function submitAuth() {
  if (!authTenant.value) return
  authSaving.value = true
  try {
    // 勾选目录/页面级 (check-strictly 由父节点状态自行管理, 半选父节点随子生效)
    const menuIds: number[] = authTreeRef.value.getCheckedKeys(false)
    await tenantApi.setMenus(authTenant.value.id, menuIds)
    ElMessage.success('应用授权已更新 (策略双向联动即时生效)')
    authVisible.value = false
  } finally {
    authSaving.value = false
  }
}

function statusText(s: number) { return s === 1 ? '启用' : '禁用' }
</script>

<template>
  <div v-if="visible" class="page">
    <el-card>
      <el-form inline @submit.prevent>
        <el-form-item label="关键词">
          <el-input v-model="query.keyword" placeholder="名称/标识/联系人" clearable :prefix-icon="Search" style="width: 200px" @keyup.enter="loadList" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="query.status" placeholder="全部" clearable style="width: 110px">
            <el-option label="启用" :value="1" />
            <el-option label="禁用" :value="0" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="loadList">查询</el-button>
          <el-button type="success" :icon="Plus" @click="openWizard">创建租户</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="list" border stripe>
        <el-table-column prop="id" label="ID" width="64" />
        <el-table-column prop="name" label="租户名称" min-width="130" />
        <el-table-column prop="code" label="租户标识" min-width="110">
          <template #default="{ row }"><el-tag size="small" type="info">{{ row.code }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="contact" label="联系人" min-width="90" />
        <el-table-column label="套餐" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.packageName" size="small">{{ row.packageName }}</el-tag>
            <span v-else>{{ row.plan }}</span>
          </template>
        </el-table-column>
        <el-table-column label="到期时间" width="160">
          <template #default="{ row }">
            <span v-if="row.expireAt">{{ String(row.expireAt).replace('T', ' ').slice(0, 16) }}</span>
            <el-tag v-else size="small" type="success">永久</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" :icon="Edit" @click="openEdit(row)">编辑</el-button>
            <el-button link type="warning" :icon="Key" @click="openAuth(row)">应用授权</el-button>
            <el-button link type="info" :icon="RefreshLeft" @click="handleResetConfigs(row)">重置配置</el-button>
            <el-button link type="danger" :icon="Delete" :disabled="row.id === 1" @click="handleDelete(row)">删除</el-button>
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

    <!-- 编辑抽屉 -->
    <el-drawer v-model="editVisible" title="编辑租户" size="480px">
      <el-form :model="editForm" label-width="90px">
        <el-form-item label="租户标识">
          <el-input :model-value="editForm.code" disabled>
            <template #suffix><el-tooltip content="租户码创建后不可变 (登录入口/命名空间依据)"><el-icon><InfoFilled /></el-icon></el-tooltip></template>
          </el-input>
        </el-form-item>
        <el-form-item label="租户名称" required>
          <el-input v-model="editForm.name" />
        </el-form-item>
        <el-form-item label="联系人"><el-input v-model="editForm.contact" /></el-form-item>
        <el-form-item label="套餐"><el-input v-model="editForm.plan" /></el-form-item>
        <el-form-item label="套餐绑定">
          <el-select v-model="editForm.packageId" placeholder="未绑定/自定义" clearable style="width: 100%">
            <el-option v-for="p in packages" :key="p.id" :label="`${p.name} (${p.menuIds.length} 个应用)`" :value="p.id" />
          </el-select>
          <div class="form-tip">仅记录归属; 调整授权集请用「应用授权」抽屉或在租户套餐页同步</div>
        </el-form-item>
        <el-form-item label="到期时间">
          <el-date-picker v-model="editForm.expireAt" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" placeholder="留空=永久" style="width: 100%" />
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="editForm.status" :disabled="editForm.id === 1">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="editForm.remark" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" @click="submitEdit">保存</el-button>
      </template>
    </el-drawer>

    <!-- 创建向导 (三步) -->
    <el-dialog v-model="wizardVisible" title="创建租户" width="640px" :close-on-click-modal="false">
      <el-steps :active="wizardStep" finish-status="success" simple style="margin-bottom: 18px">
        <el-step title="基本信息" />
        <el-step title="管理员账号" />
        <el-step title="应用授权预览" />
      </el-steps>

      <el-form v-if="wizardStep === 0" ref="wizardRef" :model="wizard" :rules="wizardRules" label-width="100px">
        <el-form-item label="租户名称" prop="name"><el-input v-model="wizard.name" placeholder="如 Acme 科技" /></el-form-item>
        <el-form-item label="租户标识" prop="code">
          <el-input v-model="wizard.code" placeholder="登录码, 如 acme (创建后不可改)" />
        </el-form-item>
        <el-form-item label="联系人"><el-input v-model="wizard.contact" /></el-form-item>
        <el-form-item label="套餐"><el-input v-model="wizard.plan" placeholder="standard" /></el-form-item>
        <el-form-item label="到期时间">
          <el-date-picker v-model="wizard.expireAt" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" placeholder="留空=永久" style="width: 100%" />
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="wizard.remark" type="textarea" :rows="2" /></el-form-item>
      </el-form>

      <el-form v-else-if="wizardStep === 1" label-width="120px">
        <el-alert type="info" :closable="false" show-icon style="margin-bottom: 14px"
          title="管理员将获得租户内全套管理能力 (圈养版超管); 首次登录强制修改密码" />
        <el-form-item label="管理员账号" required>
          <el-input v-model="wizard.adminUsername" placeholder="登录账号" />
        </el-form-item>
        <el-form-item label="管理员昵称"><el-input v-model="wizard.adminNickname" placeholder="留空同账号" /></el-form-item>
        <el-form-item label="初始密码">
          <el-input v-model="wizard.adminPassword" placeholder="留空默认 123456, 首登强制改密" show-password />
        </el-form-item>
      </el-form>

      <div v-else>
        <el-alert type="warning" :closable="false" show-icon style="margin-bottom: 12px"
          title="勾选该租户可用的应用模块 (目录/页面级); 可先选套餐预填再微调; 不勾=默认基础协作包; 后续可在「应用授权」抽屉调整" />
        <el-form label-width="90px" style="margin-bottom: 10px">
          <el-form-item label="选择套餐">
            <el-select
              :model-value="wizard.packageId"
              placeholder="从套餐模板预填勾选 (可继续微调)"
              clearable style="width: 100%"
              @update:model-value="(v: any) => { wizard.packageId = v || undefined; applyPackageToWizard(v) }"
            >
              <el-option v-for="p in packages" :key="p.id" :label="`${p.name} (${p.menuIds.length} 个应用)`" :value="p.id" />
            </el-select>
          </el-form-item>
        </el-form>
        <el-tree
          ref="wizardTreeRef"
          :data="menuTreeData"
          show-checkbox node-key="id" default-expand-all check-strictly
          :props="{ label: 'name', children: 'children' }"
          style="max-height: 320px; overflow: auto; border: 1px solid var(--el-border-color-lighter); border-radius: 6px; padding: 8px"
          @check="(_n: any, cs: any) => (wizard.menuIds = cs.checkedKeys)"
        >
          <template #default="{ data }">
            <span style="display: flex; align-items: center; gap: 6px">
              <span>{{ data.name }}</span>
              <el-tag v-if="data.type === 1" size="small" type="info">目录</el-tag>
              <el-tag v-else-if="data.type === 3" size="small" type="warning" effect="plain">按钮</el-tag>
            </span>
          </template>
        </el-tree>
      </div>

      <template #footer>
        <el-button v-if="wizardStep > 0" @click="wizardStep--">上一步</el-button>
        <el-button v-if="wizardStep < 2" type="primary" @click="nextStep">下一步</el-button>
        <el-button v-else type="primary" :loading="creating" @click="submitWizard">创建</el-button>
      </template>
    </el-dialog>

    <!-- 应用授权抽屉 -->
    <el-drawer v-model="authVisible" :title="`应用授权 - ${authTenant?.name || ''}`" size="520px">
      <el-alert type="info" :closable="false" show-icon style="margin-bottom: 12px"
        title="收缩即时清理租户内越权策略; 扩张自动授予租户管理员角色 (买了就用), 普通角色仍由租户管理员分配" />
      <el-select
        v-model="authImportPkg"
        placeholder="从套餐导入授权集 (预填勾选, 仍需保存)"
        clearable size="default" style="width: 100%; margin-bottom: 10px"
        @change="importPackageToAuth"
      >
        <el-option v-for="p in packages" :key="p.id" :label="`${p.name} (${p.menuIds.length} 个应用)`" :value="p.id" />
      </el-select>
      <el-tree
        ref="authTreeRef"
        :data="menuTreeData"
        show-checkbox node-key="id" default-expand-all check-strictly
        :props="treeProps"
        style="border: 1px solid var(--el-border-color-lighter); border-radius: 6px; padding: 8px"
      >
        <template #default="{ data }">
          <span style="display: flex; align-items: center; gap: 6px">
            <span>{{ data.name }}</span>
            <el-tag v-if="data.type === 1" size="small" type="info">目录</el-tag>
            <el-tag v-else-if="data.type === 3" size="small" type="warning" effect="plain">按钮</el-tag>
          </span>
        </template>
      </el-tree>
      <template #footer>
        <el-button @click="authVisible = false">取消</el-button>
        <el-button type="primary" :loading="authSaving" @click="submitAuth">保存授权</el-button>
      </template>
    </el-drawer>
  </div>
  <el-empty v-else description="多租户功能未启用或无访问权限" />
</template>

<style scoped>
.page { display: flex; flex-direction: column; gap: 12px; }
.form-tip { font-size: 12px; color: var(--el-text-color-secondary); line-height: 1.4; margin-top: 2px; }
</style>
