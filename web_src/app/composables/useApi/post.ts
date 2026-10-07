/**
 * 岗位管理 API。
 */
import { useRequest } from '~/composables/useRequest'

export interface PostItem {
  id: number
  postCode: string
  postName: string
  /** 1=普通岗, 2=主管岗 (部门主管解析依据) */
  postKind: number
  sort: number
  status: number
  remark: string
  members?: number
  updateAt?: string
}

export interface PostMemberItem {
  id: number
  userId: number
  userName: string
  orgId: number
  orgName: string
  createAt: string
}

export function usePostApi() {
  const r = useRequest()
  return {
    list: (params: { keyword?: string, status?: number, page?: number, pageSize?: number }) =>
      r.get<{ list: PostItem[], total: number }>('/system/posts', params),
    all: () => r.get<{ list: PostItem[] }>('/system/posts/all'),
    create: (data: { postCode: string, postName: string, postKind: number, sort?: number, status?: number, remark?: string }) =>
      r.post<{ id: number }>('/system/posts', data),
    update: (id: number, data: { postCode: string, postName: string, postKind: number, sort?: number, status?: number, remark?: string }) =>
      r.put(`/system/posts/${id}`, data),
    remove: (id: number) => r.del(`/system/posts/${id}`),
    members: (id: number) => r.get<{ list: PostMemberItem[] }>(`/system/posts/${id}/members`),
    memberAdd: (id: number, data: { userId: number, orgId?: number }) =>
      r.post<{ id: number }>(`/system/posts/${id}/members`, data),
    memberRemove: (relId: number) => r.del(`/system/posts/members/${relId}`),
  }
}
