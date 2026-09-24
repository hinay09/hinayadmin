<script setup lang="ts">
// 根组件: 由 NuxtLayout 选择布局, NuxtPage 渲染当前页面
import { computed } from 'vue'
import { useConfigStore } from '~/stores/config'

const route = useRoute()
const configStore = useConfigStore()

// 浏览器标题: 页面 title (definePageMeta) + 系统名称(sys.name), 无页面标题时仅显示系统名称
const headTitle = computed(() => {
  const t = route.meta.title as string | undefined
  return t ? `${t} - ${configStore.siteName}` : configStore.siteName
})
useHead({ title: headTitle })
</script>

<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</template>
