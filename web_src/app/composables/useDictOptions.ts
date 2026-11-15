/**
 * 字典选项 hook: 一次拉取全量字典 (按类型编码分组), 模块级缓存共享。
 *
 * 用法:
 *   const { options, label } = useDictOptions()   // 需在 setup 内调用 (ProTable 已内置)
 *   options('order_type')  // Ref<{ dictLabel, dictValue }[]>
 *   label('order_type', 1) // 值 -> 文案 (表格列展示)
 *
 * SSR 阶段不发起请求 (服务端无会话上下文), 客户端挂载后补拉, computed 自动更新。
 * 失败时静默为空选项 (不阻塞页面)。
 */
import { computed, ref } from 'vue'
import { useDictApi } from '~/composables/useApi/dict'

interface DictItem {
  dictLabel: string
  dictValue: string
}

// 模块级缓存: 同一会话多页面共享一次请求
const cached = ref<Record<string, DictItem[]> | null>(null)
let pending: Promise<void> | null = null
// API 实例须在 setup 上下文内创建 (useRequest 依赖 Nuxt 上下文), 首次调用时捕获复用
let dictApi: ReturnType<typeof useDictApi> | null = null

async function load() {
  if (!import.meta.client || cached.value)
    return
  if (!pending) {
    if (!dictApi)
      return // 未经过 setup 调用 (无上下文), 等待下一次
    pending = dictApi.all()
      .then((res: any) => {
        cached.value = res.list || {}
      })
      .catch(() => {
        cached.value = {}
      })
      .finally(() => {
        pending = null
      })
  }
  await pending
}

// 按编码缓存 computed: 模板里高频调用 options(code) 不会每次渲染新建
const optionsCache = new Map<string, ReturnType<typeof computed<DictItem[]>>>()

export function useDictOptions() {
	// setup 上下文内捕获 API 实例, 供后续渲染期/事件回调复用
	if (import.meta.client && !dictApi)
		dictApi = useDictApi()
	load() // 异步预热, 不阻塞
	return {
		/** 指定类型编码的选项列表 (响应式, 加载完成后自动更新; 同编码共享同一 computed) */
		options(code: string) {
			load()
			let c = optionsCache.get(code)
			if (!c) {
				c = computed(() => cached.value?.[code] || [])
				optionsCache.set(code, c)
			}
			return c
		},
    /** 值 -> 显示文案 (表格列/详情展示, 未知值原样返回) */
    label(code: string, value: any): string {
      const items = cached.value?.[code]
      if (!items)
        return String(value ?? '')
      const hit = items.find(i => i.dictValue === String(value))
      return hit ? hit.dictLabel : String(value ?? '')
    },
  }
}
