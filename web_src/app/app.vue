<script setup lang="ts">
// 根组件: 由 NuxtLayout 选择布局, NuxtPage 渲染当前页面
import { computed, watch } from 'vue'
import { useConfigStore } from '~/stores/config'
import { useTagsStore } from '~/stores/tags'

const route = useRoute()
const configStore = useConfigStore()
const tagsStore = useTagsStore()

// 多标签页: 路由每次变化登记标签 (KeepAlive 缓存名单由此驱动);
// immediate 保证刷新/首屏时当前页也入列
watch(() => route.path, () => tagsStore.addTag(route), { immediate: true })

// 浏览器标题: 页面 title (definePageMeta) + 系统名称(sys.name), 无页面标题时仅显示系统名称
const headTitle = computed(() => {
  const t = route.meta.title as string | undefined
  return t ? `${t} - ${configStore.siteName}` : configStore.siteName
})
useHead({ title: headTitle })
</script>

<template>
  <NuxtLayout>
    <!-- KeepAlive 按标签缓存页面 (include 匹配 defineOptions name);
         key 拼接 refreshSeed, 标签页"刷新"时强制销毁重建;
         Transition 提供页面切换淡入淡出 (样式见 global.css 的 .page-*) -->
    <NuxtPage v-slot="{ Component, route: pageRoute }">
      <Transition name="page" mode="out-in">
        <KeepAlive :include="tagsStore.cachedNames">
          <component
            :is="Component"
            :key="pageRoute.path + '-' + (tagsStore.refreshSeed[pageRoute.path] || 0)"
          />
        </KeepAlive>
      </Transition>
    </NuxtPage>
  </NuxtLayout>
</template>
