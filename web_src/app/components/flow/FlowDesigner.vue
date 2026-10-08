<script setup lang="ts">
/**
 * 流程设计器 (钉钉式): 发起人 → 审批/抄送/条件分支 → 流程结束。
 * 画布带网点底纹; 节点间连接线由 FlowNode 渲染; 配置抽屉支持 岗位/角色/成员/自选/主管岗。
 * 通过 provide('flowDesigner') 向递归子组件提供操作上下文, 就地修改 modelValue 树。
 */
import { computed, provide, reactive, ref } from 'vue'
import { ElMessageBox } from 'element-plus'
import { Avatar, CircleCheckFilled, Delete, Plus, ZoomIn, ZoomOut } from '@element-plus/icons-vue'
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

// 画布缩放 (CSS zoom: 影响布局, 滚动区域随之收缩; 小屏可缩到 50% 看全貌)
const zoom = ref(1)
function zoomBy(delta: number) {
  zoom.value = Math.min(1.2, Math.max(0.5, Math.round((zoom.value + delta) * 10) / 10))
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

// ---------------- 插入弹窗 ----------------
const plusState = reactive<{ visible: boolean, holder: any, before: FNode | null }>({ visible: false, holder: null, before: null })
function openPlus(holder: any, before: FNode | null) {
  plusState.visible = true
  plusState.holder = holder
  plusState.before = before
}
function pickType(type: 'approver' | 'cc' | 'condition') {
  insertNode(plusState.holder, plusState.before, type)
  plusState.visible = false
}

// ---------------- 审批/抄送节点配置 ----------------
const nodeCfgVisible = ref(false)
const cfgNode = ref<FNode | null>(null)
const approverTypeOptions = [
  { label: '指定成员', value: 'user' },
  { label: '指定角色', value: 'role' },
  { label: '指定岗位', value: 'post' },
  { label: '发起人自选', value: 'selfSelect' },
  { label: '部门主管', value: 'superior' },
  { label: '发起人本人', value: 'initiator' },
]
const ccTypeOptions = approverTypeOptions.filter(o => o.value !== 'selfSelect')
function openNodeCfg(node: FNode) {
  // 超时配置归一 (存量节点无这些字段): 期限 0=不限, 策略默认提醒
  if (node.type === 'approver') {
    if (node.timeoutHours === undefined || node.timeoutHours === null) node.timeoutHours = 0
    if (!node.timeoutAction) node.timeoutAction = 'remind'
  }
  cfgNode.value = node
  nodeCfgVisible.value = true
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

provide('flowDesigner', { openPlus, removeNode, removeBranch, addBranch, openNodeCfg, openBranchCfg, nodeDesc })
</script>

<template>
  <div class="flow-designer">
    <div class="fd-scroller">
    <div class="fd-canvas" :style="{ zoom: zoom }">
      <div class="fd-pill start">
        <el-icon :size="16"><Avatar /></el-icon>
        <span>发起人</span>
      </div>
      <FlowNode :node="modelValue?.child ?? null" :holder="modelValue" />
      <div class="fd-pill end">
        <el-icon :size="16"><CircleCheckFilled /></el-icon>
        <span>流程结束</span>
      </div>
    </div>
    </div>

    <!-- 缩放工具条 -->
    <div class="fd-zoom">
      <el-button size="small" :icon="ZoomOut" @click="zoomBy(-0.1)" />
      <span class="fd-zoom-num" @dblclick="zoom = 1" title="双击复位 100%">{{ Math.round(zoom * 100) }}%</span>
      <el-button size="small" :icon="ZoomIn" @click="zoomBy(0.1)" />
    </div>

    <!-- 添加节点选择 -->
    <el-dialog v-model="plusState.visible" title="添加节点" width="460px" append-to-body>
      <div class="fd-pick">
        <div class="fd-pick-item approver" @click="pickType('approver')">
          <div class="fd-pick-title">审批节点</div>
          <div class="fd-pick-desc">成员 / 角色 / 岗位 / 自选 / 部门主管</div>
        </div>
        <div class="fd-pick-item cc" @click="pickType('cc')">
          <div class="fd-pick-title">抄送节点</div>
          <div class="fd-pick-desc">流程到达时通知指定人</div>
        </div>
        <div class="fd-pick-item condition" @click="pickType('condition')">
          <div class="fd-pick-title">条件分支</div>
          <div class="fd-pick-desc">按表单字段路由 (含默认分支)</div>
        </div>
      </div>
    </el-dialog>

    <!-- 审批/抄送节点配置 -->
    <el-drawer v-model="nodeCfgVisible" :title="cfgNode?.type === 'cc' ? '抄送节点配置' : '审批节点配置'" size="420px" append-to-body>
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
            <span class="fd-tip-inline">小时 (0=不限时)</span>
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
          <div v-if="(cfgNode.timeoutHours || 0) > 0" class="fd-tip">
            逾期后由定时任务处理: 提醒=通知审批人 (一次); 自动转办=系统转给指定人 (转办后不再计时);
            自动通过=系统代为同意并推进。未配置期限的节点不受影响。
          </div>
        </template>
        <div v-if="cfgNode.approverType === 'superior'" class="fd-tip">
          从发起人所在组织起逐级向上查找挂「主管岗」的用户 (跳过发起人自己), 到根仍无则发起失败并提示。
        </div>
      </el-form>
    </el-drawer>

    <!-- 分支条件配置 -->
    <el-drawer v-model="branchCfgVisible" title="分支条件配置" size="480px" append-to-body>
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
        <div v-for="(c, i) in condGroup" :key="i" class="fd-cond-row">
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
/* 画布滚动容器: 大流程在容器内横向/纵向滚动, 不再撑爆弹窗 */
.fd-scroller {
  overflow: auto;
  max-height: 62vh;
  min-height: 300px;
  padding: 20px 12px;
  border-radius: 6px;
  background-image: radial-gradient(var(--el-border-color-lighter) 1px, transparent 1px);
  background-size: 16px 16px;
}
.fd-canvas {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: max-content;
  margin: 0 auto;
}
.fd-zoom {
  position: absolute;
  right: 10px;
  bottom: 10px;
  display: flex;
  align-items: center;
  gap: 4px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 3px 8px;
  box-shadow: 0 2px 6px rgba(31, 41, 55, 0.08);
}
.fd-zoom-num {
  font-size: 12px;
  width: 40px;
  text-align: center;
  cursor: pointer;
  user-select: none;
}
.fd-canvas {
  display: flex;
  flex-direction: column;
  align-items: center;
}
.fd-pill {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 26px;
  border-radius: 20px;
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  user-select: none;
}
.fd-pill.start {
  background: var(--el-text-color-secondary);
}
.fd-pill.end {
  background: var(--el-text-color-placeholder);
  margin-top: 0;
}
/* FlowNode 尾部连线与结束节点之间留出间距 */
.fd-canvas > :deep(.fd-link:last-of-type) {
  margin-bottom: 0;
}

/* 添加节点弹窗 */
.fd-pick {
  display: flex;
  gap: 12px;
  justify-content: center;
}
.fd-pick-item {
  flex: 1;
  padding: 18px 10px;
  text-align: center;
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  cursor: pointer;
  transition: box-shadow 0.15s;
}
.fd-pick-item:hover {
  box-shadow: 0 3px 10px rgba(31, 41, 55, 0.12);
}
.fd-pick-item.approver { border-color: var(--el-color-primary-light-5); }
.fd-pick-item.cc { border-color: var(--el-color-warning-light-5); }
.fd-pick-item.condition { border-color: var(--el-color-success-light-5); }
.fd-pick-title { font-weight: 600; }
.fd-pick-desc { color: var(--el-text-color-secondary); font-size: 12px; margin-top: 6px; }

.fd-cond-row {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}
.fd-tip {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.6;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  padding: 8px 10px;
}
.fd-tip-inline {
  margin-left: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
