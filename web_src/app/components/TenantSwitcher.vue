<script setup lang="ts">
/**
 * 租户切换器 (顶栏, 仅平台超管 + 多租户开启时渲染) —— docs/multi-tenant.md §8.3。
 *
 * - 切换目标只存 Pinia 内存 (useRequest 据此注入 X-Tenant-Id), 不落任何持久化:
 *   多标签页 JS 上下文天然隔离, 刷新即回平台租户 —— 零串页风险 (评审四);
 * - 切换动作内聚完成全部残态清理: 重置标签页、断开 AI SSE 长流 (事件通知聊天页
 *   abort)、重拉 userinfo/menus/config、跳回工作台 —— 防"看着 A 租户、操作发往 B 租户";
 * - 徽标常显当前操作租户 (未切换时高亮"平台租户"), 关键防误操作锚点 (§8.8)。
 */
import { ElMessage } from 'element-plus'
import { OfficeBuilding } from '@element-plus/icons-vue'
import { useTenantApi, type TenantLiteItem } from '~/composables/useApi'
import { useAuthApi } from '~/composables/useApi'
import { useUserStore } from '~/stores/user'
import { useConfigStore } from '~/stores/config'
import { useTagsStore } from '~/stores/tags'

const userStore = useUserStore()
const configStore = useConfigStore()
const tagsStore = useTagsStore()
const tenantApi = useTenantApi()
const authApi = useAuthApi()
const router = useRouter()

const tenants = ref<TenantLiteItem[]>([])
const visible = computed(() => userStore.isPlatformAdmin && configStore.multiTenant)
const currentName = computed(() => {
  if (userStore.actTenantId > 0) {
    return tenants.value.find(t => t.id === userStore.actTenantId)?.name || `租户 #${userStore.actTenantId}`
  }
  return '平台租户'
})

async function loadTenants() {
  try {
    const res = await tenantApi.listAll()
    tenants.value = res?.list || []
  } catch {}
}

async function handleSwitch(id: number) {
  if (id === userStore.actTenantId) return
  const target = id > 0 ? tenants.value.find(t => t.id === id)?.name || `租户 #${id}` : '平台租户'
  // ① 通知 AI 聊天页断开 SSE 长流 (防"看着 A 租户、流着 B 租户数据")
  if (import.meta.client) window.dispatchEvent(new CustomEvent('tenant:switched'))
  // ② 写入内存态切换目标 (useRequest 后续请求自动携带 X-Tenant-Id)
  userStore.setActTenant(id)
  // ③ 清理残态: 标签页全部重置 (KeepAlive 缓存随 cachedNames 清空)
  tagsStore.$reset()
  try {
    // ④ 以新视角重拉会话与配置
    const [ui, menus] = await Promise.all([authApi.userInfo(), authApi.menus()])
    userStore.setUserInfo(ui)
    userStore.setMenus(menus.menus || [], menus.permissions || [])
    await configStore.fetch()
    ElMessage.success(`已切换到「${target}」`)
  } catch (e: any) {
    ElMessage.error(e?.message || '切换租户失败')
  }
  // ⑤ 回工作台 (当前页面数据仍是旧租户视角)
  router.push('/dashboard')
}

onMounted(() => {
  if (visible.value) loadTenants()
})
</script>

<template>
  <el-dropdown v-if="visible" trigger="click" @visible-change="(o: boolean) => o && loadTenants()">
    <span class="tenant-switcher" :class="{ active: userStore.actTenantId > 0 }" title="平台超管代运维: 切换数据面租户">
      <el-icon><OfficeBuilding /></el-icon>
      <span class="name">{{ currentName }}</span>
      <el-icon class="arrow"><ArrowDown /></el-icon>
    </span>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item :disabled="userStore.actTenantId === 0" @click="handleSwitch(0)">
          <el-tag size="small" type="warning">平台</el-tag>
          <span style="margin-left: 6px">平台租户 (跨租户全量)</span>
        </el-dropdown-item>
        <el-dropdown-item divided v-for="t in tenants" :key="t.id" :disabled="t.id === userStore.actTenantId" @click="handleSwitch(t.id)">
          <el-tag size="small" type="info">租户</el-tag>
          <span style="margin-left: 6px">{{ t.name }} ({{ t.code }})</span>
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<style scoped>
.tenant-switcher {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 32px;
  padding: 0 10px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  outline: none;
}
.tenant-switcher:hover {
  background: var(--el-fill-color);
}
.tenant-switcher.active {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary-light-5);
  background: var(--el-color-primary-light-9);
}
.tenant-switcher .name {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
}
.tenant-switcher .arrow {
  font-size: 12px;
}
</style>
