/**
 * 消息通知 API。
 */
import { useRequest } from '~/composables/useRequest'

export interface MessageItem {
  id: number
  type: number
  title: string
  content: string
  level: number
  status: number
  senderId: number
  senderName: string
  targetScope: number
  receiverId: number
  receiverName: string
  targetRoleIds: number[]
  targetUserIds: number[]
  targetRoleNames: string[]
  isRead: boolean
  createdAt: string
  updatedAt: string
}

export interface MessageUnreadCount {
  total: number
  system: number
  private: number
}

export function useMessageApi() {
  const r = useRequest()
  const base = '/message'
  return {
    // 管理员列表
    list: (params: { type?: number; keyword?: string; level?: number; status?: number; page?: number; pageSize?: number }) =>
      r.get<{ list: MessageItem[]; total: number }>(base, params),
    // 我的收件箱
    inbox: (params: { type?: number; isRead?: number; keyword?: string; page?: number; pageSize?: number }) =>
      r.get<{ list: MessageItem[]; total: number }>(`${base}/inbox`, params),
    // 阅读 (自动已读)
    inboxRead: (id: number) =>
      r.get<MessageItem>(`${base}/inbox/${id}`),
    // 详情 (管理员视角)
    detail: (id: number) =>
      r.get<MessageItem>(`${base}/${id}`),
    // 发布系统通知
    createSystem: (data: { title: string; content: string; level?: number; status?: number; targetScope: number; targetIds?: number[] }) =>
      r.post<{ id: number }>(`${base}/system`, data),
    // 发送私信
    createPrivate: (data: { title: string; content: string; level?: number; receiverId: number }) =>
      r.post<{ id: number }>(`${base}/private`, data),
    // 管理员删除
    remove: (id: number) =>
      r.del(`${base}/${id}`),
    // 个人删除 (从收件箱隐藏)
    inboxRemove: (id: number) =>
      r.del(`${base}/inbox/${id}`),
    // 标记已读
    markRead: (id: number) =>
      r.put(`${base}/${id}/read`),
    // 全部标记已读
    markReadAll: () =>
      r.put<{ affected: number }>(`${base}/read-all`),
    // 未读数量
    unreadCount: () =>
      r.get<MessageUnreadCount>(`${base}/unread-count`),
  }
}
