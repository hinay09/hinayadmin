<script setup lang="ts">
/**
 * 流程图节点递归组件 (只读, 钉钉式纵向树): 与设计器 FlowNode 同构,
 * 连接线为带箭头虚线, 卡片/分支轨道样式与设计器统一;
 * 连接线/卡片随节点运行状态着色 (states 来自 flowStatesFromTasks),
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
    <!-- 连接线: 带箭头虚线, 随下方节点状态着色 -->
    <div class="fdg-line" :class="`is-${stOf(node)?.state || 'none'}`">
      <i class="fdg-arrow" />
    </div>

    <!-- 审批 / 抄送节点卡片 (钉钉式) -->
    <div v-if="node.type === 'approver' || node.type === 'cc'"
      class="fdg-card" :class="[node.type, `is-${stOf(node)?.state || 'none'}`]">
      <div class="fdg-card-head">
        <span class="fdg-card-ico">
          <el-icon :size="14"><User v-if="node.type === 'approver'" /><Bell v-else /></el-icon>
        </span>
        <span class="fdg-card-name">{{ node.name }}</span>
        <el-tag v-if="stateTag(node)" size="small" effect="light" :type="stateTag(node)!.tag" class="fdg-card-tag">
          {{ stateTag(node)!.text }}
        </el-tag>
      </div>
      <div v-if="stOf(node)" class="fdg-card-desc">
        {{ stOf(node)!.users.join('、') }}<template v-if="stOf(node)!.time"> · {{ stOf(node)!.time }}</template>
      </div>
      <div v-else class="fdg-card-desc is-empty">未经过</div>
    </div>

    <!-- 条件分支: 横向轨道 -->
    <div v-else-if="node.type === 'condition'" class="fdg-cond">
      <div class="fdg-cols">
        <div v-for="br in node.branches || []" :key="br.id" class="fdg-col">
          <div class="fdg-chip" :class="{ active: branchActive(br) }">
            {{ br.name }}<em v-if="br.isDefault">默认</em>
          </div>
          <FlowDiagramNode :node="br.child ?? null" :states="states" />
        </div>
      </div>
    </div>

    <FlowDiagramNode :node="node.child ?? null" :states="states" />
  </template>

  <!-- 链尾: 连接到终点 -->
  <div v-else class="fdg-line is-none">
    <i class="fdg-arrow" />
  </div>
</template>

<style scoped>
/* ---- 连接线 (竖直虚线 + 底端箭头, 随状态着色) ----
 * 虚线只铺到箭头上方 (background-size 裁掉底部), 箭头独立成段不与虚线粘连 */
.fdg-line {
  --fdg-c: var(--el-text-color-placeholder);
  position: relative;
  width: 2px;
  height: 36px;
  flex: none;
  background-image: repeating-linear-gradient(to bottom, var(--fdg-c) 0 5px, transparent 5px 10px);
  background-size: 100% calc(100% - 12px);
  background-repeat: no-repeat;
}
.fdg-line.is-done { --fdg-c: var(--el-color-success-light-3); }
.fdg-line.is-current { --fdg-c: var(--el-color-primary); }
.fdg-line.is-rejected { --fdg-c: var(--el-color-danger-light-3); }
.fdg-line.is-voided { --fdg-c: var(--el-border-color); }
.fdg-arrow {
  position: absolute;
  left: 50%;
  bottom: 0;
  transform: translateX(-50%);
  width: 0;
  height: 0;
  border: 5px solid transparent;
  border-top: 7px solid var(--fdg-c);
}

/* ---- 审批/抄送卡片 (钉钉式, 状态着色) ---- */
.fdg-card {
  width: 262px;
  padding: 10px 12px 11px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
  box-shadow: var(--el-box-shadow-lighter);
}
.fdg-card.is-none {
  border-style: dashed;
  opacity: 0.6;
  box-shadow: none;
}
.fdg-card-head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.fdg-card-ico {
  width: 24px;
  height: 24px;
  border-radius: 5px;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: var(--el-text-color-placeholder);
}
.fdg-card.approver .fdg-card-ico { background: var(--el-color-primary); }
.fdg-card.cc .fdg-card-ico { background: var(--el-color-warning); }
.fdg-card-name {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fdg-card-tag { flex: none; }
.fdg-card-desc {
  margin-top: 7px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fdg-card-desc.is-empty { color: var(--el-text-color-placeholder); }
/* 状态覆写 */
.fdg-card.is-done { border-color: var(--el-color-success-light-5); }
.fdg-card.is-done .fdg-card-ico { background: var(--el-color-success); }
.fdg-card.is-current {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 3px var(--el-color-primary-light-9);
}
.fdg-card.is-rejected { border-color: var(--el-color-danger-light-5); }
.fdg-card.is-rejected .fdg-card-ico { background: var(--el-color-danger); }
.fdg-card.is-voided { border-color: var(--el-border-color-lighter); }
.fdg-card.is-voided .fdg-card-ico { background: var(--el-text-color-placeholder); }

/* ---- 条件分支轨道 (与设计器同构) ---- */
.fdg-cond {
  position: relative;
  padding: 0 14px;
}
.fdg-cols {
  position: relative;
  display: flex;
  align-items: stretch;
  border-top: 2px solid var(--el-text-color-placeholder);
  border-bottom: 2px solid var(--el-text-color-placeholder);
}
.fdg-cols::before,
.fdg-cols::after {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  width: 2px;
  background: var(--el-text-color-placeholder);
}
.fdg-cols::before { left: 0; }
.fdg-cols::after { right: 0; }
.fdg-col {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 14px 16px 16px;
  min-width: 266px;
}
.fdg-col + .fdg-col { border-left: 1px solid var(--el-border-color-lighter); }
.fdg-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin: 2px 0 8px;
  padding: 3px 12px;
  border: 1px solid var(--el-color-primary-light-7);
  border-radius: 12px;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  font-size: 12px;
  font-weight: 500;
  user-select: none;
}
.fdg-chip em {
  font-style: normal;
  color: var(--el-text-color-secondary);
  font-weight: 400;
}
.fdg-chip.active {
  background: var(--el-color-primary);
  border-color: var(--el-color-primary);
  color: #fff;
  font-weight: 600;
}
</style>
