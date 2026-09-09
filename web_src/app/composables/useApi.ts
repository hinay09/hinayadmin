/**
 * 业务 API 集中点。
 */
import { useRequest } from '~/composables/useRequest'
import type { LoginUser, MenuNode } from '~/stores/user'

export function useAuthApi() {
  const r = useRequest()
  return {
    publicKey: () =>
      r.get<{ keyId: string; publicKey: string }>('/auth/public-key'),
    login: (username: string, password: string, keyId: string) =>
      r.post<{ token: string; expireAt: number; userInfo: LoginUser }>(
        '/auth/login', { username, password, keyId },
      ),
    logout: () => r.post('/auth/logout'),
    userInfo: () => r.get<LoginUser>('/auth/userInfo'),
    menus: () => r.get<{ menus: MenuNode[]; permissions: string[] }>(
      '/auth/menus',
    ),
    profile: () => r.get<LoginUser>('/auth/profile'),
    updateProfile: (data: { nickname: string; avatar?: string; email?: string; phone?: string }) =>
      r.put('/auth/profile', data),
    changePassword: (oldPassword: string, newPassword: string) =>
      r.put('/auth/password', { oldPassword, newPassword }),
    uploadAvatar: (file: File) => {
      const formData = new FormData()
      formData.append('file', file)
      return r.post<{ url: string }>('/auth/avatar', formData)
    },
  }
}

export function useUserApi() {
  const r = useRequest()
  const base = '/system/users'
  return {
    list: (params: any) => r.get(base, params),
    detail: (id: number) => r.get(`${base}/${id}`),
    create: (data: any) => r.post(base, data),
    update: (id: number, data: any) => r.put(`${base}/${id}`, data),
    remove: (id: number) => r.del(`${base}/${id}`),
    resetPwd: (id: number, password: string) =>
      r.put(`${base}/${id}/password`, { password }),
  }
}

export function useRoleApi() {
  const r = useRequest()
  const base = '/system/roles'
  return {
    list: (params: any) => r.get(base, params),
    all: () => r.get(`${base}/all`),
    detail: (id: number) => r.get(`${base}/${id}`),
    create: (data: any) => r.post(base, data),
    update: (id: number, data: any) => r.put(`${base}/${id}`, data),
    remove: (id: number) => r.del(`${base}/${id}`),
    assignMenus: (id: number, menuIds: number[]) =>
      r.put(`${base}/${id}/menus`, { menuIds }),
    getMenus: (id: number) =>
      r.get(`${base}/${id}/menus`),
    assignApis: (id: number, apis: Array<{ path: string; method: string }>) =>
      r.put(`${base}/${id}/apis`, { apis }),
    getApis: (id: number) =>
      r.get(`${base}/${id}/apis`),
  }
}

export function useMenuApi() {
  const r = useRequest()
  const base = '/system/menus'
  return {
    list: (params?: any) => r.get(base, params),
    tree: () => r.get(`${base}/tree`),
    detail: (id: number) => r.get(`${base}/${id}`),
    create: (data: any) => r.post(base, data),
    update: (id: number, data: any) => r.put(`${base}/${id}`, data),
    remove: (id: number) => r.del(`${base}/${id}`),
  }
}

export function useOrgApi() {
  const r = useRequest()
  const base = '/system/orgs'
  return {
    tree: () => r.get(`${base}/tree`),
    list: (params?: any) => r.get(base, params),
    detail: (id: number) => r.get(`${base}/${id}`),
    create: (data: any) => r.post(base, data),
    update: (id: number, data: any) => r.put(`${base}/${id}`, data),
    remove: (id: number) => r.del(`${base}/${id}`),
  }
}

export function useApiResourceApi() {
  const r = useRequest()
  const base = '/system/apis'
  return {
    list: (params?: { groupName?: string; path?: string; page?: number; pageSize?: number }) =>
      r.get<{ list: any[]; total: number }>(base, params),
    all: () => r.get<{ list: any[] }>(`${base}/all`),
    create: (data: { path: string; method: string; groupName?: string; description?: string }) =>
      r.post(base, data),
    update: (id: number, data: { path: string; method: string; groupName?: string; description?: string }) =>
      r.put(`${base}/${id}`, data),
    remove: (id: number) =>
      r.del(`${base}/${id}`),
  }
}

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

export function useAuditLogApi() {
  const r = useRequest()
  const base = '/system/audit-logs'
  return {
    list: (params: { keyword?: string; action?: string; startAt?: string; endAt?: string; page?: number; pageSize?: number }) =>
      r.get<{ list: any[]; total: number }>(base, params),
  }
}

export function useDictTypeApi() {
  const r = useRequest()
  const base = '/system/dict-types'
  return {
    list: (params: { keyword?: string; page?: number; pageSize?: number }) =>
      r.get<{ list: any[]; total: number }>(base, params),
    all: () => r.get<{ list: any[] }>(`${base}/all`),
    create: (data: { typeCode: string; typeName: string; status?: number; remark?: string }) =>
      r.post<{ id: number }>(base, data),
    update: (id: number, data: { typeCode: string; typeName: string; status?: number; remark?: string }) =>
      r.put(`${base}/${id}`, data),
    remove: (id: number) => r.del(`${base}/${id}`),
  }
}

export function useDictApi() {
  const r = useRequest()
  const base = '/system/dict-types'
  return {
    listByType: (typeId: number, params?: { keyword?: string; page?: number; pageSize?: number }) =>
      r.get<{ list: any[]; total: number }>(`${base}/${typeId}/items`, params),
    create: (typeId: number, data: { dictLabel: string; dictValue?: string; sort?: number; status?: number; remark?: string }) =>
      r.post<{ id: number }>(`${base}/${typeId}/items`, data),
    update: (typeId: number, id: number, data: { dictLabel: string; dictValue?: string; sort?: number; status?: number; remark?: string }) =>
      r.put(`${base}/${typeId}/items/${id}`, data),
    remove: (typeId: number, id: number) => r.del(`${base}/${typeId}/items/${id}`),
    sort: (typeId: number, items: Array<{ id: number; sort: number }>) =>
      r.put(`${base}/${typeId}/items/sort`, { items }),
    all: () => r.get<{ list: any }>('/system/dicts/all'),
  }
}

export function useFileApi() {
  const r = useRequest()
  const base = '/system/files'
  return {
    list: (params: { keyword?: string; mimeType?: string; page?: number; pageSize?: number }) =>
      r.get<{ list: any[]; total: number }>(base, params),
    upload: (file: File) => {
      const formData = new FormData()
      formData.append('file', file)
      return r.post<{ id: number; name: string; originalName: string; url: string; size: number; extension: string }>(
        `${base}/upload`, formData,
      )
    },
    remove: (id: number) => r.del(`${base}/${id}`),
  }
}

export interface ConfigItem {
  id: number
  configKey: string
  configValue: string
  configType: number
  name: string
  remark: string
  status: number
  sort: number
  createdAt: string
  updatedAt: string
}

export function useConfigApi() {
  const r = useRequest()
  const base = '/system/configs'
  return {
    list: (params: { keyword?: string; configType?: number; page?: number; pageSize?: number }) =>
      r.get<{ list: ConfigItem[]; total: number }>(base, params),
    all: () => r.get<{ list: Record<string, { value: string; type: number; name: string }> }>(`${base}/all`),
    create: (data: { configKey: string; configValue?: string; configType?: number; name: string; remark?: string; status?: number; sort?: number }) =>
      r.post<{ id: number }>(base, data),
    update: (id: number, data: { configKey: string; configValue?: string; configType?: number; name: string; remark?: string; status?: number; sort?: number }) =>
      r.put(`${base}/${id}`, data),
    remove: (id: number) => r.del(`${base}/${id}`),
  }
}
