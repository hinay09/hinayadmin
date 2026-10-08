<script setup lang="ts">
/**
 * 审批表单渲染器: 按 form_conf 字段定义渲染动态表单。
 * 发起弹窗 (可编辑) 与详情页 (只读) 共用。
 * 图片/附件字段值统一存 FlowFileItem[] (JSON 数组), 上传走系统文件接口。
 */
import { CircleCloseFilled, Delete, Plus, UploadFilled } from '@element-plus/icons-vue'
import type { FlowFileItem, FlowFormField } from '~/composables/useApi/flow'
import { parseFileItems } from '~/composables/useApi/flow'

const props = defineProps<{
  fields: FlowFormField[]
  modelValue: Record<string, any>
  disabled?: boolean
}>()
const emit = defineEmits<{ (e: 'update:modelValue', v: Record<string, any>): void }>()

function set(key: string, v: any) {
  emit('update:modelValue', { ...props.modelValue, [key]: v })
}
function display(key: string) {
  const v = props.modelValue?.[key]
  return v === undefined || v === null ? '-' : String(v)
}
function filesOf(key: string): FlowFileItem[] {
  return parseFileItems(props.modelValue?.[key])
}

/** 上传成功后追加到字段值 (通道与重试逻辑由 FileUploader 内置) */
function onUploaded(key: string, res: any) {
  const item: FlowFileItem = { name: res.originalName || res.name, url: res.url }
  set(key, [...filesOf(key), item])
}

function removeFile(key: string, i: number) {
  set(key, filesOf(key).filter((_, idx) => idx !== i))
}
</script>

<template>
  <el-form label-width="110px" :disabled="disabled">
    <template v-for="f in fields" :key="f.key">
      <!-- 只读态: 图片缩略图 / 附件链接 / 其余统一文本 -->
      <el-form-item v-if="disabled" :label="f.label">
        <template v-if="f.type === 'image' && filesOf(f.key).length">
          <div class="fr-imgs">
            <el-image v-for="(it, i) in filesOf(f.key)" :key="i" class="fr-imgs__item" :src="it.url"
              fit="cover" :preview-src-list="filesOf(f.key).map(x => x.url)" :initial-index="i" preview-teleported />
          </div>
        </template>
        <template v-else-if="f.type === 'file' && filesOf(f.key).length">
          <div class="fr-files">
            <div v-for="(it, i) in filesOf(f.key)" :key="i" class="fr-files__row">
              <el-link type="primary" :href="it.url" target="_blank">{{ it.name }}</el-link>
            </div>
          </div>
        </template>
        <span v-else style="white-space:pre-wrap">{{ display(f.key) }}</span>
      </el-form-item>

      <el-form-item v-else :label="f.label" :required="f.required">
        <el-input-number v-if="f.type === 'number'" style="width:220px"
          :model-value="modelValue?.[f.key]" @update:model-value="(v: any) => set(f.key, v)" />
        <el-input v-else-if="f.type === 'textarea'" type="textarea" :rows="3" style="max-width:420px"
          :model-value="modelValue?.[f.key]" @update:model-value="(v: string) => set(f.key, v)" />
        <el-select v-else-if="f.type === 'select'" style="width:220px"
          :model-value="modelValue?.[f.key]" @update:model-value="(v: any) => set(f.key, v)">
          <el-option v-for="o in f.options || []" :key="o" :label="o" :value="o" />
        </el-select>
        <el-date-picker v-else-if="f.type === 'date'" type="date" value-format="YYYY-MM-DD" style="width:220px"
          :model-value="modelValue?.[f.key]" @update:model-value="(v: any) => set(f.key, v)" />

        <!-- 图片: 缩略图 + 删除 + 追加上传 -->
        <template v-else-if="f.type === 'image'">
          <div class="fr-imgs">
            <div v-for="(it, i) in filesOf(f.key)" :key="i" class="fr-imgs__box">
              <el-image class="fr-imgs__item" :src="it.url" fit="cover"
                :preview-src-list="filesOf(f.key).map(x => x.url)" :initial-index="i" preview-teleported />
              <el-icon class="fr-imgs__del" @click="removeFile(f.key, i)"><CircleCloseFilled /></el-icon>
            </div>
            <FileUploader accept="image/*" multiple :disabled="disabled" @success="(res: any) => onUploaded(f.key, res)">
              <template #default="{ uploading }">
                <div v-loading="uploading" class="fr-imgs__add"><el-icon><Plus /></el-icon></div>
              </template>
            </FileUploader>
          </div>
        </template>

        <!-- 附件: 链接列表 + 追加上传 -->
        <template v-else-if="f.type === 'file'">
          <div class="fr-files">
            <div v-for="(it, i) in filesOf(f.key)" :key="i" class="fr-files__row">
              <el-link type="primary" :href="it.url" target="_blank">{{ it.name }}</el-link>
              <el-icon class="fr-files__del" @click="removeFile(f.key, i)"><Delete /></el-icon>
            </div>
            <FileUploader multiple :disabled="disabled" @success="(res: any) => onUploaded(f.key, res)">
              <template #default="{ uploading }">
                <el-button size="small" plain :icon="UploadFilled" :loading="uploading">上传附件</el-button>
              </template>
            </FileUploader>
          </div>
        </template>

        <el-input v-else style="max-width:320px"
          :model-value="modelValue?.[f.key]" @update:model-value="(v: string) => set(f.key, v)" />
      </el-form-item>
    </template>
  </el-form>
</template>

<style scoped>
.fr-imgs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: flex-start;
}
.fr-imgs__box {
  position: relative;
}
.fr-imgs__item {
  width: 72px;
  height: 72px;
  border-radius: 6px;
  border: 1px solid var(--el-border-color-light);
  display: block;
}
.fr-imgs__del {
  position: absolute;
  top: -7px;
  right: -7px;
  font-size: 17px;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  background: var(--el-bg-color);
  border-radius: 50%;
}
.fr-imgs__del:hover { color: var(--el-color-danger); }
.fr-imgs__add {
  box-sizing: border-box;
  width: 72px;
  height: 72px;
  border: 1px dashed var(--el-text-color-disabled);
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-secondary);
  font-size: 20px;
  cursor: pointer;
}
.fr-imgs__add:hover { border-color: var(--el-color-primary); color: var(--el-color-primary); }

.fr-files {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.fr-files__row {
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: 420px;
}
.fr-files__del {
  color: var(--el-text-color-secondary);
  cursor: pointer;
  flex: none;
}
.fr-files__del:hover { color: var(--el-color-danger); }
</style>
