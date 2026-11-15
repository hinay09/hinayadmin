<script lang="ts">
/** 设计器字段: prop 为标识 (gencode 直用; flow 侧适配 key), 其余按控件附加 */
export interface DesignerField {
  prop: string
  label: string
  type: string
  required: boolean
  /** 控件专属属性原样挂在字段上 (dictCode/relMod/options 等), patch 时保留 */
  [k: string]: any
}

/** 组件库项 */
export interface DesignerPaletteItem {
  type: string
  label: string
  hint?: string
}
</script>

<script setup lang="ts">
/**
 * 表单设计器壳 (拖拽三栏): 左组件库 → 中画布(拖入/排序/上下移/删除) → 右属性面板。
 * 代码生成表单设计器与审批流程表单设计器共用; 控件清单经 palette 传入,
 * 控件专属属性经 #extra="{ field, patch }" 插槽渲染 (patch 回写保留其余属性)。
 *
 * 数据流: 全量不可变 —— 任何修改都 emit 全新数组 (update:modelValue),
 * 不就地改动传入对象, 便于外层做字段名适配 (如 flow 的 key ↔ prop)。
 */
import { ref, computed } from 'vue'
import { Delete, Top, Bottom, Plus } from '@element-plus/icons-vue'

const props = defineProps<{
  modelValue: DesignerField[]
  palette: DesignerPaletteItem[]
  /** 标识输入框文案 */
  propPlaceholder?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: DesignerField[]): void
}>()

const selected = ref(-1)
const current = computed(() => (selected.value >= 0 ? props.modelValue[selected.value] : null))

/** 通用补丁: 替换选中/指定项 (浅拷贝合并, 保留控件专属属性) */
function patch(index: number, p: Partial<DesignerField>) {
  const list = props.modelValue.map((f, i) => (i === index ? { ...f, ...p } : f))
  emit('update:modelValue', list)
}

const patchSelected = (p: Partial<DesignerField>) => {
  if (selected.value >= 0) patch(selected.value, p)
}

function newField(type: string): DesignerField {
  const def = props.palette.find(x => x.type === type)
  return { prop: `field_${props.modelValue.length + 1}`, label: def?.label || type, type, required: false }
}

function insertField(type: string, at?: number) {
  const list = [...props.modelValue]
  const idx = at === undefined ? list.length : at
  list.splice(idx, 0, newField(type))
  emit('update:modelValue', list)
  selected.value = idx
}

function removeField(index: number) {
  const list = props.modelValue.filter((_, i) => i !== index)
  emit('update:modelValue', list)
  if (selected.value >= list.length) selected.value = list.length - 1
}

function moveField(from: number, to: number) {
  if (from < 0 || from >= props.modelValue.length || to < 0 || to >= props.modelValue.length) return
  const list = [...props.modelValue]
  const [item] = list.splice(from, 1)
  list.splice(to, 0, item)
  emit('update:modelValue', list)
  selected.value = to
}

/* ---- 拖拽 (HTML5 原生: 组件库拖入追加, 画布卡片拖动排序) ---- */
let dragType = '' // 组件库拖出的控件类型
let dragIndex = -1 // 画布内被拖动卡片的下标

function onPaletteDragStart(type: string, e: DragEvent) {
  dragType = type
  dragIndex = -1
  e.dataTransfer!.effectAllowed = 'copy'
  e.dataTransfer!.setData('text/plain', type)
}

function onCardDragStart(index: number, e: DragEvent) {
  dragIndex = index
  dragType = ''
  e.dataTransfer!.effectAllowed = 'move'
  e.dataTransfer!.setData('text/plain', String(index))
}

function onCanvasDrop(e: DragEvent) {
  e.preventDefault()
  if (dragType) {
    insertField(dragType)
    dragType = ''
  }
  else if (dragIndex >= 0) {
    moveField(dragIndex, props.modelValue.length - 1)
    dragIndex = -1
  }
}

function onCardDrop(index: number, e: Event) {
  e.preventDefault()
  if (dragType) {
    insertField(dragType, index)
    dragType = ''
  }
  else if (dragIndex >= 0 && dragIndex !== index) {
    moveField(dragIndex, index)
    dragIndex = -1
  }
}
</script>

<template>
  <div class="fdshell">
    <!-- 左: 组件库 -->
    <div class="fdshell-palette">
      <div class="fdshell-panel-title">组件库</div>
      <div
        v-for="p in palette"
        :key="p.type"
        class="fdshell-palette-item"
        draggable="true"
        @dragstart="onPaletteDragStart(p.type, $event)"
      >
        <div class="fdshell-palette-label">{{ p.label }}</div>
        <div v-if="p.hint" class="fdshell-palette-hint">{{ p.hint }}</div>
      </div>
    </div>

    <!-- 中: 画布 -->
    <div class="fdshell-canvas">
      <div class="fdshell-panel-title canvas-title">
        <span>画布 ({{ modelValue.length }} 个字段)</span>
        <el-button size="small" :icon="Plus" @click="insertField(palette[0]?.type || 'input')">追加字段</el-button>
      </div>
      <div class="fdshell-drop" @dragover.prevent @drop="onCanvasDrop">
        <div
          v-for="(f, i) in modelValue"
          :key="`${f.prop}-${i}`"
          class="fdshell-card"
          :class="{ active: selected === i }"
          draggable="true"
          @click="selected = i"
          @dragstart="onCardDragStart(i, $event)"
          @dragover.prevent
          @drop="onCardDrop(i, $event)"
        >
          <div class="fdshell-card-main">
            <span class="fdshell-card-label">{{ f.label }}</span>
            <el-tag size="small" type="info">{{ f.type }}</el-tag>
            <el-tag v-if="f.required" size="small" type="danger">必填</el-tag>
          </div>
          <code class="fdshell-card-prop">{{ f.prop }}</code>
          <div class="fdshell-card-actions" @click.stop>
            <el-button link size="small" :icon="Top" :disabled="i === 0" @click="moveField(i, i - 1)" />
            <el-button link size="small" :icon="Bottom" :disabled="i === modelValue.length - 1" @click="moveField(i, i + 1)" />
            <el-button link size="small" type="danger" :icon="Delete" @click="removeField(i)" />
          </div>
        </div>
        <div v-if="!modelValue.length" class="fdshell-empty">从左侧拖入组件</div>
      </div>
    </div>

    <!-- 右: 属性面板 -->
    <div class="fdshell-props">
      <div class="fdshell-panel-title">字段属性</div>
      <el-empty v-if="!current" description="点击画布字段配置" :image-size="50" />
      <el-form v-else label-width="72px" size="default" @submit.prevent>
        <el-form-item label="标识">
          <el-input
            :model-value="current.prop"
            :placeholder="propPlaceholder || '字段标识'"
            @update:model-value="(v: string) => patchSelected({ prop: v })"
          />
        </el-form-item>
        <el-form-item label="标签">
          <el-input
            :model-value="current.label"
            placeholder="显示名"
            @update:model-value="(v: string) => patchSelected({ label: v })"
          />
        </el-form-item>
        <el-form-item label="控件">
          <el-select :model-value="current.type" @update:model-value="(v: string) => patchSelected({ type: v })">
            <el-option v-for="p in palette" :key="p.type" :value="p.type" :label="p.label" />
          </el-select>
        </el-form-item>
        <!-- 控件专属属性 (dictCode/relMod/options...) -->
        <slot name="extra" :field="current" :patch="patchSelected" />
        <el-form-item label="必填">
          <el-switch :model-value="current.required" @update:model-value="(v: boolean) => patchSelected({ required: v })" />
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<style scoped>
.fdshell {
  display: grid;
  grid-template-columns: 170px 1fr 260px;
  gap: 10px;
  align-items: start;
}

.fdshell-panel-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 8px;
}

.canvas-title {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.fdshell-palette-item {
  padding: 8px 10px;
  border: 1px dashed var(--el-border-color);
  border-radius: 6px;
  margin-bottom: 8px;
  cursor: grab;
  user-select: none;
  transition: border-color 0.15s;
}

.fdshell-palette-item:hover {
  border-color: var(--el-color-primary);
}

.fdshell-palette-label {
  font-size: 13px;
}

.fdshell-palette-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.fdshell-drop {
  min-height: 320px;
  border: 1px dashed var(--el-border-color);
  border-radius: 6px;
  padding: 8px;
}

.fdshell-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  margin-bottom: 8px;
  cursor: pointer;
  background: var(--el-bg-color);
}

.fdshell-card:hover {
  border-color: var(--el-color-primary-light-5);
}

.fdshell-card.active {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 1px var(--el-color-primary-light-7);
}

.fdshell-card-main {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
}

.fdshell-card-label {
  font-size: 13px;
}

.fdshell-card-prop {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.fdshell-card-actions {
  display: flex;
  align-items: center;
}

.fdshell-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  color: var(--el-text-color-placeholder);
}

@media (max-width: 1100px) {
  .fdshell {
    grid-template-columns: 1fr;
  }
}
</style>
