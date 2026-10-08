<script setup lang="ts">
/**
 * 流程图公共弹窗: 按实例ID取快照渲染只读流程图 (含节点运行状态着色)。
 * 业务页"流程图"按钮: <FlowDiagramDialog v-model="show" :instance-id="row.flowInstance" />
 */
import { computed, ref, watch } from 'vue'
import {
  useFlowApi, flowInstStatusMap, flowStatesFromTasks, flowEndState,
  type FlowInstanceDetail, type FlowNode,
} from '~/composables/useApi/flow'
import FlowDiagram from './FlowDiagram.vue'

defineOptions({ name: 'FlowDiagramDialog' })

const props = defineProps<{ modelValue: boolean, instanceId?: number }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const api = useFlowApi()
const visible = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v),
})

const loading = ref(false)
const detail = ref<FlowInstanceDetail | null>(null)

watch(() => [props.modelValue, props.instanceId] as const, async ([v, id]) => {
  if (!v || !id) return
  loading.value = true
  try {
    detail.value = await api.instDetail(id)
  } catch {
    detail.value = null
  } finally { loading.value = false }
})

const inst = computed(() => detail.value?.instance)
const statusText = computed(() => flowInstStatusMap[inst.value?.status || 0]?.text || '未知')
const statusTag = computed(() => flowInstStatusMap[inst.value?.status || 0]?.tag || 'info')

function parseRoot(s?: string): FlowNode | null {
  try { return s ? JSON.parse(s) as FlowNode : null } catch { return null }
}
const root = computed<FlowNode | null>(() => parseRoot(detail.value?.flowConf))
const states = computed(() => flowStatesFromTasks(detail.value?.tasks || []))
const end = computed(() => flowEndState(inst.value?.status || 0))
</script>

<template>
  <el-dialog v-model="visible" title="流程图" width="860px" top="6vh" destroy-on-close>
    <div v-loading="loading" style="min-height:120px">
      <template v-if="detail">
        <div class="fdd-summary">
          <span class="fdd-summary__title">{{ inst?.title }}</span>
          <el-tag :type="statusTag" size="small">{{ statusText }}</el-tag>
          <span class="fdd-summary__meta">{{ inst?.flowName }} · 发起人 {{ inst?.startUserName }}</span>
          <el-tag v-if="inst?.status === 1 && inst?.currentNodes" size="small" effect="plain">
            当前: {{ inst.currentNodes }}
          </el-tag>
        </div>
        <FlowDiagram :root="root" :states="states" :start-user="inst?.startUserName || ''"
          :end-state="end.state" :end-text="end.text" />
      </template>
      <el-empty v-else-if="!loading" description="暂无数据" :image-size="70" />
    </div>
  </el-dialog>
</template>

<style scoped>
.fdd-summary {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-bottom: 14px;
}
.fdd-summary__title {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.fdd-summary__meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
