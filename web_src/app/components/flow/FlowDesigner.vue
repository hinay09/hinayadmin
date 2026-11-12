<script setup lang="ts">
/**
 * 流程设计器 (钉钉式画布): 发起人 → 审批/抄送/条件分支 → 流程结束。
 * 画布纯浅灰底; 节点间为带箭头虚线, "+"气泡就地加节点 (FlowNode 内 el-popover);
 * 配置抽屉带彩色图标头部与头部删除; 缩放工具条在左下角 (含适应画布)。
 * 通过 provide('flowDesigner') 向递归子组件提供操作上下文, 就地修改 modelValue 树。
 */
import { computed, nextTick, provide, ref } from 'vue'
import { ElMessageBox } from 'element-plus'
import { Avatar, Bell, CircleCheckFilled, Delete, FullScreen, Plus, Share, User, ZoomIn, ZoomOut } from '@element-plus/icons-vue'
import FlowNode from './FlowNode.vue'
import type { FlowBranch, FlowNode as FNode, FlowRoleOption, FlowUserOption, FlowPostOption, FlowFormField } from '~/composables/useApi/flow'

const props = defineProps<{
  modelValue: FNode
  users: FlowUserOption[]
  roles: FlowRoleOption[]
  posts: FlowPostOption[]
  fields: FlowFormField[]
}>()

const emit = defineEmits<{ (e: 'change'): void }>()

// ---------------- 画布缩放 ----------------
// CSS zoom: 影响布局, 滚动区域随之收缩; 适应画布时先复位 100% 再量实际宽度
const zoom = ref(1)
const scrollerRef = ref<HTMLElement>()
const canvasRef = ref<HTMLElement>()
function zoomBy(delta: number) {
  zoom.value = Math.min(1.2, Math.max(0.5, Math.round((zoom.value + delta) * 10) / 10))
}
async function fitView() {
  zoom.value = 1
  await nextTick()
  const s = scrollerRef.value, c = canvasRef.value
  if (!s || !c || !c.scrollWidth) return
  const ratio = Math.floor(((s.clientWidth - 24) / c.scrollWidth) * 10) / 10
  zoom.value = Math.max(0.5, Math.min(1, ratio))
}

let idSeq = 0
function genId(prefix: string) {
  idSeq += 1
  return `${prefix}_${Date.now() % 100000}_${idSeq}`
}
function touched() { emit('change') }

const userMap = computed(() => {
  const m: Record<number, string> = {}
  for (const u of props.users) m[u.id] = u.nickname || u.username
  return m
})
const roleMap = computed(() => {
  const m: Record<number, string> = {}
  for (const r of props.roles) m[r.id] = r.name
  return m
})
const postMap = computed(() => {
  const m: Record<number, string> = {}
  for (const p of props.posts) m[p.id] = p.name + (p.kind === 2 ? '(主管岗)' : '')
  return m
})

// ---------------- 节点树操作 ----------------
function newNode(type: 'approver' | 'cc' | 'condition'): FNode {
  if (type === 'condition') {
    return {
      id: genId('c'), type, name: '条件分支',
      branches: [
        { id: genId('b'), name: '条件1', isDefault: false, conditions: [[]], child: null },
        { id: genId('b'), name: '默认', isDefault: true, conditions: [], child: null },
      ],
    }
  }
  return {
    id: genId(type === 'approver' ? 'a' : 's'), type,
    name: type === 'approver' ? '审批人' : '抄送人',
    approverType: 'user', approverIds: [], signType: 'any',
  }
}
function insertNode(holder: any, before: FNode | null, type: 'approver' | 'cc' | 'condition') {
  const n = newNode(type)
  n.child = before
  holder.child = n
  touched()
}
async function removeNode(holder: any, node: FNode) {
  // 条件分支节点: 其下各分支链条一并删除, 之后的节点接续保留
  if (node.type === 'condition') {
    const used = (node.branches || []).filter(b => b.child).length
    await ElMessageBox.confirm(
      `删除条件分支「${node.name || '条件分支'}」?` + (used ? `其 ${used} 个分支内的节点将一并删除,` : '') + '分支之后的节点将保留。',
      '删除确认', { type: 'warning' },
    )
  }
  holder.child = node.child
  touched()
}
async function removeBranch(node: FNode, br: FlowBranch) {
  if (br.child) {
    await ElMessageBox.confirm(`删除分支「${br.name}」及其内部节点?`, '删除确认', { type: 'warning' })
  }
  node.branches = (node.branches || []).filter(b => b !== br)
  touched()
}
function addBranch(node: FNode) {
  node.branches = node.branches || []
  node.branches.push({ id: genId('b'), name: `条件${node.branches.length}`, isDefault: false, conditions: [[]], child: null })
  touched()
}
function nodeDesc(node: FNode): string {
  let who = ''
  switch (node.approverType) {
    case 'user':
      who = (node.approverIds || []).map(id => userMap.value[id] || `用户${id}`).join('、') || '未选择'
      break
    case 'role':
      who = (node.approverIds || []).length
        ? `角色: ${(node.approverIds || []).map(id => roleMap.value[id] || `角色${id}`).join('、')}`
        : '未选择'
      break
    case 'post':
      who = (node.approverIds || []).length
        ? `岗位: ${(node.approverIds || []).map(id => postMap.value[id] || `岗位${id}`).join('、')}`
        : '未选择'
      break
    case 'selfSelect': who = '发起人自选'; break
    case 'superior': who = '部门主管 (按主管岗逐级向上)'; break
    case 'initiator': who = '发起人本人'; break
    default: who = '未设置'
  }
  if (node.type === 'approver' && node.signType === 'all') who += ' (会签)'
  // 超时配置摘要: 卡片上一眼可见期限与策略
  if (node.type === 'approver' && (node.timeoutHours || 0) > 0) {
    const act = node.timeoutAction === 'transfer' ? '超时转办'
      : node.timeoutAction === 'approve' ? '超时自动通过' : '超时提醒'
    who += ` · 限${node.timeoutHours}h(${act})`
  }
  return who
}

// ---------------- 审批/抄送节点配置 ----------------
const nodeCfgVisible = ref(false)
const cfgNode = ref<FNode | null>(null)
// 节点宿主 (child 指向该节点), 抽屉头部删除用
const cfgHolder = ref<any>(null)
const approverTypeOptions = [
  { label: '指定成员', value: 'user' },
  { label: '指定角色', value: 'role' },
  { label: '指定岗位', value: 'post' },
  { label: '发起人自选', value: 'selfSelect' },
  { label: '部门主管', value: 'superior' },
  { label: '发起人本人', value: 'initiator' },
]
const ccTypeOptions = approverTypeOptions.filter(o => o.value !== 'selfSelect')
function openNodeCfg(node: FNode, holder?: any) {
  // 超时配置归一 (存量节点无这些字段): 期限 0=不限, 策略默认提醒
  if (node.type === 'approver') {
    if (node.timeoutHours === undefined || node.timeoutHours === null) node.timeoutHours = 0
    if (!node.timeoutAction) node.timeoutAction = 'remind'
  }
  cfgHolder.value = holder ?? null
  cfgNode.value = node
  nodeCfgVisible.value = true
}
async function removeCfgNode() {
  const node = cfgNode.value
  const holder = cfgHolder.value
  if (!node || !holder) return
  nodeCfgVisible.value = false
  await removeNode(holder, node)
}
const cfgOptions = computed(() => {
  if (!cfgNode.value) return []
  if (cfgNode.value.approverType === 'role') return props.roles.map(r => ({ label: r.name, value: r.id }))
  if (cfgNode.value.approverType === 'post') {
    return props.posts.map(p => ({ label: p.name + (p.kind === 2 ? ' (主管岗)' : ''), value: p.id }))
  }
  return props.users.map(u => ({ label: u.nickname || u.username, value: u.id }))
})

// ---------------- 分支条件配置 ----------------
const branchCfgVisible = ref(false)
const cfgBranch = ref<FlowBranch | null>(null)
const cfgOwner = ref<FNode | null>(null)
const opOptions = [
  { label: '等于', value: 'eq' },
  { label: '不等于', value: 'ne' },
  { label: '大于', value: 'gt' },
  { label: '小于', value: 'lt' },
  { label: '大于等于', value: 'ge' },
  { label: '小于等于', value: 'le' },
  { label: '属于', value: 'in' },
]
function openBranchCfg(owner: FNode, br: FlowBranch) {
  cfgOwner.value = owner
  cfgBranch.value = br
  if (!br.conditions || br.conditions.length === 0) br.conditions = [[]]
  branchCfgVisible.value = true
}
/** UI 编辑第一个条件组 (组内 AND); 多组 OR 由复制 JSON 的高级用法扩展 */
const condGroup = computed(() => cfgBranch.value?.conditions?.[0] || [])
function setDefault(on: any) {
  const br = cfgBranch.value
  if (!br || !cfgOwner.value) return
  if (!on) {
    br.isDefault = false
    touched()
    return
  }
  for (const b of cfgOwner.value.branches || []) b.isDefault = (b === br)
  touched()
}
function addCond() {
  const br = cfgBranch.value
  if (!br) return
  if (!br.conditions?.length) br.conditions = [[]]
  br.conditions[0].push({ field: props.fields[0]?.key || '', op: 'eq', value: '' })
  touched()
}
function removeCond(i: number) {
  cfgBranch.value?.conditions?.[0]?.splice(i, 1)
  touched()
}
function fieldOptionsOf(key: string): string[] {
  const f = props.fields.find(x => x.key === key)
  return f?.type === 'select' ? (f.options || []) : []
}

provide('flowDesigner', { insertNode, removeNode, removeBranch, addBranch, openNodeCfg, openBranchCfg, nodeDesc })
</script>

<template>
  <div class="flow-designer">
    <div ref="scrollerRef" class="fdz-scroller">
      <div ref="canvasRef" class="fdz-canvas" :style="{ zoom }">
        <div class="fdz-pill start">
          <el-icon :size="15"><Avatar /></el-icon>
          <span>发起人</span>
        </div>
        <FlowNode :node="modelValue?.child ?? null" :holder="modelValue" />
        <div class="fdz-pill end">
          <el-icon :size="15"><CircleCheckFilled /></el-icon>
          <span>流程结束</span>
        </div>
      </div>
    </div>

    <!-- 缩放工具条 (左下角, 钉钉式) -->
    <div class="fdz-zoom">
      <el-button link :icon="ZoomOut" @click="zoomBy(-0.1)" />
      <span class="fdz-zoom-num" title="双击复位 100%" @dblclick="zoom = 1">{{ Math.round(zoom * 100) }}%</span>
      <el-button link :icon="ZoomIn" @click="zoomBy(0.1)" />
      <el-divider direction="vertical" />
      <el-button link :icon="FullScreen" @click="fitView">适应</el-button>
    </div>

    <!-- 审批/抄送节点配置 (钉钉式彩色头部 + 头部删除) -->
    <el-drawer v-model="nodeCfgVisible" size="480px" append-to-body class="fdz-drawer">
      <template #header>
        <div class="fdz-dhead">
          <span class="fdz-dico" :class="cfgNode?.type === 'cc' ? 'cc' : 'approver'">
            <el-icon :size="17"><Bell v-if="cfgNode?.type === 'cc'" /><User v-else /></el-icon>
          </span>
          <span class="fdz-dtitle">{{ cfgNode?.name || (cfgNode?.type === 'cc' ? '抄送节点' : '审批节点') }}</span>
          <el-button v-if="cfgNode" link type="danger" :icon="Delete" @click="removeCfgNode">删除</el-button>
        </div>
      </template>
      <el-form v-if="cfgNode" label-width="90px">
        <el-form-item label="节点名称">
          <el-input v-model="cfgNode.name" maxlength="30" />
        </el-form-item>
        <el-form-item label="人员类型">
          <el-select v-model="cfgNode.approverType" @change="cfgNode!.approverIds = []">
            <el-option v-for="t in (cfgNode.type === 'cc' ? ccTypeOptions : approverTypeOptions)"
              :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="['user', 'role', 'post'].includes(cfgNode.approverType || '')" label="选择对象">
          <el-select v-model="cfgNode.approverIds" multiple filterable collapse-tags collapse-tags-tooltip
            :placeholder="cfgNode.approverType === 'role' ? '选择角色' : (cfgNode.approverType === 'post' ? '选择岗位' : '选择成员')">
            <el-option v-for="o in cfgOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="cfgNode.type === 'approver'" label="签核方式">
          <el-radio-group v-model="cfgNode.signType">
            <el-radio value="any">或签 (任一人通过)</el-radio>
            <el-radio value="all">会签 (全部通过)</el-radio>
          </el-radio-group>
        </el-form-item>
        <!-- 超时处理: 期限物化到任务行, 由定时任务 flow.timeoutScan 扫描 -->
        <template v-if="cfgNode.type === 'approver'">
          <el-divider content-position="left">超时处理</el-divider>
          <el-form-item label="办理期限">
            <el-input-number v-model="cfgNode.timeoutHours" :min="0" :max="720" :step="1" controls-position="right" />
            <span class="fdz-tip-inline">小时 (0=不限时)</span>
          </el-form-item>
          <el-form-item v-if="(cfgNode.timeoutHours || 0) > 0" label="超时策略">
            <el-radio-group v-model="cfgNode.timeoutAction">
              <el-radio value="remind">提醒</el-radio>
              <el-radio value="transfer">自动转办</el-radio>
              <el-radio value="approve">自动通过</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="(cfgNode.timeoutHours || 0) > 0 && cfgNode.timeoutAction === 'transfer'" label="转办给">
            <el-select v-model="cfgNode.timeoutTransfer" filterable placeholder="超时后转给谁" style="width:100%">
              <el-option v-for="u in users" :key="u.id" :label="u.nickname || u.username" :value="u.id" />
            </el-select>
          </el-form-item>
          <div v-if="(cfgNode.timeoutHours || 0) > 0" class="fdz-tip">
            逾期后由定时任务处理: 提醒=通知审批人 (一次); 自动转办=系统转给指定人 (转办后不再计时);
            自动通过=系统代为同意并推进。未配置期限的节点不受影响。
          </div>
        </template>
        <div v-if="cfgNode.approverType === 'superior'" class="fdz-tip">
          从发起人所在组织起逐级向上查找挂「主管岗」的用户 (跳过发起人自己), 到根仍无则发起失败并提示。
        </div>
      </el-form>
    </el-drawer>

    <!-- 分支条件配置 -->
    <el-drawer v-model="branchCfgVisible" size="500px" append-to-body class="fdz-drawer">
      <template #header>
        <div class="fdz-dhead">
          <span class="fdz-dico condition"><el-icon :size="17"><Share /></el-icon></span>
          <span class="fdz-dtitle">{{ cfgBranch?.name || '分支条件' }}</span>
        </div>
      </template>
      <el-form v-if="cfgBranch" label-width="90px">
        <el-form-item label="分支名称">
          <el-input v-model="cfgBranch.name" maxlength="30" />
        </el-form-item>
        <el-form-item label="默认分支">
          <el-switch :model-value="cfgBranch.isDefault" @update:model-value="(v: any) => setDefault(v)" />
          <span style="margin-left:8px;color:var(--el-text-color-secondary);font-size:12px">其他分支均未命中时走此分支</span>
        </el-form-item>
        <el-divider content-position="left">条件 (同时满足)</el-divider>
        <div v-if="cfgBranch.isDefault" style="color:var(--el-text-color-secondary)">默认分支无需配置条件</div>
        <div v-for="(c, i) in condGroup" :key="i" class="fdz-cond-row">
          <el-select v-model="c.field" filterable placeholder="字段" style="width:130px">
            <el-option v-for="f in fields" :key="f.key" :label="f.label" :value="f.key" />
          </el-select>
          <el-select v-model="c.op" style="width:110px">
            <el-option v-for="o in opOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
          <el-select v-if="fieldOptionsOf(c.field).length" v-model="c.value" style="width:150px" placeholder="值">
            <el-option v-for="v in fieldOptionsOf(c.field)" :key="v" :label="v" :value="v" />
          </el-select>
          <el-input v-else v-model="c.value" placeholder="值" style="width:150px" />
          <el-button link type="danger" :icon="Delete" @click="removeCond(i)" />
        </div>
        <el-button v-if="!cfgBranch.isDefault" style="margin-top:8px" type="primary" plain size="small" :icon="Plus" @click="addCond">
          添加条件
        </el-button>
        <div style="margin-top:12px;color:var(--el-text-color-secondary);font-size:12px">
          提示: 分支按从左到右顺序求值, 首个命中的分支生效; 大于/小于等按数值比较。
        </div>
      </el-form>
    </el-drawer>
  </div>
</template>

<style scoped>
.flow-designer {
  position: relative;
  background: var(--el-fill-color-lighter);
  border-radius: 6px;
}
/* 画布滚动容器: 纯浅灰底 (钉钉式无网点), 大流程在容器内横向/纵向滚动 */
.fdz-scroller {
  overflow: auto;
  max-height: 62vh;
  min-height: 300px;
  padding: 24px 12px;
  border-radius: 6px;
  background: var(--el-fill-color-light);
}
.fdz-canvas {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: max-content;
  margin: 0 auto;
}
/* 缩放工具条 (左下角) */
.fdz-zoom {
  position: absolute;
  left: 10px;
  bottom: 10px;
  display: flex;
  align-items: center;
  gap: 2px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 3px 8px;
  box-shadow: var(--el-box-shadow-light);
}
.fdz-zoom-num {
  font-size: 12px;
  min-width: 38px;
  text-align: center;
  cursor: pointer;
  user-select: none;
}

/* ---- 发起/结束胶囊 ---- */
.fdz-pill {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 26px;
  border-radius: 999px;
  font-size: 13px;
  font-weight: 600;
  user-select: none;
}
.fdz-pill.start {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border: 1px solid var(--el-color-primary-light-7);
}
.fdz-pill.end {
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
}

/* ---- 抽屉头部 (钉钉式彩色图标 + 标题 + 删除) ---- */
.fdz-dhead {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 28px;
}
.fdz-dico {
  width: 30px;
  height: 30px;
  border-radius: 7px;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}
.fdz-dico.approver { background: var(--el-color-primary); }
.fdz-dico.cc { background: var(--el-color-warning); }
.fdz-dico.condition { background: var(--el-color-success); }
.fdz-dtitle {
  flex: 1;
  min-width: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.fdz-cond-row {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}
.fdz-tip {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.6;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  padding: 8px 10px;
}
.fdz-tip-inline {
  margin-left: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
