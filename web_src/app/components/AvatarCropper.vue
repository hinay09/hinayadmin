<script setup lang="ts">
/**
 * 头像裁剪上传 (公共组件)
 * - 默认插槽 = 触发器(如头像); 点击后选择图片 -> 弹窗裁剪(1:1 方形裁剪框 + 圆形实时预览) -> 确认后调用 uploadFn 即时上传
 * - uploadFn/success 接口语义与 FileUploader 对齐, 头像入口可直接替换
 * - 裁剪输出 512x512 PNG (透明背景保留), 远小于后端 /auth/avatar 的 2MB 限制
 */
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { ElMessage, type UploadFile } from 'element-plus'
import { ZoomIn, ZoomOut, RefreshLeft, RefreshRight, Refresh, UploadFilled } from '@element-plus/icons-vue'
import Cropper from 'cropperjs'
import 'cropperjs/dist/cropper.css'

const props = defineProps<{
  /** 上传通道: 裁剪确认后以 PNG File 调用, 返回值原样透传给 success 事件 */
  uploadFn: (file: File) => Promise<any>
  /** 文件选择器过滤, 如 "image/*" (仅过滤选择, 校验以后端为准) */
  accept?: string
  disabled?: boolean
}>()

const emit = defineEmits<{ (e: 'success', res: any, file: File): void }>()

/** 裁剪源图类型白名单 (与 /auth/avatar 后端校验同源) */
const ACCEPT_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp']
/** 裁剪源图大小上限 (仅约束送入裁剪的原图, 最终上传的是压缩后的 512x512 PNG) */
const MAX_SOURCE_SIZE = 10 * 1024 * 1024

const dialogVisible = ref(false)
const submitting = ref(false)
const imgRef = ref<HTMLImageElement>()
const previewRef = ref<HTMLDivElement>()

let cropper: Cropper | null = null
let sourceUrl = ''

/** el-upload 仅作选择器 (auto-upload=false), 选中即进入裁剪弹窗 */
function onFileChange(f: UploadFile) {
  if (f.status !== 'ready' || !f.raw) return
  if (!ACCEPT_TYPES.includes(f.raw.type)) {
    ElMessage.warning('仅支持 JPG / PNG / GIF / WEBP 格式')
    return
  }
  if (f.raw.size > MAX_SOURCE_SIZE) {
    ElMessage.warning('图片大小不能超过 10MB')
    return
  }
  destroyCropper()
  sourceUrl = URL.createObjectURL(f.raw)
  dialogVisible.value = true
}

/** 弹窗打开后初始化裁剪器 (img/preview DOM 均已挂载) */
watch(dialogVisible, async (visible) => {
  if (visible) {
    await nextTick()
    if (!imgRef.value) return
    cropper = new Cropper(imgRef.value, {
      aspectRatio: 1,
      viewMode: 1,
      dragMode: 'move',
      autoCropArea: 1,
      background: false,
      preview: previewRef.value,
    })
  }
  else {
    destroyCropper()
  }
})

function destroyCropper() {
  cropper?.destroy()
  cropper = null
  if (sourceUrl) {
    URL.revokeObjectURL(sourceUrl)
    sourceUrl = ''
  }
}

function zoomIn() {
  cropper?.zoom(0.1)
}
function zoomOut() {
  cropper?.zoom(-0.1)
}
function rotateLeft() {
  cropper?.rotate(-90)
}
function rotateRight() {
  cropper?.rotate(90)
}
function resetCrop() {
  cropper?.reset()
}

/** 确认裁剪: 输出 512x512 PNG 并走上传通道, 成功后关闭弹窗 */
function handleConfirm() {
  if (!cropper) return
  const canvas = cropper.getCroppedCanvas({
    width: 512,
    height: 512,
    imageSmoothingEnabled: true,
    imageSmoothingQuality: 'high',
  })
  if (!canvas) {
    ElMessage.warning('裁剪失败, 请重试')
    return
  }
  canvas.toBlob(async (blob) => {
    if (!blob) {
      ElMessage.warning('裁剪失败, 请重试')
      return
    }
    const file = new File([blob], `avatar-${Date.now()}.png`, { type: 'image/png' })
    submitting.value = true
    try {
      const res = await props.uploadFn(file)
      emit('success', res, file)
      dialogVisible.value = false
    }
    catch {}
    finally {
      submitting.value = false
    }
  }, 'image/png')
}

onBeforeUnmount(() => {
  destroyCropper()
})
</script>

<template>
  <span class="avatar-cropper">
    <el-upload
      :accept="accept || ACCEPT_TYPES.join(',')"
      :disabled="disabled"
      :auto-upload="false"
      :show-file-list="false"
      class="cropper-picker"
      @change="onFileChange"
    >
      <slot />
    </el-upload>

    <!-- 裁剪弹窗: 左侧裁剪画布, 右侧圆形实时预览 -->
    <el-dialog
      v-model="dialogVisible"
      title="裁剪头像"
      width="640px"
      :close-on-click-modal="false"
      append-to-body
    >
      <div v-if="dialogVisible" class="crop-body">
        <div class="crop-canvas-wrap">
          <img ref="imgRef" :src="sourceUrl" alt="待裁剪头像">
        </div>
        <div class="crop-preview-col">
          <div ref="previewRef" class="crop-preview" />
          <div class="crop-preview-tip">头像预览 (圆形展示)</div>
        </div>
      </div>
      <div class="crop-toolbar">
        <el-button :icon="ZoomIn" circle title="放大" @click="zoomIn" />
        <el-button :icon="ZoomOut" circle title="缩小" @click="zoomOut" />
        <el-button :icon="RefreshLeft" circle title="向左旋转" @click="rotateLeft" />
        <el-button :icon="RefreshRight" circle title="向右旋转" @click="rotateRight" />
        <el-button :icon="Refresh" circle title="重置" @click="resetCrop" />
        <span class="crop-toolbar-tip">支持滚轮缩放 / 拖动图片 / 拖动裁剪框边角调整</span>
      </div>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :icon="UploadFilled" :loading="submitting" @click="handleConfirm">
          确认上传
        </el-button>
      </template>
    </el-dialog>
  </span>
</template>

<style scoped>
.avatar-cropper {
  display: inline-block;
}
.cropper-picker :deep(.el-upload) {
  display: inline-block;
  cursor: pointer;
}
.crop-body {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}
.crop-canvas-wrap {
  flex: 1;
  min-width: 0;
  height: 320px;
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  background: var(--el-fill-color-light);
}
.crop-canvas-wrap img {
  display: block;
  max-width: 100%;
  visibility: hidden; /* cropperjs 以 img 为源重建画布, 隐藏原图防闪现 */
}
.crop-preview-col {
  width: 140px;
  flex-shrink: 0;
  text-align: center;
}
.crop-preview {
  width: 140px;
  height: 140px;
  border-radius: 50%;
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-light);
}
.crop-preview :deep(img) {
  width: 100%;
  display: block;
}
.crop-preview-tip {
  margin-top: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.crop-toolbar {
  margin-top: 12px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.crop-toolbar-tip {
  margin-left: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
