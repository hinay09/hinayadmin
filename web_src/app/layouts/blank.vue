<script setup lang="ts">
/**
 * 空白布局: 用于登录等无需骨架的页面
 */
</script>

<template>
  <div class="blank-layout">
    <slot />
  </div>
</template>

<style scoped>
.blank-layout {
  width: 100%;
  min-height: 100vh;
}
</style>
