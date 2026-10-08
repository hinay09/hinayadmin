<script setup lang="ts">
/**
 * 全屏防泄密水印 (仿 RuoYi-Vue-Plus / vben): canvas 生成平铺文字水印, fixed 覆盖整个视口。
 * - 首行文案: 设置面板自定义, 留空默认取当前用户 "昵称(用户名)"; 第二行自动带当前日期
 * - 暗黑模式自动切换水印颜色 (浅字/深字)
 * - MutationObserver 防篡改: 节点被删除则原位插回, style/class 被改则立即还原
 * 悬浮于内容之上 (z-index 9999, 高于弹窗/消息提示) 但 pointer-events:none, 不影响交互。
 * ClientOnly 包裹: 设置来自 localStorage, 服务端渲染时未知, 避免水合不一致。
 */
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useSettingsStore } from '~/stores/settings'
import { useUserStore } from '~/stores/user'

const settings = useSettingsStore()
const userStore = useUserStore()
const { watermark, watermarkText, isDark } = storeToRefs(settings)

const wmRef = ref<HTMLElement | null>(null)
const dataUrl = ref('')
/** 防篡改: 记录原始插入位置, 被删除后原位插回 */
let anchor: { parent: Node | null; next: Node | null } | null = null
let observer: MutationObserver | null = null

/** 首行文案: 自定义优先, 否则取当前用户 "昵称(用户名)" */
function firstLine(): string {
  const t = settings.watermarkText.trim()
  if (t) return t
  const u = userStore.userInfo
  if (!u) return ''
  if (u.nickname && u.username && u.nickname !== u.username) return `${u.nickname}(${u.username})`
  return u.nickname || u.username || ''
}

function pad(n: number) {
  return String(n).padStart(2, '0')
}

/** canvas 生成平铺单元: 两行文字旋转 -24° */
function createWatermark(line1: string, line2: string, dark: boolean): string {
  const font1 = '600 15px "PingFang SC", "Microsoft YaHei", Arial, sans-serif'
  const font2 = '13px "PingFang SC", "Microsoft YaHei", Arial, sans-serif'
  const canvas = document.createElement('canvas')
  const ctx = canvas.getContext('2d')
  if (!ctx) return ''
  ctx.font = font1
  const w1 = ctx.measureText(line1).width
  ctx.font = font2
  const w2 = ctx.measureText(line2).width
  const tileW = Math.ceil(Math.max(w1, w2) * 1.6) + 60
  const tileH = 110
  canvas.width = tileW
  canvas.height = tileH
  // 设置 canvas 尺寸会重置绘图状态, 字体须在此之后重新声明
  ctx.font = font1
  ctx.fillStyle = dark ? 'rgba(255, 255, 255, 0.13)' : 'rgba(0, 0, 0, 0.09)'
  ctx.textAlign = 'center'
  ctx.translate(tileW / 2, tileH / 2)
  ctx.rotate(-24 * Math.PI / 180)
  ctx.fillText(line1, 0, -8)
  ctx.font = font2
  ctx.fillText(line2, 0, 16)
  return canvas.toDataURL('image/png')
}

function stopDefend() {
  observer?.disconnect()
  observer = null
  anchor = null
}

/** 挂防篡改观察器: 元素属性被改 -> 还原; 元素被删 -> 原位插回 */
function startDefend() {
  stopDefend()
  const el = wmRef.value
  if (!el || !el.parentElement) return
  anchor = { parent: el.parentElement, next: el.nextSibling }
  observer = new MutationObserver((records) => {
    const el = wmRef.value
    if (!el || !settings.watermark) return
    for (const r of records) {
      // 改样式/类名: 恢复强制样式 (水印层只依赖这几个声明)
      if (r.type === 'attributes') {
        el.style.setProperty('position', 'fixed')
        el.style.setProperty('pointer-events', 'none')
        el.style.setProperty('z-index', '9999')
        el.style.removeProperty('display')
      }
      // 被删除: 插回原位 (仅在水印仍开启时; 关闭开关的正常卸载不拦截)
      else if (r.type === 'childList' && !el.isConnected && anchor?.parent) {
        anchor.parent.insertBefore(el, anchor.next)
        nextTick(startDefend)
        return
      }
    }
  })
  observer.observe(el, { attributes: true, attributeFilter: ['style', 'class'] })
  observer.observe(el.parentElement, { childList: true })
}

async function render() {
  // 先停观察器: 水印关闭时 v-if 卸载元素, 不能与防篡改插回逻辑打架
  stopDefend()
  const line = firstLine()
  if (!settings.watermark || !line) {
    dataUrl.value = ''
    return
  }
  const d = new Date()
  const line2 = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
  dataUrl.value = createWatermark(line, line2, settings.isDark)
  await nextTick()
  startDefend()
}

watch([watermark, watermarkText, isDark], render)
watch(() => userStore.userInfo?.nickname, render)
onMounted(render)
onBeforeUnmount(stopDefend)
</script>

<template>
  <ClientOnly>
    <div
      v-if="watermark && dataUrl"
      ref="wmRef"
      class="app-watermark"
      :style="{ backgroundImage: `url(${dataUrl})` }"
    />
  </ClientOnly>
</template>

<style scoped>
.app-watermark {
  position: fixed;
  inset: 0;
  z-index: 9999;
  pointer-events: none;
  background-repeat: repeat;
  background-position: center;
}
</style>
