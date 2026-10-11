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
    /** 获取图形验证码 (公开接口, 一次调用同时探测 sys.captcha_enable 开关与取图; silent 失败不弹提示) */
    captcha: () =>
      r.get<{ captchaEnabled: boolean; captchaId: string; image: string }>(
        '/auth/captcha', undefined, { silent: true },
      ),
    /** 登录 (账号支持用户名或手机号; 密码须用 publicKey 返回的公钥加密后提交;
     * tenantCode 多租户开启时的租户识别, 缺省走 X-Tenant-Code 头/平台租户) */
    login: (account: string, password: string, keyId: string, captchaId?: string, captchaCode?: string, tenantCode?: string) =>
      r.post<LoginResult>('/auth/login', { username: account, password, keyId, captchaId, captchaCode, tenantCode }),
    /** 查询注册开关 (公开接口; multiTenant 随此口下发供登录/注册页渲染租户码输入) */
    registerStatus: () =>
      r.get<{ allowRegister: boolean, multiTenant: boolean }>('/auth/register/status', undefined, { silent: true }),
    /** 用户注册 (密码须用 publicKey 返回的公钥加密后提交, 注册成功后跳登录页) */
    register: (username: string, password: string, keyId: string, nickname?: string, captchaId?: string, captchaCode?: string, tenantCode?: string) =>
      r.post<void>('/auth/register', { username, password, keyId, nickname, captchaId, captchaCode, tenantCode }),
    /** 公开租户信息 (仅显示名, 按 IP 限流防枚举; 登录页租户名提示条) */
    tenantInfo: (code: string) =>
      r.get<{ multiTenant: boolean, name: string, found: boolean, enabled: boolean }>(
        '/auth/tenant/info', { code }, { silent: true },
      ),
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
