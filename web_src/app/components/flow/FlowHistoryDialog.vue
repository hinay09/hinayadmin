<script setup lang="ts">
/**
 * 审批历史公共弹窗: 按实例ID展示 流转记录 + 审批任务 两张表。
 * 业务页"审批历史"按钮: <FlowHistoryDialog v-model="show" :instance-id="row.flowInstance" />
 */
import { computed, ref, watch } from 'vue'
import {
  useFlowApi, flowInstStatusMap, flowActionMap, flowTaskStatusMap,
  type FlowInstanceDetail,
} from '~/composables/useApi/flow'

defineOptions({ name: 'FlowHistoryDialog' })

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

function actionText(a: string) { return flowActionMap[a] || a }
function actionTagType(a: string): string {
  if (a === 'approve' || a === 'finish') return 'success'
  if (a === 'reject' || a === 'terminate') return 'danger'
  if (a === 'resubmit' || a === 'back' || a === 'urge' || a === 'reduce') return 'warning'
  if (a === 'cancel') return 'info'
  return 'primary'
}
function taskStatusText(row: any) {
  if (row.nodeType === 2 && row.status === 2) return '已阅'
  return flowTaskStatusMap[row.status]?.text || '未知'
}
</script>

<template>
  <el-dialog v-model="visible" title="审批历史" width="860px" top="6vh" destroy-on-close>
    <div v-loading="loading">
      <template v-if="detail">
        <div class="fh-summary">
          <span class="fh-summary__title">{{ inst?.title }}</span>
          <el-tag :type="statusTag" size="small">{{ statusText }}</el-tag>
          <span class="fh-summary__meta">
            {{ inst?.flowName }} · {{ inst?.startUserName }} 发起于 {{ inst?.createdAt }}
          </span>
        </div>

        <el-tabs>
          <el-tab-pane label="流转记录">
            <el-table :data="detail.records || []" border stripe size="small" max-height="420">
              <el-table-column prop="createdAt" label="时间" width="170" />
              <el-table-column label="动作" width="110">
                <template #default="{ row }">
                  <el-tag size="small" :type="actionTagType(row.action)" effect="light">{{ actionText(row.action) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作人" width="110">
                <template #default="{ row }">{{ row.operatorId === 0 ? '系统' : row.operatorName }}</template>
              </el-table-column>
              <el-table-column prop="nodeName" label="节点" width="120" />
              <el-table-column prop="comment" label="意见" min-width="160" show-overflow-tooltip />
            </el-table>
          </el-tab-pane>
          <el-tab-pane label="审批任务">
            <el-table :data="detail.tasks || []" border stripe size="small" max-height="420">
              <el-table-column prop="nodeName" label="节点" width="120" />
              <el-table-column label="类型" width="70">
                <template #default="{ row }">
                  <el-tag size="small" effect="plain" :type="row.nodeType === 1 ? 'primary' : 'info'">
                    {{ row.nodeType === 1 ? '审批' : '抄送' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="签核" width="70">
                <template #default="{ row }">{{ row.signType === 2 ? '会签' : '或签' }}</template>
              </el-table-column>
              <el-table-column prop="assigneeName" label="处理人" width="100" />
              <el-table-column label="状态" width="90">
                <template #default="{ row }">
                  <el-tag size="small" :type="flowTaskStatusMap[row.status]?.tag || 'info'">{{ taskStatusText(row) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="comment" label="意见" min-width="140" show-overflow-tooltip />
              <el-table-column prop="receiveTime" label="到达时间" width="165" />
              <el-table-column prop="actedAt" label="处理时间" width="165" />
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </template>
      <el-empty v-else-if="!loading" description="暂无数据" :image-size="70" />
    </div>
  </el-dialog>
</template>

<style scoped>
.fh-summary {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}
.fh-summary__title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}
.fh-summary__meta {
  font-size: 12px;
  color: #909399;
}
</style>
