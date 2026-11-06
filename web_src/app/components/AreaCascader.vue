<script setup lang="ts">
/**
 * 省市区级联选择 (公共组件)
 * - 内置省市区三级联动数据 (element-china-area-data, 文本值)
 * - v-model 绑定 ['省','市','区'] 名称数组, 库侧按名称存储可直接回显
 * - 用法: <el-form-item label="所在地区"><AreaCascader v-model="form.region" /></el-form-item>
 */
import { pcaTextArr } from 'element-china-area-data'

defineProps<{
  modelValue: string[]
  disabled?: boolean
  placeholder?: string
}>()

const emit = defineEmits<{ (e: 'update:modelValue', val: string[]): void }>()

/** el-cascader 清空时 emit null, 统一归一为空数组 */
function onChange(val: string[] | null) {
  emit('update:modelValue', val || [])
}
</script>

<template>
  <el-cascader
    :model-value="modelValue"
    :options="pcaTextArr"
    :disabled="disabled"
    :placeholder="placeholder || '请选择省 / 市 / 区'"
    filterable
    clearable
    style="width: 100%"
    @update:model-value="onChange"
  />
</template>
