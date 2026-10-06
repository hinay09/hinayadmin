<script setup lang="ts">
/**
 * 顶部消息通知铃铛组件。
 * - 定时轮询未读总数 + 分类统计
 * - 下拉气泡: 系统通知 / 私信通知 两个 Tab, 各展示最近 5 条未读
 * - 点击单条 -> 自动已读 + 跳转到对应列表
 * - 点击「全部已读」 / 「查看全部」
 */
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage, ElNotification } from 'element-plus'
import { Bell } from '@element-plus/icons-vue'
import { useMessageApi, type MessageItem, type MessageUnreadCount } from '~/composables/useApi'
import { useUserStore } from '~/stores/user'

const api = useMessageApi()
const router = useRouter()
const userStore = useUserStore()

const POLL_INTERVAL_ACTIVE = 15_000 // SSE 不可用时的兜底轮询周期
const POLL_INTERVAL_SSE = 120_000   // SSE 在线时的保底对齐周期(未读数完全由事件驱动)

const unread = ref<MessageUnreadCount>({ total: 0, system: 0, private: 0 })
const loading = ref(false)
const activeTab = ref<'system' | 'private'>('system')
const systemList = ref<MessageItem[]>([])
const privateList = ref<MessageItem[]>([])
let timer: ReturnType<typeof setInterval> | null = null
let ssePollFast = true // SSE 未连上时用短周期轮询兜底

/* ---- SSE 实时事件流 (fetch 流式读取, 支持 Authorization 头) ----
 * 服务端事件: hello(连接即下发当前未读数) / message(新消息, 弹提示并刷新未读)。
 * 断线自动退避重连 (5s, 10s, 30s 封顶); 重连期间由兜底轮询保证最终一致。 */
let sseAbort: AbortController | null = null
let sseReconnectTimer: ReturnType<typeof setTimeout> | null = null
let sseBackoff = 5_000

function setPollInterval() {
  if (timer) clearInterval(timer)
  timer = setInterval(fetchUnreadCount, ssePollFast ? POLL_INTERVAL_ACTIVE : POLL_INTERVAL_SSE)
}

function scheduleSseReconnect() {
  if (sseReconnectTimer) clearTimeout(sseReconnectTimer)
  sseReconnectTimer = setTimeout(connectSse, sseBackoff)
  sseBackoff = Math.min(sseBackoff * 2, 30_000)
}

async function connectSse() {
  if (!userStore.token || !import.meta.client) return
  const config = useRuntimeConfig()
  sseAbort = new AbortController()
  try {
    const res = await fetch(`${config.public.apiBase}/message/events`, {
      headers: { Authorization: `Bearer ${userStore.token}` },
      signal: sseAbort.signal,
    })
    // 429=连接数达上限(服务端每用户/全局配额), 401=会话失效:
    // 不再重连, 交由轮询兜底 / 路由守卫处理登出。
    if (res.status === 429 || res.status === 401) {
      if (!ssePollFast) {
        ssePollFast = true
        setPollInterval()
      }
      return
    }
    if (!res.ok || !res.body) throw new Error(`sse ${res.status}`)

    sseBackoff = 5_000
    if (ssePollFast) {
      // SSE 通道建立: 未读数改由事件驱动, 轮询降为长周期保底
      ssePollFast = false
      setPollInterval()
    }

    const reader = res.body.getReader()
    const decoder = new TextDecoder()
    let buf = ''
    for (;;) {
      const { done, value } = await reader.read()
      if (done) break
      buf += decoder.decode(value, { stream: true })
      // SSE 帧以空行分隔
      let idx
      while ((idx = buf.indexOf('\n\n')) >= 0) {
        const frame = buf.slice(0, idx)
        buf = buf.slice(idx + 2)
        handleSseFrame(frame)
      }
    }
    // 流被服务端正常收束(单连接 1 小时上限到期主动收流): 立即重连
    throw new Error('sse closed by server')
  }
  catch {
    // 连接断开/失败/服务端收流: 切回短周期轮询并退避重连
    if (!ssePollFast) {
      ssePollFast = true
      setPollInterval()
    }
    scheduleSseReconnect()
  }
}

function handleSseFrame(frame: string) {
  let event = 'message'
  let data = ''
  for (const line of frame.split('\n')) {
    if (line.startsWith('event:')) event = line.slice(6).trim()
    else if (line.startsWith('data:')) data += line.slice(5).trim()
  }
  if (!data) return // 心跳注释帧等
  if (event === 'hello') {
    try {
      const cnt = JSON.parse(data)
      unread.value = {
        total: Number(cnt?.total || 0),
        system: Number(cnt?.system || 0),
        private: Number(cnt?.private || 0),
      }
    }
    catch {}
    return
  }
  if (event === 'message') {
    try {
      const ev = JSON.parse(data)
      ElNotification({
        title: '新消息',
        message: ev?.title || '您有一条新的消息',
        type: ev?.level === 3 ? 'error' : ev?.level === 2 ? 'warning' : 'info',
        duration: 5000,
      })
    }
    catch {}
    fetchUnreadCount()
  }
}

const badge = computed(() => (unread.value.total > 99 ? '99+' : unread.value.total || ''))

async function fetchUnreadCount() {
  try {
    const res: any = await api.unreadCount()
    unread.value = {
      total: Number(res?.total || 0),
      system: Number(res?.system || 0),
      private: Number(res?.private || 0),
    }
  }
  catch {}
}

async function fetchPreviewList(type: 1 | 2) {
  loading.value = true
  try {
    const res: any = await api.inbox({ type, isRead: 0, page: 1, pageSize: 5 })
    const arr: MessageItem[] = res?.list || []
    if (type === 1) systemList.value = arr
    else privateList.value = arr
  }
  finally {
    loading.value = false
  }
}

async function handleShow() {
  // 首次打开同时拉两类，后续切 Tab 不重复拉（在 markAllRead/openItem 后会刷新）
  await Promise.all([fetchPreviewList(1), fetchPreviewList(2)])
  await fetchUnreadCount()
}

async function switchTab(tab: 'system' | 'private') {
  activeTab.value = tab
  // 按需拉：如果该 Tab 还没加载过，补一次
  if (tab === 'system' && systemList.value.length === 0 && unread.value.system > 0) {
    await fetchPreviewList(1)
  }
  if (tab === 'private' && privateList.value.length === 0 && unread.value.private > 0) {
    await fetchPreviewList(2)
  }
}

async function openItem(item: MessageItem) {
  try {
    await api.inboxRead(item.id)
  }
  catch {}
  // 携带 id 跳转到列表页，列表页会监听 query 自动弹出详情抽屉
  router.push(msgRoute(item.type, item.id))
  // 刷新未读数与预览
  fetchUnreadCount()
  fetchPreviewList(item.type === 1 ? 1 : 2)
}

// 消息落地页: 私信 → 私信通知页; 系统通知 → 管理员进发布管理页,
// 普通用户进收件视图 /message/private?type=1 (全走已授权的 inbox 接口,
// 避免误入管理端发布台触发 403)。
function msgRoute(type: 1 | 2, id?: number) {
  const q: Record<string, string> = {}
  if (id) q.id = String(id)
  let path = '/message/private'
  if (type === 1) {
    if (userStore.isAdmin) path = '/message/system'
    else q.type = '1'
  }
  return { path, query: q }
}

async function markAllRead() {
  const res: any = await api.markReadAll()
  ElMessage.success(`已标记 ${res?.affected || 0} 条为已读`)
  await Promise.all([fetchUnreadCount(), fetchPreviewList(1), fetchPreviewList(2)])
}

function viewAll() {
  router.push(msgRoute(activeTab.value === 'system' ? 1 : 2))
}

function formatTime(t: string) {
  if (!t) return ''
  const d = new Date(t.replace(' ', 'T'))
  if (isNaN(d.getTime())) return t
  const now = Date.now()
  const diff = (now - d.getTime()) / 1000
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)} 小时前`
  if (diff < 86400 * 7) return `${Math.floor(diff / 86400)} 天前`
  return t.slice(0, 16)
}

const levelTag = (l: number) => (l === 3 ? 'danger' : l === 2 ? 'warning' : 'info')
const levelText = (l: number) => (l === 3 ? '紧急' : l === 2 ? '重要' : '普通')

onMounted(() => {
  fetchUnreadCount()
  setPollInterval()
  connectSse()
})
onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
  sseAbort?.abort()
  if (sseReconnectTimer) clearTimeout(sseReconnectTimer)
})
</script>

<template>
  <el-popover
    placement="bottom-end"
    :width="380"
    trigger="click"
    :show-arrow="true"
    @show="handleShow"
  >
    <template #reference>
      <div class="bell-wrapper" title="消息通知">
        <el-badge :value="badge" :hidden="!unread.total" class="bell-badge">
          <el-icon class="header-action"><Bell /></el-icon>
        </el-badge>
      </div>
    </template>

    <div class="msg-pop">
      <div class="msg-tabs">
        <div
          class="msg-tab"
          :class="{ active: activeTab === 'system' }"
          @click="switchTab('system')"
        >
          系统通知
          <el-badge v-if="unread.system" :value="unread.system" :max="99" class="tab-badge" />
        </div>
        <div
          class="msg-tab"
          :class="{ active: activeTab === 'private' }"
          @click="switchTab('private')"
        >
          私信
          <el-badge v-if="unread.private" :value="unread.private" :max="99" class="tab-badge" />
        </div>
      </div>

      <div v-loading="loading" class="msg-body">
        <template v-if="activeTab === 'system'">
          <div v-if="!systemList.length" class="empty">暂无未读系统通知</div>
          <div
            v-for="m in systemList"
            :key="m.id"
            class="msg-item"
            @click="openItem(m)"
          >
            <span class="unread-dot" />
            <div class="msg-item-main">
              <div class="msg-item-head">
                <el-tag type="primary" size="small" effect="plain">系统</el-tag>
                <el-tag :type="levelTag(m.level)" size="small">{{ levelText(m.level) }}</el-tag>
                <span class="msg-title">{{ m.title }}</span>
              </div>
              <div class="msg-meta">
                <span>{{ m.senderName || '系统' }}</span>
                <span>{{ formatTime(m.createdAt) }}</span>
              </div>
            </div>
          </div>
        </template>
        <template v-else>
          <div v-if="!privateList.length" class="empty">暂无未读私信</div>
          <div
            v-for="m in privateList"
            :key="m.id"
            class="msg-item"
            @click="openItem(m)"
          >
            <span class="unread-dot" />
            <div class="msg-item-main">
              <div class="msg-item-head">
                <el-tag type="success" size="small" effect="plain">私信</el-tag>
                <el-tag :type="levelTag(m.level)" size="small">{{ levelText(m.level) }}</el-tag>
                <span class="msg-title">{{ m.title }}</span>
              </div>
              <div class="msg-meta">
                <span>来自 {{ m.senderName }}</span>
                <span>{{ formatTime(m.createdAt) }}</span>
              </div>
            </div>
          </div>
        </template>
      </div>

      <div class="msg-footer">
        <el-button link type="primary" size="small" @click="markAllRead">全部已读</el-button>
        <el-button link type="primary" size="small" @click="viewAll">查看全部</el-button>
      </div>
    </div>
  </el-popover>
</template>

<style scoped>
.bell-badge :deep(.el-badge__content) {
  top: 4px;
  right: 8px;
}
.bell-wrapper {
  display: inline-flex;
  align-items: center;
  cursor: pointer;
}
.header-action {
  font-size: 18px;
  width: 36px;
  height: 36px;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: #606266;
  transition: background-color .2s, color .2s;
}
.header-action:hover {
  background-color: #f0f2f5;
  color: #409eff;
}
.msg-pop {
  margin: -12px;
}
.msg-tabs {
  display: flex;
  border-bottom: 1px solid #ebeef5;
}
.msg-tab {
  flex: 1;
  text-align: center;
  padding: 10px 0;
  cursor: pointer;
  font-size: 13px;
  color: #606266;
  position: relative;
  transition: color .2s;
}
.msg-tab.active {
  color: #409eff;
  font-weight: 600;
  border-bottom: 2px solid #409eff;
}
.tab-badge {
  margin-left: 4px;
}
.tab-badge :deep(.el-badge__content) {
  position: relative;
  top: 0;
  right: 0;
  transform: none;
}
.msg-body {
  max-height: 360px;
  overflow-y: auto;
  padding: 4px 0;
}
.empty {
  text-align: center;
  color: #909399;
  font-size: 13px;
  padding: 40px 0;
}
.msg-item {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 10px 12px 10px 22px;
  cursor: pointer;
  border-bottom: 1px solid #f5f7fa;
  transition: background-color .2s;
}
.msg-item:hover {
  background-color: #f0f7ff;
}
.msg-item:last-child {
  border-bottom: none;
}
.unread-dot {
  position: absolute;
  left: 10px;
  top: 18px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #f56c6c;
  box-shadow: 0 0 0 2px rgba(245, 108, 108, .15);
}
.msg-item-main {
  flex: 1;
  min-width: 0;
}
.msg-item-head {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}
.msg-title {
  font-size: 13px;
  color: #303133;
  font-weight: 500;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.msg-meta {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #909399;
}
.msg-footer {
  display: flex;
  justify-content: space-between;
  padding: 8px 12px;
  border-top: 1px solid #ebeef5;
}
</style>
