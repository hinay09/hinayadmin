<script setup lang="ts">
/**
 * 审批表单设计器: 基于共享拖拽壳 (FormDesignerShell) 编辑 form_conf 字段列表。
 * 控件: 基础文本/数字/开关/日期 + select(静态选项) + dict(字典下拉) + relation(关联下拉) + image/file。
 * 字段标识对外保持 key (与后端 FormField/流程条件配置对齐), 壳内部 prop 经适配层映射。
 */
import { computed } from 'vue'
import type { FlowFormField } from '~/composables/useApi/flow'
import FormDesignerShell, { type DesignerField, type DesignerPaletteItem } from '~/components/FormDesignerShell.vue'

const props = defineProps<{ modelValue: FlowFormField[] }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: FlowFormField[]): void }>()

const palette: DesignerPaletteItem[] = [
  { type: 'input', label: '单行文本', hint: '常规填写' },
  { type: 'textarea', label: '多行文本', hint: '事由/说明' },
  { type: 'number', label: '数字', hint: '天数/金额' },
  { type: 'switch', label: '开关', hint: '是/否 (1/0)' },
  { type: 'select', label: '单选(静态)', hint: '固定选项' },
  { type: 'dict', label: '字典下拉', hint: '选项来自字典管理' },
  { type: 'relation', label: '关联下拉', hint: '选项来自其他模块 /all' },
  { type: 'date', label: '日期', hint: '日期选择' },
  { type: 'image', label: '图片', hint: '上传图片' },
  { type: 'file', label: '附件', hint: '上传附件' },
]

/** FlowFormField[](key) ↔ 壳 DesignerField[](prop): 壳全量不可变 emit, 映射即整体转换 */
const shellFields = computed<DesignerField[]>({
  get: () => (props.modelValue || []).map(f => ({ ...f, prop: f.key })),
  set: (list) => {
    emit('update:modelValue', list.map(({ prop, ...rest }) => ({ ...rest, key: prop }) as FlowFormField))
  },
})
</script>

<template>
  <div>
    <FormDesignerShell v-model="shellFields" :palette="palette" prop-placeholder="字段标识, 如 reason">
      <template #extra="{ field, patch }">
        <el-form-item v-if="field.type === 'select'" label="选项">
          <el-select
            :model-value="field.options || []"
            multiple
            filterable
            allow-create
            default-first-option
            placeholder="输入后回车添加选项"
            style="width:100%"
            @update:model-value="(v: string[]) => patch({ options: v })"
          />
        </el-form-item>
        <el-form-item v-if="field.type === 'dict'" label="字典编码">
          <el-input :model-value="field.dictCode" placeholder="字典管理的类型编码" @update:model-value="(v: string) => patch({ dictCode: v })" />
        </el-form-item>
        <el-form-item v-if="field.type === 'relation'" label="关联模块">
          <el-input :model-value="field.relMod" placeholder="如 post → usePostApi().all()" @update:model-value="(v: string) => patch({ relMod: v })" />
        </el-form-item>
      </template>
    </FormDesignerShell>
    <div class="fd-tip">
      字典/关联下拉的选项在发起与审批时动态加载 (字典走字典管理, 关联模块须暴露 /all 接口);
      条件分支可直接引用这些字段的值 (字典比较字典值, 关联比较 ID)。
    </div>
  </div>
</template>

<style scoped>
.fd-tip {
  margin-top: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}
</style>
