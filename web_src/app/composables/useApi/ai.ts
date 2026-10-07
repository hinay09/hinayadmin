/**
 * AI 助手 API。
 * 流式对话基于 @microsoft/fetch-event-source (POST SSE):
 * 携带鉴权头/不因切后台暂停/不自动重试; 渲染侧配合 markdown-it + highlight.js。
 */
import { fetchEventSource } from '@microsoft/fetch-event-source'
import { useRequest } from '~/composables/useRequest'
import { useUserStore } from '~/stores/user'

/** 工具调用步骤 (SSE toolStart/toolEnd 事件与历史回放共用) */
export interface AiToolStep {
  id: string
  name: string
  args?: string // 调用参数 (JSON 文本)
  result?: string // 执行结果 (JSON 文本)
  error?: string // 执行错误
}

/** 一轮对话的 token 用量 (网关未回报时全 0, 前端隐藏) */
export interface AiTokenUsage {
  prompt: number
  completion: number
  total: number
}

export interface AiHistoryMessage {
  role: 'user' | 'assistant' | 'tool'
  content: string // role=tool 时为空
  tool?: AiToolStep // role=tool: 步骤详情 (前端折叠到其后紧邻的 assistant 气泡)
  usage?: AiTokenUsage // role=assistant: 本轮 token 用量 (旧数据/网关未回报无)
}

export interface AiSessionItem {
  sessionId: string
  title: string
  updatedAt: number // 最后活跃时间 (Unix 秒)
}

export interface AiConfigStatus {
  configured: boolean
  baseUrl: string
  model: string
  keyMasked: string
}

export function useAiApi() {
  const r = useRequest()
  const config = useRuntimeConfig()
  const userStore = useUserStore()

  /**
   * 流式对话: 仅上送本轮 message + sessionId, 上下文由服务端按 sessionId 记忆。
   * 回调: onDelta(正文) / onReasoning(思考过程) / onToolStart·onToolEnd(工具调用步骤)
   *   / onUsage(本轮 token 用量, 成功收尾一次) / onError(服务端业务错误);
   *   连接/协议错误直接 throw。
   * 帧: {"content"} / {"reasoning"} / {"toolStart"} / {"toolEnd"} / {"usage"} / {"error"} / [DONE]。
   */
  async function chatStream(
    data: { sessionId: string, message: string },
    handlers: {
      onDelta: (t: string) => void,
      onReasoning?: (t: string) => void,
      onToolStart?: (s: AiToolStep) => void,
      onToolEnd?: (s: AiToolStep) => void,
      onUsage?: (u: AiTokenUsage) => void,
      onError?: (msg: string) => void,
      signal?: AbortSignal,
    },
  ): Promise<void> {
    userStore.restore()
    await fetchEventSource(`${config.public.apiBase}/ai/chat`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(userStore.token ? { Authorization: `Bearer ${userStore.token}` } : {}),
      },
      body: JSON.stringify(data),
      signal: handlers.signal,
      openWhenHidden: true, // 切后台继续接收
      async onopen(res) {
        if (res.ok && (res.headers.get('content-type') || '').includes('text/event-stream')) return
        // 非流式失败 (401/403/参数校验): 标准协议 JSON, 解出 message
        let msg = `请求失败 (${res.status})`
        try {
          const j: any = await res.json()
          msg = j.message || msg
        } catch {}
        throw new Error(msg)
      },
      onmessage(ev) {
        if (!ev.data || ev.data === '[DONE]') return // 收尾由服务端关流触发 onclose
        try {
          const j = JSON.parse(ev.data)
          if (j.error) {
            handlers.onError?.(j.error)
            return
          }
          if (j.content) handlers.onDelta(j.content)
          if (j.reasoning) handlers.onReasoning?.(j.reasoning)
          if (j.toolStart) handlers.onToolStart?.(j.toolStart)
          if (j.toolEnd) handlers.onToolEnd?.(j.toolEnd)
          if (j.usage) handlers.onUsage?.(j.usage)
        } catch { /* 忽略坏帧 */ }
      },
      onerror(err) {
        throw err // 网络错误不自动重试, 交给调用方提示
      },
    })
  }

  return {
    config: () => r.get<AiConfigStatus>('/ai/config'),
    history: (sessionId: string) => r.get<{ list: AiHistoryMessage[] }>('/ai/history', { sessionId }),
    sessions: () => r.get<{ list: AiSessionItem[] }>('/ai/sessions'),
    deleteSession: (sessionId: string) => r.del<{ }>('/ai/sessions', { sessionId }),
    chatStream,
  }
}
