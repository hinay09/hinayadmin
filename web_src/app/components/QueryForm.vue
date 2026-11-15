<script setup lang="ts">
import { Search, Refresh } from '@element-plus/icons-vue'
/**
 * 查询表单容器: 收编 inline 表单 + 查询/重置按钮 + 回车搜索样板。
 *
 * 用法:
 *   <QueryForm :query="query" @search="loadList" @reset="resetQuery">
 *     <el-form-item label="关键词">...</el-form-item>
 *     <el-form-item><el-button 新增/></el-form-item>  <!-- 操作按钮混排也可 -->
 *   </QueryForm>
 *
 * search 事件触发时自动把 page 重置为 1 (query 含 page 字段时)。
 */
const props = defineProps<{
  /** 页面查询对象 (含 page/pageSize 时, search 前自动重置 page=1) */
  query?: Record<string, any>
}>()

const emit = defineEmits<{
  (e: 'search'): void
  (e: 'reset'): void
}>()

function onSearch() {
  if (props.query && 'page' in props.query)
    props.query.page = 1
  emit('search')
}

function onReset() {
  emit('reset')
}
</script>

<template>
  <el-form inline @submit.prevent>
    <slot />
    <el-form-item>
      <el-button type="primary" :icon="Search" @click="onSearch">查询</el-button>
      <el-button :icon="Refresh" @click="onReset">重置</el-button>
    </el-form-item>
  </el-form>
</template>
