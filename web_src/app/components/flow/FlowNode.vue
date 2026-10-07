<script setup lang="ts">
/**
 * 流程节点递归组件 (钉钉式):
 *   节点间为竖直连接线, "+"插入按钮位于连线上;
 *   审批/抄送为带图标卡片; 条件分支为横向轨道 (上下横线 + 列分隔竖线 + 条件标签)。
 * holder 指向 "child 属性指向本节点" 的宿主对象 (上级节点/分支/根), 用于插入与删除。
 */
import { inject } from 'vue'
import { User, Bell, Delete } from '@element-plus/icons-vue'
import type { FlowNode as FNode } from '~/composables/useApi/flow'

defineOptions({ name: 'FlowNode' })

defineProps<{ node: FNode | null, holder: any }>()

const fd: any = inject('flowDesigner')
</script>

<template>
  <template v-if="node">
    <!-- 连接线 + 插入按钮 -->
    <div class="fd-link" title="插入节点" @click.stop="fd.openPlus(holder, node)">
      <span class="fd-add">+</span>
    </div>

    <!-- 审批 / 抄送节点卡片 -->
    <div
      v-if="node.type === 'approver' || node.type === 'cc'"
      class="fd-card" :class="node.type" @click="fd.openNodeCfg(node)"
    >
      <div class="fd-card-icon">
        <el-icon :size="20"><User v-if="node.type === 'approver'" /><Bell v-else /></el-icon>
      </div>
      <div class="fd-card-main">
        <div class="fd-card-title">{{ node.name }}</div>
        <div class="fd-card-desc">{{ fd.nodeDesc(node) }}</div>
      </div>
      <span class="fd-card-del" title="删除节点" @click.stop="fd.removeNode(holder, node)">
        <el-icon :size="12"><Delete /></el-icon>
      </span>
    </div>

    <!-- 条件分支: 横向轨道 (右上角悬浮删除整个条件节点) -->
    <div v-else-if="node.type === 'condition'" class="fd-cond">
      <span class="fd-card-del cond-del" title="删除条件分支" @click.stop="fd.removeNode(holder, node)">
        <el-icon :size="12"><Delete /></el-icon>
      </span>
      <div class="fd-cols">
        <div v-for="br in node.branches" :key="br.id" class="fd-col">
          <div class="fd-chip" title="点击配置分支条件" @click="fd.openBranchCfg(node, br)">
            {{ br.name }}<span v-if="br.isDefault" class="fd-def">默认</span>
            <span v-if="(node.branches?.length || 0) > 2" class="fd-chip-del" title="删除该分支"
              @click.stop="fd.removeBranch(node, br)">×</span>
          </div>
          <FlowNode :node="br.child" :holder="br" />
        </div>
        <div class="fd-col fd-col-add">
          <div class="fd-chip add" @click="fd.addBranch(node)">＋ 添加分支</div>
        </div>
      </div>
    </div>

    <FlowNode :node="node.child" :holder="node" />
  </template>

  <!-- 链尾追加 -->
  <div v-else class="fd-link" title="添加节点" @click.stop="fd.openPlus(holder, null)">
    <span class="fd-add">+</span>
  </div>
</template>

<style scoped>
/* ---- 连接线与插入按钮 ---- */
.fd-link {
  position: relative;
  width: 2px;
  height: 34px;
  background: #d5d9e0;
  margin: 0 auto;
  cursor: pointer;
}
.fd-add {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: #fff;
  border: 1px solid #c9ced8;
  color: #8f97a3;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 17px;
  line-height: 1;
  user-select: none;
  transition: all 0.15s;
}
.fd-link:hover .fd-add {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
  transform: translate(-50%, -50%) scale(1.12);
}

/* ---- 审批/抄送卡片 ---- */
.fd-card {
  position: relative;
  display: flex;
  align-items: stretch;
  width: 252px;
  background: #fff;
  border-radius: 4px;
  box-shadow: 0 1px 4px rgba(31, 41, 55, 0.1);
  cursor: pointer;
  transition: box-shadow 0.15s;
}
.fd-card:hover {
  box-shadow: 0 3px 12px rgba(31, 41, 55, 0.18);
}
.fd-card-icon {
  width: 40px;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
}
.fd-card.approver .fd-card-icon {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}
.fd-card.cc .fd-card-icon {
  background: var(--el-color-warning-light-9);
  color: var(--el-color-warning);
}
.fd-card-main {
  flex: 1;
  padding: 8px 10px;
  min-width: 0;
}
.fd-card-title {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
}
.fd-card-desc {
  font-size: 12px;
  color: #9aa1ac;
  margin-top: 3px;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
.fd-card-del {
  position: absolute;
  top: 0;
  right: 0;
  width: 20px;
  height: 20px;
  border-radius: 0 4px 0 8px;
  background: var(--el-color-danger);
  color: #fff;
  display: none;
  align-items: center;
  justify-content: center;
}
.fd-card:hover .fd-card-del {
  display: flex;
}
.fd-card-del:hover {
  opacity: 0.85;
}

/* ---- 条件分支轨道 ---- */
.fd-cond {
  position: relative;
  margin: 2px 0;
}
.cond-del {
  display: none;
}
.fd-cond:hover .cond-del {
  display: flex;
}
.fd-cols {
  display: flex;
  align-items: stretch;
  border-top: 2px solid #d5d9e0;
  border-bottom: 2px solid #d5d9e0;
  border-left: 2px solid #d5d9e0;
  border-right: 2px solid #d5d9e0;
  border-radius: 2px;
}
.fd-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 8px 14px;
  min-width: 256px;
}
.fd-col + .fd-col {
  border-left: 2px solid #d5d9e0;
}
.fd-col-add {
  justify-content: flex-start;
}
.fd-chip {
  font-size: 12px;
  color: var(--el-color-success);
  border: 1px dashed var(--el-color-success-light-5);
  background: var(--el-color-success-light-9);
  border-radius: 12px;
  padding: 3px 12px;
  margin: 2px 0 6px;
  cursor: pointer;
  user-select: none;
}
.fd-chip:hover {
  border-style: solid;
}
.fd-def {
  margin-left: 4px;
  color: #9aa1ac;
}
.fd-chip-del {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 15px;
  height: 15px;
  margin-left: 5px;
  border-radius: 50%;
  background: var(--el-color-danger);
  color: #fff;
  font-size: 11px;
  line-height: 1;
}
.fd-chip-del:hover {
  opacity: 0.85;
}
</style>
