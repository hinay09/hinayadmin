<script setup lang="ts">
/**
 * 根路径: 重定向到 /dashboard。
 * 用页面中间件而非 onMounted: 中间件在导航确认前执行 (SSR/客户端均生效),
 * "/" 不会成为落定路由, 多标签页不会出现"首页"标签, 也消除挂载后的闪跳。
 */
definePageMeta({
  middleware: [function () {
    return navigateTo('/dashboard', { replace: true })
  }],
})
</script>

<template>
  <div />
</template>
