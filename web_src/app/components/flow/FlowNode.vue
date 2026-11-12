<script setup lang="ts">
/**
 * 流程节点递归组件 (钉钉式画布):
 *   节点间为带箭头的竖直虚线, "+"主色圆钮悬浮在连线上, 点击弹出气泡选节点类型;
 *   审批/抄送为白底卡片 (彩色图标 + 标题 + 右侧箭头 + hover 角标删除);
 *   条件分支为横向轨道 (上下横线 + 两端出口竖线 + 列分隔线 + 蓝色条件标签)。
 * holder 指向 "child 属性指向本节点" 的宿主对象 (上级节点/分支/根), 用于插入与删除。
 */
import { inject, ref } from 'vue'
import { User, Bell, Share, Plus, Close, ArrowRight } from '@element-plus/icons-vue'
import type { FlowNode as FNode } from '~/composables/useApi/flow'

defineOptions({ name: 'FlowNode' })

const props = defineProps<{ node: FNode | null, holder: any }>()

const fd: any = inject('flowDesigner')

// 气泡可见性: 同一实例要么渲染节点段要么渲染链尾段, 单一状态即可
const addVisible = ref(false)
function pick(type: 'approver' | 'cc' | 'condition') {
  addVisible.value = false
  fd.insertNode(props.holder, props.node, type)
}
</script>

<template>
  <template v-if="node">
    <!-- 连接线 (虚线+箭头) + 插入按钮气泡 -->
    <div class="fdz-line">
      <el-popover v-model:visible="addVisible" placement="right" trigger="click" :width="232" :hide-after="0">
        <template #reference>
          <span class="fdz-plus" title="插入节点"><el-icon :size="14"><Plus /></el-icon></span>
        </template>
        <div class="fdz-menu">
          <div class="fdz-menu-item approver" @click="pick('approver')">
            <span class="fdz-menu-ico"><el-icon :size="16"><User /></el-icon></span>
            <span class="fdz-menu-txt">审批人<i>成员 / 角色 / 岗位 / 主管等</i></span>
          </div>
          <div class="fdz-menu-item cc" @click="pick('cc')">
            <span class="fdz-menu-ico"><el-icon :size="16"><Bell /></el-icon></span>
            <span class="fdz-menu-txt">抄送人<i>流程到达时通知指定人</i></span>
          </div>
          <div class="fdz-menu-item condition" @click="pick('condition')">
            <span class="fdz-menu-ico"><el-icon :size="16"><Share /></el-icon></span>
            <span class="fdz-menu-txt">条件分支<i>按表单字段路由 (含默认分支)</i></span>
          </div>
        </div>
      </el-popover>
      <i class="fdz-arrow" />
    </div>

    <!-- 审批 / 抄送节点卡片 -->
    <div
      v-if="node.type === 'approver' || node.type === 'cc'"
      class="fdz-card" :class="node.type" @click="fd.openNodeCfg(node, holder)"
    >
      <span class="fdz-del" title="删除节点" @click.stop="fd.removeNode(holder, node)">
        <el-icon :size="10"><Close /></el-icon>
      </span>
      <div class="fdz-card-head">
        <span class="fdz-card-ico">
          <el-icon :size="15"><User v-if="node.type === 'approver'" /><Bell v-else /></el-icon>
        </span>
        <span class="fdz-card-name">{{ node.name }}</span>
        <el-icon class="fdz-card-more" :size="14"><ArrowRight /></el-icon>
      </div>
      <div class="fdz-card-desc">{{ fd.nodeDesc(node) }}</div>
    </div>

    <!-- 条件分支: 横向轨道 (hover 右上角删除整个条件节点) -->
    <div v-else-if="node.type === 'condition'" class="fdz-cond">
      <span class="fdz-del cond-del" title="删除条件分支" @click.stop="fd.removeNode(holder, node)">
        <el-icon :size="10"><Close /></el-icon>
      </span>
      <div class="fdz-addcond" @click="fd.addBranch(node)">
        <el-icon :size="12"><Plus /></el-icon>添加条件
      </div>
      <div class="fdz-cols">
        <div v-for="br in node.branches" :key="br.id" class="fdz-col">
          <div class="fdz-chip" title="点击配置分支条件" @click="fd.openBranchCfg(node, br)">
            {{ br.name }}<em v-if="br.isDefault">默认</em>
            <span v-if="(node.branches?.length || 0) > 2" class="fdz-chip-del" title="删除该分支"
              @click.stop="fd.removeBranch(node, br)"><el-icon :size="8"><Close /></el-icon></span>
          </div>
          <FlowNode :node="br.child" :holder="br" />
        </div>
      </div>
    </div>

    <FlowNode :node="node.child" :holder="node" />
  </template>

  <!-- 链尾追加 (连线接到下一节点或终点) -->
  <div v-else class="fdz-line tail">
    <el-popover v-model:visible="addVisible" placement="right" trigger="click" :width="232" :hide-after="0">
      <template #reference>
        <span class="fdz-plus" title="添加节点"><el-icon :size="14"><Plus /></el-icon></span>
      </template>
      <div class="fdz-menu">
        <div class="fdz-menu-item approver" @click="pick('approver')">
          <span class="fdz-menu-ico"><el-icon :size="16"><User /></el-icon></span>
          <span class="fdz-menu-txt">审批人<i>成员 / 角色 / 岗位 / 主管等</i></span>
        </div>
        <div class="fdz-menu-item cc" @click="pick('cc')">
          <span class="fdz-menu-ico"><el-icon :size="16"><Bell /></el-icon></span>
          <span class="fdz-menu-txt">抄送人<i>流程到达时通知指定人</i></span>
        </div>
        <div class="fdz-menu-item condition" @click="pick('condition')">
          <span class="fdz-menu-ico"><el-icon :size="16"><Share /></el-icon></span>
          <span class="fdz-menu-txt">条件分支<i>按表单字段路由 (含默认分支)</i></span>
        </div>
      </div>
    </el-popover>
    <i class="fdz-arrow" />
  </div>
</template>

<style scoped>
/* ---- 连接线 (竖直虚线 + 底端箭头) ----
 * 虚线只铺到箭头上方 (background-size 裁掉底部), 箭头独立成段不与虚线粘连 */
.fdz-line {
  --fdz-c: var(--el-text-color-secondary);
  position: relative;
  width: 2px;
  height: 48px;
  margin: 0 auto;
  flex: none;
  background-image: repeating-linear-gradient(to bottom, var(--fdz-c) 0 5px, transparent 5px 10px);
  background-size: 100% calc(100% - 13px);
  background-repeat: no-repeat;
}
.fdz-arrow {
  position: absolute;
  left: 50%;
  bottom: 0;
  transform: translateX(-50%);
  width: 0;
  height: 0;
  border: 5px solid transparent;
  border-top: 7px solid var(--fdz-c);
}
/* 主色圆钮 "+": 居于虚线段中点 (避开底部箭头区, 不与箭头粘连) */
.fdz-plus {
  position: absolute;
  left: 50%;
  top: 16px;
  transform: translate(-50%, -50%);
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: var(--el-color-primary);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  user-select: none;
  box-shadow: var(--el-box-shadow-light);
  transition: transform 0.15s, box-shadow 0.15s;
}
.fdz-line:hover .fdz-plus {
  transform: translate(-50%, -50%) scale(1.18);
  box-shadow: 0 0 0 4px var(--el-color-primary-light-9), var(--el-box-shadow-light);
}

/* ---- 气泡菜单 (teleport 到 body, 画布缩放不影响) ---- */
.fdz-menu {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.fdz-menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 7px 8px;
  border-radius: 6px;
  cursor: pointer;
}
.fdz-menu-item:hover {
  background: var(--el-fill-color-light);
}
.fdz-menu-ico {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}
.fdz-menu-item.approver .fdz-menu-ico { background: var(--el-color-primary); }
.fdz-menu-item.cc .fdz-menu-ico { background: var(--el-color-warning); }
.fdz-menu-item.condition .fdz-menu-ico { background: var(--el-color-success); }
.fdz-menu-txt {
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  line-height: 1.35;
}
.fdz-menu-txt i {
  display: block;
  font-style: normal;
  font-size: 12px;
  font-weight: 400;
  color: var(--el-text-color-secondary);
}

/* ---- 审批/抄送卡片 ---- */
.fdz-card {
  position: relative;
  width: 260px;
  padding: 10px 12px 11px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
  box-shadow: var(--el-box-shadow-lighter);
  cursor: pointer;
  transition: box-shadow 0.15s, border-color 0.15s, transform 0.15s;
}
.fdz-card:hover {
  border-color: var(--el-color-primary-light-5);
  box-shadow: var(--el-box-shadow-light);
  transform: translateY(-1px);
}
.fdz-card-head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.fdz-card-ico {
  width: 24px;
  height: 24px;
  border-radius: 5px;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}
.fdz-card.approver .fdz-card-ico { background: var(--el-color-primary); }
.fdz-card.cc .fdz-card-ico { background: var(--el-color-warning); }
.fdz-card-name {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fdz-card-more {
  flex: none;
  color: var(--el-text-color-placeholder);
}
.fdz-card-desc {
  margin-top: 7px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
/* hover 角标删除 */
.fdz-del {
  position: absolute;
  top: -9px;
  right: -9px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--el-color-danger);
  color: #fff;
  display: none;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: var(--el-box-shadow-light);
  z-index: 3;
}
.fdz-card:hover .fdz-del,
.fdz-cond:hover > .fdz-del {
  display: flex;
}
.fdz-del:hover {
  opacity: 0.85;
}

/* ---- 条件分支轨道 ---- */
.fdz-cond {
  position: relative;
  padding: 0 14px;
}
.cond-del {
  top: -4px;
  right: 2px;
}
.fdz-addcond {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  margin: 0 auto 10px;
  padding: 3px 14px;
  border: 1px dashed var(--el-border-color-darker);
  border-radius: 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  user-select: none;
  background: var(--el-bg-color);
}
.fdz-addcond:hover {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary-light-5);
  border-style: solid;
}
.fdz-cols {
  position: relative;
  display: flex;
  align-items: stretch;
  border-top: 2px solid var(--el-text-color-placeholder);
  border-bottom: 2px solid var(--el-text-color-placeholder);
}
/* 两端出口竖线: 顶横线两端向下连到底横线 */
.fdz-cols::before,
.fdz-cols::after {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  width: 2px;
  background: var(--el-text-color-placeholder);
}
.fdz-cols::before { left: 0; }
.fdz-cols::after { right: 0; }
.fdz-col {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 14px 16px 16px;
  min-width: 262px;
}
.fdz-col + .fdz-col {
  border-left: 1px solid var(--el-border-color-lighter);
}
.fdz-chip {
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
  cursor: pointer;
  user-select: none;
}
.fdz-chip:hover {
  border-color: var(--el-color-primary-light-5);
}
.fdz-chip em {
  font-style: normal;
  color: var(--el-text-color-secondary);
  font-weight: 400;
}
.fdz-chip-del {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  margin-left: 3px;
  border-radius: 50%;
  background: var(--el-color-danger);
  color: #fff;
}
.fdz-chip-del:hover {
  opacity: 0.85;
}
</style>
