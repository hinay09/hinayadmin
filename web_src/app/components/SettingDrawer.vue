<script setup lang="ts">
/**
 * 系统布局设置抽屉 (仿 RuoYi): 顶栏齿轮触发, el-drawer 承载全部个性化配置。
 * - 主题颜色: 预设色板 + 取色器, 运行时重写 Element Plus 主色 CSS 变量
 * - 布局开关: 暗黑模式 / 固定头部 / 侧边栏 Logo / 多标签页
 * - 页面水印: 开关 + 自定义文案 (留空默认取当前用户 "昵称(用户名)")
 * 所有设置即时生效并持久化 localStorage, 支持一键恢复默认。
 */
import { computed, ref } from 'vue'
import { Check, RefreshLeft, Setting } from '@element-plus/icons-vue'
import { useSettingsStore } from '~/stores/settings'

const settings = useSettingsStore()
const visible = ref(false)

/** 预设主题色 (RuoYi/plus-ui 常用配色) */
const presetThemes = [
  '#409eff', '#13c2c2', '#36ad6a', '#f5222d',
  '#fa8c16', '#722ed1', '#eb2f96', '#1f2d3d',
]

/* 开关均经 store action 走 "应用 + 持久化" 链路, 直接改 state 不会触发副作用 */
const darkProxy = computed({
  get: () => settings.isDark,
  set: v => settings.toggleDark(v),
})
const fixedHeaderProxy = computed({
  get: () => settings.fixedHeader,
  set: v => settings.update({ fixedHeader: v }),
})
const showLogoProxy = computed({
  get: () => settings.showLogo,
  set: v => settings.update({ showLogo: v }),
})
const showTagsProxy = computed({
  get: () => settings.showTags,
  set: v => settings.update({ showTags: v }),
})
const watermarkProxy = computed({
  get: () => settings.watermark,
  set: v => settings.update({ watermark: v }),
})
const watermarkTextProxy = computed({
  get: () => settings.watermarkText,
  set: v => settings.update({ watermarkText: v }),
})

const themeProxy = computed({
  get: () => settings.theme,
  set: v => {
    // 取色器清空/中间态收到 null 时忽略
    if (v) settings.setTheme(v)
  },
})

function handleReset() {
  settings.reset()
}
</script>

<template>
  <el-tooltip content="布局设置" placement="bottom">
    <el-icon class="setting-trigger" @click="visible = true">
      <Setting />
    </el-icon>
  </el-tooltip>

  <el-drawer v-model="visible" title="系统布局配置" size="320px" append-to-body>
    <div class="setting-panel">
      <div class="setting-block">
        <div class="setting-block__title">主题颜色</div>
        <div class="theme-swatches">
          <span
            v-for="c in presetThemes"
            :key="c"
            class="theme-swatch"
            :class="{ active: settings.theme === c }"
            :style="{ background: c }"
            @click="settings.setTheme(c)"
          >
            <el-icon v-if="settings.theme === c" color="#fff" :size="12"><Check /></el-icon>
          </span>
          <el-color-picker v-model="themeProxy" :predefine="presetThemes" />
        </div>
      </div>

      <div class="setting-block">
        <div class="setting-block__title">布局设置</div>
        <div class="setting-row">
          <span>暗黑模式</span>
          <el-switch v-model="darkProxy" />
        </div>
        <div class="setting-row">
          <span>固定头部</span>
          <el-switch v-model="fixedHeaderProxy" />
        </div>
        <div class="setting-row">
          <span>侧边栏 Logo</span>
          <el-switch v-model="showLogoProxy" />
        </div>
        <div class="setting-row">
          <span>多标签页</span>
          <el-switch v-model="showTagsProxy" />
        </div>
      </div>

      <div class="setting-block">
        <div class="setting-block__title">页面水印</div>
        <div class="setting-row">
          <span>开启全屏水印</span>
          <el-switch v-model="watermarkProxy" />
        </div>
        <el-input
          v-if="settings.watermark"
          v-model="watermarkTextProxy"
          placeholder="默认为当前用户 昵称(用户名)"
          clearable
          maxlength="30"
        />
      </div>

      <el-divider />

      <el-button plain style="width: 100%" @click="handleReset">
        <el-icon style="margin-right: 6px"><RefreshLeft /></el-icon>
        恢复默认
      </el-button>
    </div>
  </el-drawer>
</template>

<style scoped>
/* 触发图标与布局顶栏的 header-action 同款外观 (父组件 scoped 样式无法作用到子组件) */
.setting-trigger {
  font-size: 18px;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--el-text-color-regular);
  transition: background-color .2s, color .2s;
}
.setting-trigger:hover {
  background-color: var(--el-fill-color);
  color: var(--el-color-primary);
}

.setting-panel {
  padding: 0 4px;
}
.setting-block__title {
  font-weight: 600;
  font-size: 14px;
  color: var(--el-text-color-primary);
  margin-bottom: 14px;
}
.setting-block {
  margin-bottom: 24px;
}
.theme-swatches {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}
.theme-swatch {
  width: 22px;
  height: 22px;
  border-radius: 6px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--el-border-color-lighter);
  transition: transform .15s;
}
.theme-swatch:hover {
  transform: scale(1.12);
}
.theme-swatch.active {
  outline: 2px solid var(--el-color-primary);
  outline-offset: 2px;
}
.setting-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 14px;
  color: var(--el-text-color-regular);
  padding: 9px 0;
}
</style>
