<script setup lang="ts">
/**
 * 公共文件上传组件。
 *
 * 包装 el-upload 仅作文件选择器 (关闭自动上传), 统一走 useFileApi().upload():
 * S3 兼容存储默认预签名直传、本地存储/预签名失败自动回落服务端中转。
 * 头像等自有后端通道的场景用 uploadFn 替换默认上传函数。
 *
 * 业务失败提示由 useRequest 统一弹出, 本组件不重复提示;
 * uploading 状态经作用域插槽抛出, 供触发器绑定 loading。
 */
import { ref } from 'vue'
import type { UploadFile } from 'element-plus'
import { useFileApi } from '~/composables/useApi'

defineOptions({ name: 'FileUploader' })

const props = defineProps<{
  /** 文件选择器过滤, 如 "image/*" 或 ".xlsx" (仅过滤选择, 校验以后端为准) */
  accept?: string
  multiple?: boolean
  disabled?: boolean
  /** 自定义上传通道 (默认走文件管理上传), 返回值原样透传给 success 事件 */
  uploadFn?: (file: File) => Promise<any>
}>()

const emit = defineEmits<{
  (e: 'success', res: any, file: File): void
}>()

const api = useFileApi()
const uploading = ref(false)

async function onChange(f: UploadFile) {
  // auto-upload 关闭时仅 "选中" 一次触发 ready, 状态守卫防重复
  if (f.status !== 'ready' || !f.raw || uploading.value) return
  uploading.value = true
  try {
    const res = props.uploadFn ? await props.uploadFn(f.raw) : await api.upload(f.raw)
    emit('success', res, f.raw)
  }
  finally {
    uploading.value = false
  }
}
</script>

<template>
  <el-upload
    :show-file-list="false"
    :auto-upload="false"
    :accept="accept"
    :multiple="multiple"
    :disabled="disabled || uploading"
    :on-change="onChange"
  >
    <slot :uploading="uploading" />
  </el-upload>
</template>
