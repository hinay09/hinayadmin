<script setup lang="ts">
/**
 * 根路径: 重定向到 /dashboard
 */
definePageMeta({ title: '首页' })
onMounted(() => {
  navigateTo('/dashboard', { replace: true })
})
</script>

<template>
  <div />
</template>
