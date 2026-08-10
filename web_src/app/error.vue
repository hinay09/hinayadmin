<script setup lang="ts">
/**
 * Nuxt 全局错误页 (error.vue)
 * - 接收 Nuxt 注入的 error prop: { statusCode, message, ... }
 * - 403 / 404 独立设计, 其他错误显示通用提示
 */
import { computed } from 'vue'
import { WarningFilled, CircleCloseFilled, HomeFilled } from '@element-plus/icons-vue'

const props = defineProps<{
  error: {
    statusCode: number
    message: string
    url?: string
  }
}>()

const is404 = computed(() => props.error.statusCode === 404)
const is403 = computed(() => props.error.statusCode === 403)

const statusText = computed(() => {
  if (is404.value) return '页面不存在'
  if (is403.value) return '无访问权限'
  return '服务异常'
})

const description = computed(() => {
  if (is404.value) return '抱歉, 您访问的页面不存在或已被移除。'
  if (is403.value) return '抱歉, 您没有权限访问此页面, 请联系管理员。'
  return props.error.message || '抱歉, 服务器发生了未知错误, 请稍后重试。'
})

function goHome() {
  clearError({ redirect: '/' })
}

function goBack() {
  if (import.meta.client && window.history.length > 1) {
    window.history.back()
  }
  else {
    clearError({ redirect: '/' })
  }
}
</script>

<template>
  <div class="error-page">
    <div class="error-card">
      <div class="error-icon-wrap">
        <el-icon v-if="is404" :size="72" color="#E6A23C"><WarningFilled /></el-icon>
        <el-icon v-else-if="is403" :size="72" color="#F56C6C"><CircleCloseFilled /></el-icon>
        <el-icon v-else :size="72" color="#909399"><WarningFilled /></el-icon>
      </div>

      <h1 class="error-code">{{ error.statusCode }}</h1>
      <p class="error-title">{{ statusText }}</p>
      <p class="error-desc">{{ description }}</p>

      <div class="error-actions">
        <el-button size="large" @click="goBack">返回上页</el-button>
        <el-button type="primary" size="large" :icon="HomeFilled" @click="goHome">回到首页</el-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.error-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f7fa;
}

.error-card {
  text-align: center;
  padding: 48px 40px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
  max-width: 480px;
}

.error-icon-wrap {
  margin-bottom: 8px;
}

.error-code {
  font-size: 72px;
  font-weight: 700;
  color: #303133;
  margin: 0;
  line-height: 1;
  letter-spacing: 2px;
}

.error-title {
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  margin: 16px 0 8px;
}

.error-desc {
  font-size: 14px;
  color: #909399;
  margin: 0 0 32px;
  line-height: 1.6;
}

.error-actions {
  display: flex;
  justify-content: center;
  gap: 12px;
}
</style>
