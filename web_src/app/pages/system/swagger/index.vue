<script setup lang="ts">
/**
 * 接口文档 (仅超管可见, 菜单挂在系统工具下): 内嵌后端 GoFrame 内置的 Redoc 页面。
 * - src 用相对路径 /swagger: 开发环境由 nitro.devProxy 转发到 :8000,
 *   生产部署需在 nginx 将 /swagger 与 /api.json 反代到后端 (与 /api 同理)
 * - 后端 /swagger 本身无鉴权 (GoFrame 内置静态路由, 不走 Auth 中间件),
 *   菜单控制的是入口便捷性; 生产环境如不希望公开可在 config.yaml 置空 swaggerPath
 */
import { ref, onMounted } from 'vue'
import { Link } from '@element-plus/icons-vue'

definePageMeta({ title: '接口文档' })
defineOptions({ name: 'system-swagger' })

const loading = ref(true)
const failed = ref(false)
const src = '/swagger'

function onLoaded() {
  // 同源(经代理)时 load 事件可靠; Redoc 渲染在 load 之后, 稍候再撤 loading
  loading.value = false
}

/** 新窗口打开原始文档页 (模板作用域拿不到 window, 在此转发) */
function openInNewTab() {
  if (import.meta.client) window.open(src, '_blank')
}

onMounted(() => {
  // iframe 静默失败(代理未配置/后端未启动)不触发 onerror, 用超时兜底提示
  setTimeout(() => {
    if (loading.value) failed.value = true
  }, 8000)
})
</script>

<template>
  <div class="page swagger-page">
    <div class="toolbar">
      <span class="toolbar-title">接口文档</span>
      <span class="toolbar-sub">OpenAPI 3 · 由后端自动生成, 随代码实时更新</span>
      <el-button class="toolbar-open" size="small" text type="primary" :icon="Link" @click="openInNewTab">
        新窗口打开
      </el-button>
    </div>
    <div class="swagger-frame-wrap">
      <iframe
        v-loading="loading"
        :src="src"
        class="swagger-frame"
        title="接口文档"
        @load="onLoaded"
      />
      <div v-if="failed" class="swagger-fallback">
        <p>文档加载不出来? 常见原因:</p>
        <p>1. 后端未启动, 或开发代理未生效 (需在 nuxt.config devProxy 配置 /swagger 与 /api.json)</p>
        <p>2. 生产部署: nginx 需将 /swagger 与 /api.json 反代到后端</p>
        <el-button type="primary" size="small" @click="openInNewTab">直接打开 /swagger</el-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.swagger-page {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 122px); /* 顶栏 + 标签栏 + 主区内边距 */
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.toolbar-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.toolbar-sub {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.toolbar-open {
  margin-left: auto;
}
.swagger-frame-wrap {
  position: relative;
  flex: 1;
  min-height: 0;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  overflow: hidden;
}
.swagger-frame {
  width: 100%;
  height: 100%;
  border: 0;
  display: block;
}
.swagger-fallback {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font-size: 13px;
  text-align: center;
  padding: 24px;
}
</style>
