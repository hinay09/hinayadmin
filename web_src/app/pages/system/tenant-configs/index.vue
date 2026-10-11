<script setup lang="ts">
/**
 * 租户设置 (租户管理员维护白名单覆盖配置) —— docs/multi-tenant.md D5/§8。
 *
 * - 仅展示/维护**本租户覆盖行** (租户注入天然限界), 键限白名单
 *   (站点名/Logo/版权/注册开关/密码策略); 平台级行此处不可见;
 * - 无覆盖行的键显示"跟随平台默认", 新建覆盖行后即生效
 *   (configs/all 读序: 租户 > 平台, 前端布局自动生效);
 * - 删除由平台"重置租户配置"干预口承担, 本页不提供删除。
 */
import { ElMessage, type FormInstance } from 'element-plus'
import { Plus, Edit, Search } from '@element-plus/icons-vue'
import { useConfigApi } from '~/composables/useApi'

definePageMeta({ title: '租户设置' })
defineOptions({ name: 'system-tenant-configs' })

const configApi = useConfigApi()

/** 租户可覆盖键白名单 (与后端 configx.OverrideAllowed 对齐) */
const WHITELIST_KEYS: Array<{ key: string, name: string, type: number }> = [
  { key: 'sys.name', name: '站点名称', type: 0 },
  { key: 'sys.logo', name: '站点 Logo (图片URL)', type: 0 },
  { key: 'sys.copyright', name: '页脚版权文字', type: 0 },
  { key: 'sys.allow_register', name: '开放注册 (布尔)', type: 2 },
  { key: 'sys.password.min_length', name: '密码最小长度', type: 1 },
  { key: 'sys.password.max_length', name: '密码最大长度', type: 1 },
  { key: 'sys.password.require_upper', name: '密码需含大写字母 (布尔)', type: 2 },
  { key: 'sys.password.require_lower', name: '密码需含小写字母 (布尔)', type: 2 },
  { key: 'sys.password.require_digit', name: '密码需含数字 (布尔)', type: 2 },
  { key: 'sys.password.require_special', name: '密码需含特殊字符 (布尔)', type: 2 },
  { key: 'sys.password.expire_days', name: '密码有效期天数 (0=永不过期)', type: 1 },
]

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const query = reactive({ keyword: '', page: 1, pageSize: 10 })

async function loadList() {
  loading.value = true
  try {
    const res: any = await configApi.list(query)
    list.value = res?.list || []
    total.value = res?.total || 0
  } finally {
    loading.value = false
  }
}
onMounted(loadList)

/** 未覆盖的白名单键 (新建下拉选项) */
const uncoveredKeys = computed(() =>
  WHITELIST_KEYS.filter(k => !list.value.some(r => r.configKey === k.key) || query.keyword === ''))

/* ---- 新增/编辑抽屉 ---- */
const drawerRef = ref<FormInstance>()
const drawerVisible = ref(false)
const isEdit = ref(false)
const form = reactive({ id: 0, configKey: '', configValue: '', configType: 0, name: '', remark: '', status: 1, sort: 0 })

function openCreate() {
  isEdit.value = false
  const first = uncoveredKeys.value[0] || WHITELIST_KEYS[0]
  Object.assign(form, { id: 0, configKey: first.key, configValue: '', configType: first.type, name: first.name, remark: '租户覆盖', status: 1, sort: 0 })
  drawerVisible.value = true
}

function openEdit(row: any) {
  isEdit.value = true
  Object.assign(form, { id: row.id, configKey: row.configKey, configValue: row.configValue, configType: row.configType, name: row.name, remark: row.remark, status: row.status, sort: row.sort })
  drawerVisible.value = true
}

async function handleSubmit() {
  if (isEdit.value) {
    await configApi.update(form.id, { ...form })
  } else {
    const meta = WHITELIST_KEYS.find(k => k.key === form.configKey)
    await configApi.create({ ...form, name: meta?.name || form.name, configType: meta?.type ?? form.configType })
  }
  ElMessage.success('已保存 (立即生效)')
  drawerVisible.value = false
  loadList()
}

function boolPlaceholder() { return form.configType === 2 ? 'true / false' : '配置值' }
</script>

<template>
  <div class="page">
    <el-alert type="info" :closable="false" show-icon style="margin-bottom: 0"
      title="此处维护本租户的个性化配置 (站点名/Logo/版权/注册开关/密码策略); 未覆盖的键跟随平台默认值" />
    <el-card>
      <el-form inline @submit.prevent>
        <el-form-item label="键">
          <el-input v-model="query.keyword" placeholder="配置键" clearable :prefix-icon="Search" style="width: 220px" @keyup.enter="loadList" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="loadList">查询</el-button>
          <el-button type="success" :icon="Plus" :disabled="!uncoveredKeys.length" @click="openCreate">新增覆盖</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="list" border stripe>
        <el-table-column prop="configKey" label="配置键" min-width="200" />
        <el-table-column prop="name" label="名称" min-width="150" />
        <el-table-column label="值" min-width="180">
          <template #default="{ row }">
            <el-tag v-if="row.configType === 2" size="small" :type="row.configValue === 'true' || row.configValue === '1' ? 'success' : 'info'">{{ row.configValue }}</el-tag>
            <span v-else>{{ row.configValue }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">{{ row.status === 1 ? '生效' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="110" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" :icon="Edit" @click="openEdit(row)">编辑</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="!list.length && !loading" class="empty-tip">
        暂无覆盖配置 —— 全部跟随平台默认值; 点击「新增覆盖」个性化本租户的站点名/Logo/密码策略等。
      </div>

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

    <el-drawer v-model="drawerVisible" :title="isEdit ? '编辑覆盖配置' : '新增覆盖配置'" size="480px">
      <el-form ref="drawerRef" :model="form" label-width="90px">
        <el-form-item label="配置键">
          <el-select v-if="!isEdit" v-model="form.configKey" style="width: 100%">
            <el-option v-for="k in uncoveredKeys" :key="k.key" :label="`${k.name} (${k.key})`" :value="k.key" />
          </el-select>
          <el-input v-else :model-value="form.configKey" disabled />
        </el-form-item>
        <el-form-item label="配置值">
          <el-switch v-if="form.configType === 2" :model-value="form.configValue === 'true' || form.configValue === '1'"
            @change="(v: any) => (form.configValue = v ? 'true' : 'false')" />
          <el-input v-else v-model="form.configValue" :placeholder="boolPlaceholder()" />
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">生效</el-radio>
            <el-radio :value="0">停用 (跟随平台)</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="drawerVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">保存</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped>
.page { display: flex; flex-direction: column; gap: 12px; }
.empty-tip { color: var(--el-text-color-secondary); font-size: 13px; padding: 8px 4px; }
</style>
