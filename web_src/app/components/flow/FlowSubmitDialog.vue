<script setup lang="ts">
/**
 * 提交审批公共弹窗:
 *  - 人工发起: 不传 handler, 直接 api.instStart (无业务关联, bizId=0);
 *  - 业务发起: 传 flowKey + handler, 审批人/表单选择结果交业务 API (后端 StartForBiz);
 *    业务单表单由页面自己维护时传 :show-form="false" 仅选审批人, :show-title="false" 隐藏标题。
 * 用法见 docs/pro/flow-frontend.md。
 */
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  useFlowApi, collectSelfSelectNodes, isEmptyFormValue, type FlowSubmitPayload,
  type FlowDefinitionItem, type FlowFormField, type FlowNode, type FlowUserOption,
} from '~/composables/useApi/flow'
import FormRender from './FormRender.vue'

defineOptions({ name: 'FlowSubmitDialog' })

const props = withDefaults(defineProps<{
  modelValue: boolean
  /** 业务流程标识: 传入则自动选中对应已发布定义, 不再显示流程下拉 */
  flowKey?: string
  /** 业务自定义提交: 接收完整发起参数, 返回实例ID (可空, 用于跳转详情) */
  handler?: (payload: FlowSubmitPayload) => Promise<number | void>
  showForm?: boolean
  showTitle?: boolean
  title?: string
  formData?: Record<string, any>
}>(), {
  flowKey: '',
  showForm: true,
  showTitle: true,
  title: '',
  formData: undefined,
})

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'submitted', instanceId: number): void
}>()

const api = useFlowApi()
const visible = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v),
})

const loading = ref(false)
const submitting = ref(false)
const defs = ref<FlowDefinitionItem[]>([])
const users = ref<FlowUserOption[]>([])
const form = reactive({
  defId: undefined as number | undefined,
  title: '',
  data: {} as Record<string, any>,
  selfSelects: {} as Record<string, number[]>,
})

const currentDef = computed(() => defs.value.find(d => d.id === form.defId))
const fields = computed<FlowFormField[]>(() => parseSafe(currentDef.value?.formConf || '', []))
const root = computed<FlowNode | null>(() => parseSafe<FlowNode | null>(currentDef.value?.flowConf || '', null))
const selfSelectNodes = computed(() => collectSelfSelectNodes(root.value))

function parseSafe<T>(s: string, fallback: T): T {
  try { return s ? JSON.parse(s) as T : fallback } catch { return fallback }
}

watch(() => props.modelValue, async (v) => {
  if (!v) return
  form.title = props.title
  form.data = { ...(props.formData || {}) }
  form.selfSelects = {}
  form.defId = undefined
  // 每次打开都重新拉取: 流程定义可能刚发布/停用, 用缓存列表会选中已失效版本
  loading.value = true
  try {
    const [res, opt] = await Promise.all([api.defUsable(), api.options()])
    defs.value = res.list || []
    users.value = opt.users || []
  } finally { loading.value = false }
  if (props.flowKey) {
    // 业务流程: 绑定 flowKey 对应定义 (可发起列表即最新发布版)
    const hit = defs.value.find(d => d.flowKey === props.flowKey)
    if (!hit) ElMessage.warning(`未找到流程标识「${props.flowKey}」的已发布流程`)
    form.defId = hit?.id
  }
})

function onDefChange() {
  form.data = { ...(props.formData || {}) }
  form.selfSelects = {}
}

async function submit() {
  if (!form.defId) { ElMessage.warning('请选择流程'); return }
  if (props.showTitle && !form.title.trim()) { ElMessage.warning('请输入申请标题'); return }
  if (props.showForm) {
    for (const f of fields.value) {
      // isEmptyFormValue: 图片/附件字段值为数组, 空数组也算未填写 (与后端同口径)
      if (f.required && isEmptyFormValue(form.data[f.key])) {
        ElMessage.warning(`请填写「${f.label}」`)
        return
      }
    }
  }
  for (const n of selfSelectNodes.value) {
    if (!form.selfSelects[n.id]?.length) {
      ElMessage.warning(`请在「${n.name}」中选择审批人`)
      return
    }
  }
  const payload: FlowSubmitPayload = {
    definitionId: form.defId,
    title: form.title.trim(),
    formData: form.data,
    selfSelects: form.selfSelects,
  }
  submitting.value = true
  try {
    let instanceId = 0
    if (props.handler) instanceId = (await props.handler(payload)) || 0
    else instanceId = (await api.instStart(payload)).id
    ElMessage.success('已提交审批')
    emit('submitted', instanceId)
    visible.value = false
  } catch {} finally { submitting.value = false }
}
</script>

<template>
  <el-dialog v-model="visible" title="提交审批" width="560px" top="8vh" destroy-on-close>
    <el-form v-loading="loading" label-width="90px">
      <el-form-item v-if="!flowKey" label="选择流程" required>
        <el-select v-model="form.defId" filterable placeholder="选择已发布的流程" style="width:100%" @change="onDefChange">
          <el-option v-for="d in defs" :key="d.id" :label="`${d.name} (v${d.version})`" :value="d.id" />
        </el-select>
      </el-form-item>
      <el-form-item v-else label="流程">
        <el-tag>{{ currentDef?.name || flowKey }}</el-tag>
        <span style="margin-left:8px;font-size:12px;color:var(--el-text-color-secondary)">v{{ currentDef?.version || '-' }}</span>
      </el-form-item>
      <el-form-item v-if="showTitle" label="申请标题" required>
        <el-input v-model="form.title" maxlength="120" placeholder="如: 请假 3 天" />
      </el-form-item>

      <template v-if="showForm && currentDef">
        <el-divider content-position="left">申请表单</el-divider>
        <FormRender v-model="form.data" :fields="fields" />
      </template>

      <template v-if="selfSelectNodes.length">
        <el-divider content-position="left">选择审批人</el-divider>
        <el-form-item v-for="n in selfSelectNodes" :key="n.id" :label="n.name" required>
          <el-select v-model="form.selfSelects[n.id]" multiple filterable placeholder="选择审批人" style="width:100%">
            <el-option v-for="u in users" :key="u.id" :label="u.nickname || u.username" :value="u.id" />
          </el-select>
        </el-form-item>
      </template>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <!-- 不挂权限指令: 人工模式的入口页自行用 flow:instance:start 控制, 业务模式由业务权限控制 -->
      <el-button type="primary" :loading="submitting" @click="submit">提交审批</el-button>
    </template>
  </el-dialog>
</template>
