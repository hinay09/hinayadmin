<script lang="ts">
/** 列定义 (script setup 不允许 export, 单独普通 script 块导出类型) */
export interface ProColumn {
  prop: string
  label: string
  width?: number | string
  minWidth?: number | string
  /** 0/1 状态列渲染 el-tag */
  tag?: boolean
  /** 字典类型编码: 值 -> 字典文案 */
  dict?: string
  /** 时间列 (固定宽度) */
  time?: boolean
  /** 值为多选标签数组 */
  tags?: boolean
  /** 自定义列: 以该名渲染具名插槽 <template #xxx="{ row }"> */
  slot?: string
}
</script>

<script setup lang="ts" generic="T extends { id: number }">
/**
 * 通用表格: 收编 el-table + 分页 + 多选 + 加载态样板。
 *
 * 用法:
 *   <ProTable ref="tableRef" :fetch="api.list" :columns="cols" :query="query" selectable row-key="id">
 *     <template #toolbar><el-button 新增 /></template>
 *     <template #actions="{ row }"><el-button 编辑 /></template>
 *   </ProTable>
 *
 * 列定义 cols:
 *   { prop, label, width?, minWidth?, tag?, dict?, time? }
 *   - tag: 0/1 渲染 启用/禁用 el-tag (状态列)
 *   - dict: 字典类型编码, 值经 useDictOptions 映射文案 (字典列)
 *   - time: 按时间列宽渲染
 * 暴露: reload() 重新加载(保持页码), selection 当前选中行, clearSelection()
 */
import type { TableInstance } from 'element-plus'

const props = withDefaults(defineProps<{
  /** 列表请求函数 (自动带上 query 与分页参数) */
  fetch: (params: any) => Promise<{ list: T[]; total: number }>
  columns: ProColumn[]
  /** 查询条件对象 (search 前由 QueryForm 重置 page) */
  query?: Record<string, any>
  selectable?: boolean
  rowKey?: string
  pageSizes?: number[]
  /** 操作列宽度 (0 隐藏操作列) */
  actionWidth?: number
  /** 默认页大小 */
  pageSize?: number
  /** 是否渲染创建时间列 (默认渲染) */
  showCreatedAt?: boolean
}>(), {
  rowKey: 'id',
  pageSizes: () => [10, 20, 50],
  actionWidth: 150,
  pageSize: 10,
  showCreatedAt: true,
})

const emit = defineEmits<{
  (e: 'selection-change', rows: T[]): void
}>()

const loading = ref(false)
const list = ref<T[]>([]) as Ref<T[]>
const total = ref(0)
const page = ref(1)
const size = ref(props.pageSize)
const tableRef = ref<TableInstance>()
const selection = ref<T[]>([]) as Ref<T[]>

const { label: dictLabel } = useDictOptions()

async function reload() {
  loading.value = true
  try {
    const params = { ...(props.query || {}), page: page.value, pageSize: size.value }
    const res = await props.fetch(params)
    list.value = res.list || []
    total.value = res.total || 0
    // 数据变更后选中集失效, 主动清空避免残留
    clearSelection()
  }
  finally {
    loading.value = false
  }
}

function clearSelection() {
  tableRef.value?.clearSelection()
  selection.value = []
}

function onSelectionChange(rows: any[]) {
  selection.value = rows
  emit('selection-change', rows)
}

function onPageChange() {
  reload()
}

/** 搜索入口: 重置页码后加载 */
function search() {
  page.value = 1
  reload()
}

onMounted(reload)

defineExpose({ reload, search, selection, clearSelection })
</script>

<template>
  <div>
    <slot name="toolbar" />

    <el-table
      ref="tableRef"
      v-loading="loading"
      :data="list"
      :row-key="rowKey"
      border
      stripe
      style="margin-top:8px"
      @selection-change="onSelectionChange"
    >
      <el-table-column v-if="selectable" type="selection" width="45" fixed="left" />
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column
        v-for="col in columns"
        :key="col.prop"
        :prop="col.prop"
        :label="col.label"
        :width="col.time ? 170 : col.width"
        :min-width="col.width || col.time ? undefined : (col.minWidth || 120)"
      >
        <template v-if="col.tag || col.dict || col.tags || col.slot" #default="{ row }">
          <el-tag v-if="col.tag" :type="row[col.prop] === 1 ? 'success' : 'info'">
            {{ row[col.prop] === 1 ? '启用' : '禁用' }}
          </el-tag>
          <template v-else-if="col.tags">
            <el-tag v-for="t in row[col.prop] || []" :key="t" size="small" style="margin-right:4px">{{ t }}</el-tag>
          </template>
          <slot v-else-if="col.slot" :name="col.slot" :row="row" />
          <span v-else>{{ dictLabel(col.dict!, row[col.prop]) }}</span>
        </template>
      </el-table-column>
      <el-table-column v-if="showCreatedAt" prop="createdAt" label="创建时间" width="170" />
      <el-table-column v-if="actionWidth > 0 && $slots.actions" label="操作" :width="actionWidth" fixed="right">
        <template #default="{ row }">
          <slot name="actions" :row="row" />
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      v-model:current-page="page"
      v-model:page-size="size"
      :total="total"
      :page-sizes="pageSizes"
      layout="total, sizes, prev, pager, next, jumper"
      style="margin-top:16px;justify-content:flex-end"
      @current-change="onPageChange"
      @size-change="onPageChange"
    />
  </div>
</template>
