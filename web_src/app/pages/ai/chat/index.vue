<script setup lang="ts">
/**
 * AI 智能对话 (版式参考 snowy-admin-web / 主流 AI 对话产品):
 * - 左侧会话栏: 新对话按钮 + 分割线 + 历史会话列表 (选择切换 / 悬浮删除);
 * - 右侧对话区: 头像式消息布局 / 等待三点动画 / 输入框;
 * - 仅上送本轮 message + sessionId, 上下文由服务端按 sessionId 记忆 (Redis, 7 天空闲过期);
 * - 回复 SSE 流式渲染 (markdown-it + highlight.js 暗色代码块), 支持停止生成;
 * - 推理模型思考过程可折叠展示 (正文开始自动折叠), 思考不入上下文;
 * - 工具调用步骤条: 执行中转圈 → 完成/失败, 点击展开参数与结果; 历史回放时折叠到对应 assistant 气泡;
 * - 每轮 token 用量展示 (输入/输出/合计, 气泡底部; 网关未回报时隐藏), 历史回放同样展示;
 * - 管理员未配置 ai.api_key 时给出引导提示。
 */
import MarkdownIt from 'markdown-it'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Promotion, VideoPause, Plus, User, Delete, ChatDotRound, InfoFilled, Loading, CircleCheckFilled, CircleCloseFilled, Lightning } from '@element-plus/icons-vue'
import { useAiApi, type AiConfigStatus, type AiHistoryMessage, type AiSessionItem, type AiToolStep, type AiTokenUsage } from '~/composables/useApi/ai'

definePageMeta({ title: '智能对话' })
defineOptions({ name: 'ai-chat' })

const api = useAiApi()

// Markdown 渲染 (html 关闭防注入; 代码块 highlight.js 高亮, 配暗色代码块样式)
const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
  highlight(code, lang) {
    if (lang && hljs.getLanguage(lang)) {
      return hljs.highlight(code, { language: lang, ignoreIllegals: true }).value
    }
    return hljs.highlightAuto(code).value
  },
})
function renderMd(text: string) {
  return md.render(text || '')
}

// token 数量千分位 (如 1,801)
function fmtNum(n: number) {
  return (n || 0).toLocaleString()
}

// ============================================================
// 会话: 前端生成并保管 sessionId, 服务端按其维系上下文记忆与用户会话索引
// ============================================================
const SESSION_KEY = 'ai_chat_session_id'
function newSessionId() {
  return `sess_${Date.now()}_${Math.random().toString(36).slice(2, 11)}`
}
function setActiveSession(id: string) {
  activeSessionId.value = id
  if (import.meta.client) localStorage.setItem(SESSION_KEY, id)
}

/** 工具步骤条目 (流式与历史回放共用 UI 状态) */
interface ToolStepUI extends AiToolStep {
  done: boolean // 执行中转圈 → 完成打勾/失败叉
  open?: boolean // 详情展开
}

interface Bubble {
  role: 'user' | 'assistant'
  content: string
  reasoning?: string
  showThink?: boolean
  error?: boolean
  streaming?: boolean
  steps?: ToolStepUI[] // 本轮工具调用步骤 (折叠在正文上方)
  usage?: AiTokenUsage // 本轮 token 用量 (total=0 或缺省时不展示)
}

const bubbles = ref<Bubble[]>([])
const input = ref('')
const streaming = ref(false)
const listRef = ref<HTMLElement>()
let abort: AbortController | null = null

// 左侧历史会话栏
const sessions = ref<AiSessionItem[]>([])
const sessionsLoading = ref(false)
const activeSessionId = ref('')
const activeTitle = computed(() => {
  if (!activeSessionId.value) return '新对话'
  const hit = sessions.value.find(s => s.sessionId === activeSessionId.value)
  return hit?.title || '新对话'
})

const cfg = ref<AiConfigStatus | null>(null)
const cfgLoaded = ref(false)
const notConfigured = computed(() => cfgLoaded.value && cfg.value && !cfg.value.configured)

async function loadCfg() {
  try {
    cfg.value = await api.config()
  } catch {} finally { cfgLoaded.value = true }
}

async function loadSessions() {
  sessionsLoading.value = true
  try {
    const res = await api.sessions()
    sessions.value = res.list || []
  } catch {} finally { sessionsLoading.value = false }
}

// 恢复某个会话的历史: 服务端按 sessionId 持久化了会话, 切换/刷新时回填界面
// (思考过程不入历史; 工具步骤行 role=tool 折叠到其后紧邻的 assistant 气泡)
async function loadHistory(sessionId: string) {
  if (!sessionId) return
  try {
    const res = await api.history(sessionId)
    const list: AiHistoryMessage[] = res.list || []
    const out: Bubble[] = []
    let pending: ToolStepUI[] = [] // 暂存待挂靠的工具步骤
    for (const m of list) {
      if (m.role === 'tool') {
        if (m.tool) pending.push({ ...m.tool, done: true, open: false })
        continue
      }
      if (m.role === 'assistant') {
        out.push({
          role: 'assistant',
          content: m.content,
          reasoning: '',
          showThink: false,
          steps: pending.length ? pending : undefined,
          usage: m.usage,
        })
        pending = []
      } else {
        pending = [] // user 行打断未挂靠的孤儿步骤 (正常不会出现)
        out.push({ role: 'user', content: m.content, reasoning: '', showThink: false })
      }
    }
    bubbles.value = out
    scrollBottom()
  } catch {}
}

// 切换历史会话: 终止进行中的流 + 恢复该会话历史 (重复点击当前会话 = 从服务端刷新)
function selectSession(sessionId: string) {
  stop()
  setActiveSession(sessionId)
  bubbles.value = []
  loadHistory(sessionId)
}

// 新建对话: 换新 sessionId (服务端旧会话随空闲过期) + 清空界面
function handleNewConversation() {
  stop()
  setActiveSession(newSessionId())
  bubbles.value = []
}

// 删除历史会话 (删除后若正处于该会话, 顺延到最近会话或新开一个)
async function removeSession(s: AiSessionItem) {
  try {
    await ElMessageBox.confirm(`删除对话「${s.title || '新对话'}」? 删除后不可恢复。`, '删除对话', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  try {
    await api.deleteSession(s.sessionId)
    ElMessage.success('已删除')
  } catch {
    return // 失败提示已由 useRequest 弹出
  }
  await loadSessions()
  if (s.sessionId === activeSessionId.value) {
    if (sessions.value.length) selectSession(sessions.value[0].sessionId)
    else handleNewConversation()
  }
}

function scrollBottom() {
  nextTick(() => {
    const el = listRef.value
    if (el) el.scrollTop = el.scrollHeight
  })
}

function stop() {
  abort?.abort()
}

async function send() {
  const message = input.value.trim()
  if (!message || streaming.value) return
  if (notConfigured.value) {
    ElMessage.warning('AI 服务未配置, 请联系管理员在 全局配置 中设置 ai.api_key')
    return
  }
  input.value = ''
  bubbles.value.push({ role: 'user', content: message })
  // 注意: 必须是 reactive 代理对象 —— 回调里持续改 content/reasoning, 若是普通对象,
  // 模板经数组代理读取虽能拿到新值, 但原始引用的修改不触发更新, 界面会积攒到流结束一次性渲染
  const reply = reactive<Bubble>({ role: 'assistant', content: '', reasoning: '', showThink: true, streaming: true })
  bubbles.value.push(reply)
  streaming.value = true
  scrollBottom()

  abort = new AbortController()
  try {
    await api.chatStream(
      { sessionId: activeSessionId.value, message },
      {
        signal: abort.signal,
        onReasoning: (t) => {
          reply.reasoning = (reply.reasoning || '') + t
          scrollBottom()
        },
        onToolStart: (s) => {
          // 工具开始执行: 步骤条出现并转圈 (正文尚未开始)
          if (!reply.steps) reply.steps = []
          reply.steps.push({ id: s.id, name: s.name, args: s.args, done: false, open: false })
          scrollBottom()
        },
        onToolEnd: (s) => {
          const st = reply.steps?.find(x => x.id === s.id)
          if (st) {
            st.result = s.result
            st.error = s.error
            st.done = true
          }
          scrollBottom()
        },
        onUsage: (u) => {
          reply.usage = u
        },
        onDelta: (t) => {
          // 正文开始: 思考面板自动折叠 (仍可手动展开)
          if (reply.showThink && reply.content === '') reply.showThink = false
          reply.content += t
          scrollBottom()
        },
        onError: (msg) => {
          reply.error = true
          reply.content = msg
          ElMessage.error(msg)
        },
      },
    )
  } catch (e: any) {
    if (e?.name === 'AbortError') {
      reply.content = reply.content || '（已停止生成）'
    } else {
      reply.error = true
      reply.content = e?.message || '请求失败'
      ElMessage.error(reply.content)
    }
  } finally {
    reply.streaming = false
    streaming.value = false
    abort = null
    scrollBottom()
    // 本轮成功: 服务端已将该会话写入用户会话索引, 刷新左侧列表 (新会话入列/活跃时间更新)
    if (!reply.error) loadSessions()
  }
}

function onInputKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    send()
  }
}

// 会话时间显示: 今天 HH:mm / 昨天 / 今年 MM-DD / 往年 YYYY-MM-DD
function fmtTime(unixSec: number) {
  if (!unixSec) return ''
  const d = new Date(unixSec * 1000)
  const now = new Date()
  const hm = `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
  const sameDay = (a: Date, b: Date) =>
    a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate()
  if (sameDay(d, now)) return `今天 ${hm}`
  const yesterday = new Date(now)
  yesterday.setDate(now.getDate() - 1)
  if (sameDay(d, yesterday)) return `昨天 ${hm}`
  const md = `${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
  return d.getFullYear() === now.getFullYear() ? md : `${d.getFullYear()}-${md}`
}

onMounted(async () => {
  loadCfg()
  await loadSessions()
  // 恢复上次会话: 优先 localStorage 记录 (仍存活), 其次最近会话, 否则新开
  const saved = import.meta.client ? localStorage.getItem(SESSION_KEY) : ''
  if (saved && sessions.value.some(s => s.sessionId === saved)) {
    selectSession(saved)
  } else if (sessions.value.length) {
    selectSession(sessions.value[0].sessionId)
  } else {
    handleNewConversation()
  }
})
</script>

<template>
  <div class="page ai-page">
    <div class="chat-layout">
      <!-- 左侧: 新对话 + 分割线 + 历史会话列表 -->
      <aside class="session-panel">
        <el-button type="primary" class="new-btn" :icon="Plus" @click="handleNewConversation">新对话</el-button>
        <el-divider class="session-divider">历史对话</el-divider>

        <div v-loading="sessionsLoading" class="session-list">
          <div
            v-for="s in sessions"
            :key="s.sessionId"
            class="session-item"
            :class="{ active: s.sessionId === activeSessionId }"
            @click="selectSession(s.sessionId)"
          >
            <el-icon class="session-icon"><ChatDotRound /></el-icon>
            <div class="session-info">
              <div class="session-title">{{ s.title || '新对话' }}</div>
              <div class="session-time">{{ fmtTime(s.updatedAt) }}</div>
            </div>
            <el-tooltip content="删除对话" placement="top">
              <el-icon class="session-del" @click.stop="removeSession(s)"><Delete /></el-icon>
            </el-tooltip>
          </div>
          <div v-if="!sessions.length && !sessionsLoading" class="session-empty">暂无历史对话</div>
        </div>
      </aside>

      <!-- 右侧: 对话区 (消息 + 输入) -->
      <el-card class="chat-card">
        <template #header>
          <div class="chat-head">
            <div class="chat-title">
              <span class="chat-session-name" :title="activeTitle">{{ activeTitle }}</span>
              <el-tag v-if="cfg?.configured" size="small" type="success">{{ cfg.model }}</el-tag>
              <el-tag v-else-if="cfgLoaded" size="small" type="info">未配置</el-tag>
            </div>
          </div>
        </template>

        <el-alert v-if="notConfigured" type="warning" :closable="false" show-icon style="margin-bottom:8px"
          title="AI 服务尚未配置"
          description="请管理员在 系统管理-全局配置 中填写 ai.api_key (OpenAI 兼容接口), 可选调整 ai.base_url / ai.model。" />

        <!-- 消息区 -->
        <div ref="listRef" class="message-list">
          <el-empty v-if="!bubbles.length" description="开始和 AI 对话吧" :image-size="80" />

          <div v-for="(b, i) in bubbles" :key="i" class="message-item" :class="b.role">
            <div class="message-avatar">
              <el-avatar v-if="b.role === 'user'" :size="34" class="avatar-user" :icon="User" />
              <el-avatar v-else :size="34" class="avatar-ai">AI</el-avatar>
            </div>
            <div class="message-content">
              <div class="message-bubble" :class="{ error: b.error, md: b.role === 'assistant' && !b.error }">
                <!-- 思考过程 (推理模型): 可折叠面板 -->
                <div v-if="b.reasoning" class="think-box">
                  <div class="think-toggle" @click="b.showThink = !b.showThink">
                    <span class="think-arrow" :class="{ open: b.showThink }">▸</span>
                    <span>思考过程</span>
                    <i v-if="b.streaming && !b.content" class="cursor">▌</i>
                  </div>
                  <div v-show="b.showThink" class="think-text">{{ b.reasoning }}</div>
                </div>

                <!-- 工具调用步骤: 执行中转圈 / 完成打勾 / 失败红叉, 点击展开参数与结果 -->
                <div v-if="b.steps && b.steps.length" class="tool-steps">
                  <div
                    v-for="st in b.steps"
                    :key="st.id"
                    class="tool-step"
                    :class="{ failed: st.error }"
                    @click="st.open = !st.open"
                  >
                    <el-icon class="step-state" :class="{ spin: !st.done }">
                      <Loading v-if="!st.done" />
                      <CircleCloseFilled v-else-if="st.error" />
                      <CircleCheckFilled v-else />
                    </el-icon>
                    <span class="step-name">{{ st.name }}</span>
                    <span class="step-toggle">{{ st.open ? '收起' : '详情' }}</span>
                    <div v-show="st.open" class="step-detail">
                      <div v-if="st.args"><span class="lbl">参数</span>{{ st.args }}</div>
                      <div v-if="st.error" class="err"><span class="lbl">错误</span>{{ st.error }}</div>
                      <div v-else-if="st.result"><span class="lbl">结果</span>{{ st.result }}</div>
                    </div>
                  </div>
                </div>

                <div v-if="b.role === 'user' || b.error" class="plain-text">{{ b.content }}</div>
                <div v-else-if="b.content" class="markdown-body" v-html="renderMd(b.content)" />

                <!-- 等待首帧: 三点动画 (工具步骤已在展示时让位) -->
                <div v-if="b.role === 'assistant' && b.streaming && !b.content && !b.reasoning && !(b.steps && b.steps.length)" class="typing">
                  <span class="dot" /><span class="dot" /><span class="dot" />
                </div>
                <i v-if="b.streaming && b.content" class="cursor tail-cursor">▌</i>

                <!-- token 用量: 收尾后展示 (网关未回报 total=0 时隐藏; 历史回放同样展示) -->
                <div
                  v-if="b.role === 'assistant' && !b.streaming && b.usage && b.usage.total > 0"
                  class="usage-line"
                >
                  <el-icon><Lightning /></el-icon>
                  <span>共 {{ fmtNum(b.usage.total) }} tokens（输入 {{ fmtNum(b.usage.prompt) }} / 输出 {{ fmtNum(b.usage.completion) }}）</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 输入区 -->
        <div class="input-area">
          <el-input v-model="input" type="textarea" :autosize="{ minRows: 1, maxRows: 5 }"
            placeholder="输入你的问题, Enter 发送 / Shift+Enter 换行" maxlength="8000" @keydown="onInputKeydown" />
          <el-button v-if="!streaming" type="primary" :icon="Promotion" :disabled="!input.trim()" @click="send">发送</el-button>
          <el-button v-else type="warning" :icon="VideoPause" @click="stop">停止</el-button>
        </div>

        <!-- AI 生成内容提示 -->
        <div class="ai-disclaimer">
          <el-icon><InfoFilled /></el-icon>
          <span>内容由 AI 生成, 可能存在错误或过时信息, 请注意甄别</span>
        </div>
      </el-card>
    </div>
  </div>
</template>

<style scoped>
.ai-page {
  /* 高度跟随布局主区 (el-main) 自适应撑满: 版权页脚有无均精确适配,
     避免页面级滚动条与消息区滚动条叠加出现双滚动条 */
  height: 100%;
  overflow: hidden;
}
.chat-layout {
  height: 100%;
  display: flex;
  gap: 12px;
  min-width: 0;
}

/* ---------- 左侧会话栏 ---------- */
.session-panel {
  width: 250px;
  flex: none;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-card-border-radius);
  padding: 12px;
  overflow: hidden;
}
.new-btn {
  width: 100%;
}
.session-divider {
  margin: 12px 0 8px;
}
.session-divider :deep(.el-divider__text) {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.session-list {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-height: 0;
}
.session-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 8px;
  cursor: pointer;
  transition: background-color 0.15s;
}
.session-item:hover {
  background: var(--el-fill-color-light);
}
.session-item.active {
  background: var(--el-color-primary-light-9);
}
.session-icon {
  flex: none;
  color: var(--el-text-color-secondary);
  font-size: 15px;
}
.session-item.active .session-icon {
  color: var(--el-color-primary);
}
.session-info {
  flex: 1;
  min-width: 0;
}
.session-title {
  font-size: 13px;
  line-height: 1.5;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--el-text-color-primary);
}
.session-time {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
  line-height: 1.5;
}
.session-del {
  flex: none;
  color: var(--el-text-color-placeholder);
  font-size: 14px;
  opacity: 0;
  transition: opacity 0.15s, color 0.15s;
}
.session-item:hover .session-del {
  opacity: 1;
}
.session-del:hover {
  color: var(--el-color-danger);
}
.session-empty {
  padding: 18px 0;
  text-align: center;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

/* ---------- 右侧对话区 ---------- */
.chat-card {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.chat-card :deep(.el-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.chat-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.chat-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  min-width: 0;
}
.chat-session-name {
  max-width: 380px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ---------- 消息列表 (头像式布局) ---------- */
.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 2px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.message-item {
  display: flex;
  gap: 12px;
}
.message-item.user {
  flex-direction: row-reverse;
}
.avatar-user {
  background-color: var(--el-color-primary);
  color: #fff;
}
.avatar-ai {
  background-color: var(--el-color-success);
  color: #fff;
  font-size: 13px;
  font-weight: 700;
}
.message-content {
  max-width: 74%;
  min-width: 0;
}
.message-bubble {
  padding: 10px 16px;
  border-radius: 10px;
  line-height: 1.7;
  word-break: break-word;
  background: var(--el-fill-color-light);
  font-size: 14px;
}
.message-item.user .message-bubble {
  background: var(--el-color-primary);
  color: #fff;
  white-space: pre-wrap;
}
.message-bubble.error {
  background: var(--el-color-danger-light-9);
  color: var(--el-color-danger);
}

/* ---------- 思考过程面板 ---------- */
.think-box {
  margin-bottom: 8px;
  border-left: 3px solid var(--el-border-color);
  padding-left: 10px;
}
.think-toggle {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  user-select: none;
}
.think-arrow {
  display: inline-block;
  transition: transform 0.15s;
  font-size: 12px;
}
.think-arrow.open {
  transform: rotate(90deg);
}
.think-text {
  margin-top: 6px;
  font-size: 12px;
  line-height: 1.7;
  color: var(--el-text-color-secondary);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 260px;
  overflow-y: auto;
}

/* ---------- 工具调用步骤条 ---------- */
.tool-steps {
  margin-bottom: 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.tool-step {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 6px;
  padding: 5px 10px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-lighter);
  border-radius: 6px;
  cursor: pointer;
  user-select: none;
}
.tool-step.failed {
  color: var(--el-color-danger);
  background: var(--el-color-danger-light-9);
}
.step-state {
  flex: none;
  align-self: center;
  font-size: 13px;
}
.step-state:not(.spin) {
  color: var(--el-color-success);
}
.tool-step.failed .step-state {
  color: var(--el-color-danger);
}
.step-state.spin {
  color: var(--el-color-primary);
  animation: tool-spin 1s linear infinite;
}
@keyframes tool-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
.step-name {
  color: var(--el-text-color-regular);
  font-family: 'JetBrains Mono', 'Fira Code', Consolas, Monaco, monospace;
}
.step-toggle {
  margin-left: auto;
  color: var(--el-color-primary);
  opacity: 0.75;
}
.step-detail {
  flex-basis: 100%;
  margin-top: 4px;
  padding: 6px 8px;
  background: var(--el-bg-color);
  border-radius: 4px;
  line-height: 1.7;
  word-break: break-all;
  white-space: pre-wrap;
  user-select: text;
}
.step-detail .lbl {
  display: inline-block;
  margin-right: 8px;
  padding: 0 6px;
  border-radius: 3px;
  background: var(--el-fill-color);
  font-size: 11px;
}
.step-detail .err {
  color: var(--el-color-danger);
}

/* ---------- token 用量行 ---------- */
.usage-line {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 8px;
  padding-top: 6px;
  border-top: 1px dashed var(--el-border-color-lighter);
  font-size: 11px;
  color: var(--el-text-color-placeholder);
  user-select: none;
}
.usage-line .el-icon {
  font-size: 12px;
}

/* ---------- Markdown 内容 ---------- */
.markdown-body {
  line-height: 1.8;
}
.markdown-body :deep(p) {
  margin: 0 0 10px;
}
.markdown-body :deep(p:last-child) {
  margin-bottom: 0;
}
.markdown-body :deep(h1),
.markdown-body :deep(h2) {
  font-size: 1.2em;
  font-weight: 700;
  margin: 16px 0 10px;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.markdown-body :deep(h3),
.markdown-body :deep(h4) {
  font-size: 1.05em;
  font-weight: 600;
  margin: 12px 0 6px;
}
/* 代码块: 暗色 */
.markdown-body :deep(pre) {
  background-color: #1e1e2e;
  border-radius: 8px;
  padding: 14px 16px;
  overflow-x: auto;
  margin: 12px 0;
}
.markdown-body :deep(pre code) {
  color: #cdd6f4;
  font-size: 13px;
  line-height: 1.6;
  font-family: 'JetBrains Mono', 'Fira Code', Consolas, Monaco, monospace;
  background: transparent;
  padding: 0;
}
/* 行内代码 */
.markdown-body :deep(code) {
  background-color: rgba(0, 0, 0, 0.06);
  border-radius: 4px;
  padding: 2px 6px;
  font-size: 13px;
  font-family: 'JetBrains Mono', 'Fira Code', Consolas, Monaco, monospace;
}
/* 列表 */
.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: 22px;
  margin: 10px 0;
}
.markdown-body :deep(li) {
  margin: 4px 0;
}
/* 引用 */
.markdown-body :deep(blockquote) {
  border-left: 4px solid var(--el-color-primary);
  background: var(--el-fill-color-lighter);
  padding: 8px 14px;
  margin: 12px 0;
  color: #595959;
  border-radius: 0 6px 6px 0;
}
/* 表格 */
.markdown-body :deep(table) {
  border-collapse: collapse;
  margin: 12px 0;
  border: 1px solid var(--el-border-color-lighter);
}
.markdown-body :deep(th),
.markdown-body :deep(td) {
  border: 1px solid var(--el-border-color-lighter);
  padding: 6px 12px;
  font-size: 13px;
  text-align: left;
}
.markdown-body :deep(th) {
  background: var(--el-fill-color-light);
  font-weight: 600;
}
.markdown-body :deep(tr:nth-child(even)) {
  background: var(--el-fill-color-lighter);
}
.markdown-body :deep(a) {
  color: var(--el-color-primary);
}
.markdown-body :deep(hr) {
  border: none;
  border-top: 1px solid var(--el-border-color-lighter);
  margin: 14px 0;
}

/* ---------- 等待三点动画 ---------- */
.typing {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 4px 0;
}
.typing .dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--el-text-color-secondary);
  animation: typing 1.4s infinite ease-in-out;
}
.typing .dot:nth-child(2) {
  animation-delay: 0.2s;
}
.typing .dot:nth-child(3) {
  animation-delay: 0.4s;
}
@keyframes typing {
  0%, 80%, 100% { transform: scale(0.6); opacity: 0.5; }
  40% { transform: scale(1); opacity: 1; }
}

/* ---------- 流式光标 ---------- */
.cursor {
  font-style: normal;
  animation: blink 0.9s infinite;
  margin-left: 1px;
  color: var(--el-text-color-secondary);
}
.tail-cursor {
  display: inline-block;
}
@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

/* ---------- 输入区 ---------- */
.input-area {
  display: flex;
  align-items: flex-end;
  gap: 10px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.input-area .el-button {
  flex: none;
}

/* ---------- AI 生成内容提示 ---------- */
.ai-disclaimer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  margin-top: 8px;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  user-select: none;
}
.ai-disclaimer .el-icon {
  font-size: 13px;
}
</style>
