<script setup lang="ts">
/**
 * 只读流程图 (公共组件): 按 flow_conf 快照渲染钉钉式纵向节点树。
 * 可选传入节点运行状态 (flowStatesFromTasks) 与终点状态 (flowEndState) 实现进度着色。
 * 业务页"查看流程图"用 FlowDiagramDialog; 本组件供详情页等直接内嵌。
 */
import { Position } from '@element-plus/icons-vue'
import type { FlowNode as FNode, FlowNodeState } from '~/composables/useApi/flow'
import FlowDiagramNode from './FlowDiagramNode.vue'

defineOptions({ name: 'FlowDiagram' })

withDefaults(defineProps<{
  root?: FNode | null
  states?: Record<string, FlowNodeState>
  /** 发起人昵称 (展示在发起节点上, 可空) */
  startUser?: string
  endState?: 'done' | 'rejected' | 'voided' | 'warn' | 'wait'
  endText?: string
  legend?: boolean
}>(), {
  root: null,
  states: undefined,
  startUser: '',
  endState: 'wait',
  endText: '结束',
  legend: true,
})
</script>

<template>
  <div class="fdg">
    <div v-if="legend" class="fdg-legend">
      <span><i class="is-done" />已通过</span>
      <span><i class="is-current" />进行中</span>
      <span><i class="is-rejected" />已驳回</span>
      <span><i class="is-none" />未经过</span>
    </div>
    <div class="fdg-scroll">
      <div class="fdg-tree">
        <div class="fdg-pill is-done">
          <el-icon :size="14"><Position /></el-icon>
          <span>发起人</span>
          <span v-if="startUser" class="fdg-pill__user">{{ startUser }}</span>
        </div>
        <FlowDiagramNode :node="root?.child ?? null" :states="states" />
        <div class="fdg-pill" :class="`is-${endState}`">{{ endText }}</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.fdg-legend {
  display: flex;
  gap: 16px;
  justify-content: flex-end;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 12px;
}
.fdg-legend i {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  margin-right: 4px;
  vertical-align: -1px;
}
.fdg-legend .is-done { background: var(--el-color-success); }
.fdg-legend .is-current { background: var(--el-color-primary); }
.fdg-legend .is-rejected { background: var(--el-color-danger); }
.fdg-legend .is-none { background: var(--el-bg-color); border: 1px dashed var(--el-text-color-disabled); }

.fdg-scroll { overflow-x: auto; }
.fdg-tree {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: max-content;
  padding: 4px 24px 10px;
  margin: 0 auto;
}

/* ---- 发起/终点胶囊 ---- */
.fdg-pill {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 18px;
  border-radius: 999px;
  font-size: 13px;
  font-weight: 600;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-regular);
  border: 1px solid var(--el-border-color-light);
}
.fdg-pill.is-done {
  background: var(--el-color-success-light-9);
  color: var(--el-color-success);
  border-color: var(--el-color-success-light-5);
}
.fdg-pill.is-rejected {
  background: var(--el-color-danger-light-9);
  color: var(--el-color-danger);
  border-color: var(--el-color-danger-light-5);
}
.fdg-pill.is-warn {
  background: var(--el-color-warning-light-9);
  color: var(--el-color-warning);
  border-color: var(--el-color-warning-light-5);
}
.fdg-pill.is-voided { background: var(--el-fill-color-dark); color: var(--el-text-color-secondary); }
.fdg-pill.is-wait { border-style: dashed; color: var(--el-text-color-placeholder); background: var(--el-bg-color); }
.fdg-pill__user { font-weight: 400; opacity: 0.85; }
</style>
