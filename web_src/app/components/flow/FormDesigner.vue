<script setup lang="ts">
/**
 * 审批表单设计器: 编辑 form_conf 字段列表 (key/label/类型/必填/选项)。
 */
import { computed } from 'vue'
import { Delete, Plus, Top, Bottom } from '@element-plus/icons-vue'
import type { FlowFormField } from '~/composables/useApi/flow'

const props = defineProps<{ modelValue: FlowFormField[] }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: FlowFormField[]): void }>()

const fields = computed({
  get: () => props.modelValue || [],
  set: v => emit('update:modelValue', v),
})

const typeOptions = [
  { label: '单行文本', value: 'input' },
  { label: '数字', value: 'number' },
  { label: '多行文本', value: 'textarea' },
  { label: '单选', value: 'select' },
  { label: '日期', value: 'date' },
  { label: '图片', value: 'image' },
  { label: '附件', value: 'file' },
]

function add() {
  fields.value = [...fields.value, { key: `field_${fields.value.length + 1}`, label: '新字段', type: 'input', required: false }]
}
function update(i: number, patch: Partial<FlowFormField>) {
  const list = [...fields.value]
  list[i] = { ...list[i], ...patch }
  fields.value = list
}
function remove(i: number) {
  fields.value = fields.value.filter((_, idx) => idx !== i)
}
function move(i: number, dir: -1 | 1) {
  const j = i + dir
  if (j < 0 || j >= fields.value.length) return
  const list = [...fields.value]
  ;[list[i], list[j]] = [list[j], list[i]]
  fields.value = list
}
</script>

<template>
  <div>
    <el-table :data="fields" border size="small">
      <el-table-column label="字段标识 (key)" width="180">
        <template #default="{ $index, row }">
          <el-input :model-value="row.key" size="small" @update:model-value="(v: string) => update($index, { key: v })" placeholder="如 reason" />
        </template>
      </el-table-column>
      <el-table-column label="显示名" width="160">
        <template #default="{ $index, row }">
          <el-input :model-value="row.label" size="small" @update:model-value="(v: string) => update($index, { label: v })" />
        </template>
      </el-table-column>
      <el-table-column label="类型" width="130">
        <template #default="{ $index, row }">
          <el-select :model-value="row.type" size="small" @update:model-value="(v: any) => update($index, { type: v })">
            <el-option v-for="t in typeOptions" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </template>
      </el-table-column>
      <el-table-column label="选项 (单选用, 逗号分隔)" min-width="180">
        <template #default="{ $index, row }">
          <el-input v-if="row.type === 'select'" :model-value="(row.options || []).join(',')" size="small"
            placeholder="如: 事假,病假,年假"
            @update:model-value="(v: string) => update($index, { options: v.split(',').map(s => s.trim()).filter(Boolean) })" />
          <span v-else style="color:#999">-</span>
        </template>
      </el-table-column>
      <el-table-column label="必填" width="70" align="center">
        <template #default="{ $index, row }">
          <el-switch :model-value="!!row.required" size="small" @update:model-value="(v: any) => update($index, { required: !!v })" />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="130" align="center">
        <template #default="{ $index }">
          <el-button link type="primary" size="small" :icon="Top" :disabled="$index === 0" @click="move($index, -1)" />
          <el-button link type="primary" size="small" :icon="Bottom" :disabled="$index === fields.length - 1" @click="move($index, 1)" />
          <el-button link type="danger" size="small" :icon="Delete" @click="remove($index)" />
        </template>
      </el-table-column>
    </el-table>
    <el-button style="margin-top:10px" type="primary" plain :icon="Plus" size="small" @click="add">添加字段</el-button>
  </div>
</template>
