/**
 * 认证与个人中心 API。
 */
import { useRequest } from '~/composables/useRequest'
import type { LoginUser, MenuNode } from '~/stores/user'

/** 登录响应: 开启两步验证的用户第一步只返回 need2fa + ticket, 需再调 loginTotp 换 token */
export interface LoginResult {
  need2fa?: boolean
  ticket?: string
  token?: string
  expireAt?: number
  userInfo?: LoginUser
}

export function useAuthApi() {
  const r = useRequest()
  return {
    publicKey: () =>
      r.get<{ keyId: string; publicKey: string }>('/auth/public-key'),
    login: (username: string, password: string, keyId: string) =>
      r.post<LoginResult>('/auth/login', { username, password, keyId }),
    /** 查询注册开关 (公开接口, 登录页据此决定是否展示注册入口; silent 失败不弹提示) */
    registerStatus: () =>
      r.get<{ allowRegister: boolean }>('/auth/register/status', undefined, { silent: true }),
    /** 用户注册 (密码须用 publicKey 返回的公钥加密后提交, 注册成功后跳登录页) */
    register: (username: string, password: string, keyId: string, nickname?: string) =>
      r.post<void>('/auth/register', { username, password, keyId, nickname }),
    /** 两步验证登录第二步: 票据 + 动态码换 token */
    loginTotp: (ticket: string, code: string) =>
      r.post<{ token: string; expireAt: number; userInfo: LoginUser }>(
        '/auth/login/totp', { ticket, code },
      ),
    /** 生成/重置两步验证绑定密钥 (返回 otpauth URI 供渲染二维码) */
    totpSetup: () =>
      r.get<{ secret: string; otpauth: string }>('/auth/totp/setup'),
    /** 确认绑定两步验证 */
    totpEnable: (code: string) => r.put('/auth/totp/enable', { code }),
    /** 解绑两步验证 (须提供当前动态码) */
    totpDisable: (code: string) => r.put('/auth/totp/disable', { code }),
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
