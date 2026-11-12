<script setup lang="ts">
/**
 * 递归菜单节点: 叶子渲染 el-menu-item, 有子级渲染 el-sub-menu 并递归下钻,
 * 支持任意层级 (经典侧边栏与双层菜单的面板/附加框共用同一渲染结构)。
 */
import type { MenuNode } from '~/stores/user'

defineProps<{ item: MenuNode }>()
</script>

<template>
  <el-sub-menu v-if="item.children && item.children.length" :index="String(item.id)">
    <template #title>
      <el-icon><component :is="item.icon || 'Folder'" /></el-icon>
      <span>{{ item.name }}</span>
    </template>
    <SidebarItem v-for="c in item.children" :key="c.id" :item="c" />
  </el-sub-menu>
  <el-menu-item v-else :index="item.path">
    <el-icon><component :is="item.icon || 'Menu'" /></el-icon>
    <template #title>{{ item.name }}</template>
  </el-menu-item>
</template>
