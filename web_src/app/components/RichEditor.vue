<script setup lang="ts">
/**
 * wangeditor 富文本编辑器封装 (内容管理 CMS)。
 *
 * - 仅客户端渲染: wangeditor 依赖浏览器 DOM, 组件在 onMounted 动态加载
 *   编辑器与样式 (SSR 阶段零求值), Nuxt 水合安全;
 * - v-model 双向绑定 HTML 字符串。数据流必须顺着包装层的约定 (关键!):
 *   编辑器变更 → 包装层 emit update:modelValue(html) → 本组件 html → 上抛父级;
 *   外部赋值 (详情加载) → html 变化 → 包装层内部 watch 调 setHtml 回灌。
 *   绝不在编辑器产出之外改写 html 值 —— 包装层用内部 curValue 比对 v-model,
 *   不一致即 setHtml 重置文档, 曾因空文档归一化成 '' 造成输入被清空的死循环;
 * - 图片/视频上传统一走 useFileApi().upload (S3 预签名直传优先, 失败回落服务端),
 *   插入时同时携带 src (当下展示 URL) 与 data-ref (持久引用) —— 后端入库把 src
 *   归一为 data-ref, 回显时再换回展示 URL, 预签名过期/切换存储不腐蚀内容;
 * - 上传失败提示由 useRequest 统一弹出, 本组件静默;
 * - 暗黑模式: 通过 Element Plus CSS 变量覆盖 wangeditor 亮色默认样式。
 */
import { onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { useFileApi } from '~/composables/useApi'

defineOptions({ name: 'RichEditor' })

const props = withDefaults(defineProps<{
  modelValue: string
  /** 编辑区高度 (px) */
  height?: number
  placeholder?: string
}>(), { height: 420, placeholder: '请输入内容...' })

const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>()

// 动态加载的组件引用 (客户端异步注入, 模板 <component :is> 渲染)
const EditorComp = ref<any>(null)
const ToolbarComp = ref<any>(null)
// 编辑器实例 (官方要求用 shallowRef 避免 deep 响应式代理拖垮 slate 内部数据结构)
const editorRef = shallowRef<any>(null)
// 与包装层 v-model 绑定的唯一 HTML 状态: 只承接包装层 emit 与外部赋值, 不做任何归一化
const html = ref(props.modelValue || '')
const loading = ref(true)

onMounted(async () => {
  const [{ Editor, Toolbar }] = await Promise.all([
    import('@wangeditor/editor-for-vue'),
    // 编辑器样式随组件异步分块加载, 不进全局 CSS
    import('@wangeditor/editor/dist/css/style.css'),
  ]) as any
  EditorComp.value = Editor
  ToolbarComp.value = Toolbar
})

// 组件销毁时销毁编辑器实例 (官方要求, 防内存泄漏)
onBeforeUnmount(() => {
  const ed = editorRef.value
  editorRef.value = null
  if (ed) ed.destroy()
})

// 外部值流入 (如编辑页详情加载完成): 只改 html, 由包装层内部 watch 调 setHtml 回灌;
// 编辑器自身变更的回声 (父级原值流回) 被 html 比对挡下
watch(() => props.modelValue, (v) => {
  const val = v || ''
  if (val !== html.value) html.value = val
})

// 编辑器变更上抛父级 (html 此刻就是包装层刚 emit 的原值, 与其 curValue 一致)
watch(html, v => emit('update:modelValue', v))

const fileApi = useFileApi()

const toolbarConfig = { excludeKeys: [] }

const editorConfig = {
  placeholder: props.placeholder,
  MENU_CONF: {
    // 图片/视频: 自定义上传, 插入带 data-ref 持久引用的标签 (见文件头注释)
    uploadImage: {
      async customUpload(file: File) {
        const res = await fileApi.upload(file)
        const view = res.url
        const durable = res.ref || res.url
        editorRef.value?.dangerouslyInsertHtml(
          `<img src="${view}" data-ref="${durable}" alt="${escapeAttr(file.name)}">`,
        )
      },
    },
    uploadVideo: {
      async customUpload(file: File) {
        const res = await fileApi.upload(file)
        const view = res.url
        const durable = res.ref || res.url
        editorRef.value?.dangerouslyInsertHtml(
          `<video src="${view}" data-ref="${durable}" controls style="max-width:100%"></video>`,
        )
      },
    },
  },
}

// escapeAttr 文件名进 alt 属性前的最小转义
function escapeAttr(s: string) {
  return s.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;')
}

function handleCreated(editor: any) {
  editorRef.value = editor
  loading.value = false
}
</script>

<template>
  <div class="rich-editor">
    <component
      :is="ToolbarComp"
      v-if="ToolbarComp"
      class="re-toolbar"
      :editor="editorRef"
      :default-config="toolbarConfig"
      mode="default"
    />
    <component
      :is="EditorComp"
      v-if="EditorComp"
      v-model="html"
      class="re-editor"
      :style="{ height: `${height}px`, overflowY: 'hidden' }"
      :default-config="editorConfig"
      mode="default"
      @on-created="handleCreated"
    />
    <div v-if="loading" class="re-loading" :style="{ height: `${height}px` }">
      <el-skeleton :rows="4" animated />
    </div>
  </div>
</template>

<style scoped>
/* 暗黑模式适配: wangeditor 默认亮色, 用 Element Plus 变量覆盖 (见 AGENTS.md 颜色纪律) */
.rich-editor {
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-base);
}

.rich-editor :deep(.w-e-toolbar) {
  background-color: var(--el-fill-color-light);
  border-bottom: 1px solid var(--el-border-color-lighter) !important;
}

.rich-editor :deep(.w-e-bar-item button) {
  color: var(--el-text-color-primary);
}

.rich-editor :deep(.w-e-text-container) {
  background-color: var(--el-bg-color);
}

.rich-editor :deep(.w-e-text-placeholder) {
  color: var(--el-text-color-placeholder);
}

.re-loading {
  padding: 16px;
}
</style>
