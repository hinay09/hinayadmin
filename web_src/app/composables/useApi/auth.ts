/**
 * 认证与个人中心 API。
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
