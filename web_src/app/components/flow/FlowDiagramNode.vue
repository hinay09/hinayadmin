<script setup lang="ts">
/**
 * 流程图节点递归组件 (只读, 钉钉式纵向树): 与设计器 FlowNode 同构,
 * 连接线/卡片随节点运行状态着色 (states 来自 flowStatesFromTasks);
 * 条件分支为横向轨道, 实际命中的分支标签高亮。
 */
import { User, Bell } from '@element-plus/icons-vue'
import type { FlowNode as FNode, FlowNodeState } from '~/composables/useApi/flow'

defineOptions({ name: 'FlowDiagramNode' })

const props = defineProps<{ node: FNode | null, states?: Record<string, FlowNodeState> }>()

function stOf(n: FNode): FlowNodeState | undefined { return props.states?.[n.id] }

/** 分支是否被命中: 分支链上任一节点有运行状态 */
function branchActive(br: { child?: FNode | null }): boolean {
  let cur = br.child
  while (cur) {
    if (stOf(cur)) return true
    if (cur.type === 'condition') {
      for (const b of cur.branches || []) if (branchActive(b)) return true
    }
    cur = cur.child
  }
  return false
}

const stateTexts: Record<string, { text: string, tag: string }> = {
  done: { text: '已通过', tag: 'success' },
  current: { text: '进行中', tag: 'primary' },
  rejected: { text: '已驳回', tag: 'danger' },
  voided: { text: '已失效', tag: 'info' },
}

function stateTag(n: FNode) {
  const s = stOf(n)
  if (!s) return null
  if (n.type === 'cc') {
    return s.state === 'done' ? { text: '已阅', tag: 'success' } : { text: '待阅', tag: 'primary' }
  }
  return stateTexts[s.state] || null
}
</script>

<template>
  <template v-if="node">
    <!-- 连接线: 随下方节点状态着色 -->
    <div class="fdg-line" :class="`is-${stOf(node)?.state || 'none'}`" />

    <!-- 审批 / 抄送节点卡片 -->
    <div v-if="node.type === 'approver' || node.type === 'cc'"
      class="fdg-card" :class="[node.type, `is-${stOf(node)?.state || 'none'}`]">
      <div class="fdg-card-icon">
        <el-icon :size="18"><User v-if="node.type === 'approver'" /><Bell v-else /></el-icon>
      </div>
      <div class="fdg-card-main">
        <div class="fdg-card-title">
          <span class="fdg-card-name">{{ node.name }}</span>
          <el-tag v-if="stateTag(node)" size="small" effect="light" :type="stateTag(node)!.tag">
            {{ stateTag(node)!.text }}
          </el-tag>
        </div>
        <div v-if="stOf(node)" class="fdg-card-desc">
          {{ stOf(node)!.users.join('、') }}<template v-if="stOf(node)!.time"> · {{ stOf(node)!.time }}</template>
        </div>
      </div>
    </div>

    <!-- 条件分支: 横向轨道 -->
    <div v-else-if="node.type === 'condition'" class="fdg-cond">
      <div class="fdg-cols">
        <div v-for="br in node.branches || []" :key="br.id" class="fdg-col">
          <div class="fdg-chip" :class="{ active: branchActive(br) }">
            {{ br.name }}<span v-if="br.isDefault" class="fdg-def">默认</span>
          </div>
          <FlowDiagramNode :node="br.child ?? null" :states="states" />
        </div>
      </div>
    </div>

    <FlowDiagramNode :node="node.child ?? null" :states="states" />
  </template>

  <!-- 链尾: 连接到终点 -->
  <div v-else class="fdg-line is-none" />
</template>

<style scoped>
/* ---- 连接线 ---- */
.fdg-line {
  width: 2px;
  height: 28px;
  background: #dcdfe6;
  flex: none;
}
.fdg-line.is-done { background: var(--el-color-success-light-5); }
.fdg-line.is-current { background: var(--el-color-primary-light-5); }
.fdg-line.is-rejected { background: var(--el-color-danger-light-5); }
.fdg-line.is-voided { background: #dcdfe6; }

/* ---- 审批/抄送卡片 ---- */
.fdg-card {
  display: flex;
  align-items: stretch;
  width: 268px;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  box-shadow: 0 1px 3px rgba(31, 41, 55, 0.06);
}
.fdg-card.is-none {
  border-style: dashed;
  opacity: 0.6;
  box-shadow: none;
}
.fdg-card-icon {
  width: 38px;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px 0 0 6px;
  background: #f5f7fa;
  color: #909399;
}
.fdg-card.approver.is-none .fdg-card-icon {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}
.fdg-card.cc.is-none .fdg-card-icon {
  background: var(--el-color-warning-light-9);
  color: var(--el-color-warning);
}
.fdg-card.is-done { border-color: var(--el-color-success-light-5); }
.fdg-card.is-done .fdg-card-icon { background: var(--el-color-success-light-9); color: var(--el-color-success); }
.fdg-card.is-current {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 3px var(--el-color-primary-light-9);
}
.fdg-card.is-current .fdg-card-icon { background: var(--el-color-primary-light-9); color: var(--el-color-primary); }
.fdg-card.is-rejected { border-color: var(--el-color-danger-light-5); }
.fdg-card.is-rejected .fdg-card-icon { background: var(--el-color-danger-light-9); color: var(--el-color-danger); }
.fdg-card.is-voided { border-color: #e4e7ed; }
.fdg-card.is-voided .fdg-card-icon { background: #f4f4f5; color: #909399; }
.fdg-card-main {
  flex: 1;
  min-width: 0;
  padding: 8px 10px;
}
.fdg-card-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.fdg-card-name {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fdg-card-desc {
  font-size: 12px;
  color: #9aa1ac;
  margin-top: 3px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ---- 条件分支轨道 ---- */
.fdg-cond { margin: 0; }
.fdg-cols {
  display: flex;
  align-items: stretch;
  border-top: 2px solid #dcdfe6;
  border-bottom: 2px solid #dcdfe6;
  border-left: 2px solid #dcdfe6;
  border-right: 2px solid #dcdfe6;
  border-radius: 2px;
}
.fdg-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 8px 14px;
  min-width: 272px;
}
.fdg-col + .fdg-col { border-left: 2px solid #dcdfe6; }
.fdg-chip {
  font-size: 12px;
  color: var(--el-color-success);
  border: 1px dashed var(--el-color-success-light-5);
  background: var(--el-color-success-light-9);
  border-radius: 12px;
  padding: 3px 12px;
  margin: 2px 0 6px;
  user-select: none;
}
.fdg-chip.active {
  border-style: solid;
  background: var(--el-color-success);
  border-color: var(--el-color-success);
  color: #fff;
  font-weight: 600;
}
.fdg-def { margin-left: 4px; color: #9aa1ac; }
</style>
