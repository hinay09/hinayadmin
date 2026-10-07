<script setup lang="ts">
/**
 * 登录页
 * - 第一步: 用户名 + 密码 (RSA 加密传输)
 * - 第二步: 开启了两步验证 (TOTP) 的用户, 输入验证器 App 的 6 位动态码
 */
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { User, Lock, Histogram, Right, Back, Iphone } from '@element-plus/icons-vue'
import { JSEncrypt } from 'jsencrypt'
import { storeToRefs } from 'pinia'
import { useUserStore, safeRedirect } from '~/stores/user'
import { useConfigStore } from '~/stores/config'
import { useAuthApi } from '~/composables/useApi'

definePageMeta({ layout: 'blank', title: '登录' })

const userStore = useUserStore()
const configStore = useConfigStore()
const { siteName, siteLogo, copyright } = storeToRefs(configStore)
const route = useRoute()
const router = useRouter()
const api = useAuthApi()

// 配置接口需登录后才能访问, 登录页读本地缓存展示(上次登录时写入), 无缓存回退默认值
onMounted(() => configStore.restore())

const formRef = ref<FormInstance>()
const totpFormRef = ref<FormInstance>()
const loading = ref(false)
// 安全: 不预填默认凭据; 默认账号提示仅存在于开发构建
const showDefaultHint = import.meta.dev
const form = reactive({
  username: '',
  password: '',
})
const rules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

/* ---- 两步验证 (TOTP) 第二步 ---- */
const step = ref<'password' | 'totp'>('password')
const totpTicket = ref('')
const totpForm = reactive({ code: '' })
const totpRules: FormRules = {
  code: [
    { required: true, message: '请输入动态验证码', trigger: 'blur' },
    { pattern: /^\d{6}$/, message: '动态验证码为 6 位数字', trigger: 'blur' },
  ],
}

/** TOTP_TICKET_INVALID 后端票据失效错误码: 回到第一步重新走密码登录 */
const TOTP_CODE_TICKET_INVALID = 50021

/** RSA_CODE_KEY_INVALID 后端一次性密钥失效的错误码, 此时静默换新密钥重试一次 */
const RSA_CODE_KEY_INVALID = 50005

/** 取一次性公钥并加密密码, 返回密文与 keyId */
async function encryptPassword(plain: string) {
  const { keyId, publicKey } = await api.publicKey()
  const encryptor = new JSEncrypt()
  encryptor.setPublicKey(publicKey)
  const cipher = encryptor.encrypt(plain)
  if (!cipher) throw new Error('密码加密失败')
  return { keyId, cipher }
}

async function doLogin() {
  const { keyId, cipher } = await encryptPassword(form.password)
  return api.login(form.username, cipher, keyId)
}

/** 登录收尾: 存 token / 用户信息 / 菜单并跳转 */
async function finishLogin(token: string, expireAt: number, userInfo: any) {
  userStore.setToken(token, expireAt)
  userStore.setUserInfo(userInfo)
  // 拉取菜单
  const menusRes = await api.menus()
  userStore.setMenus(menusRes.menus, menusRes.permissions)
  ElMessage.success('登录成功')
  // 开放跳转防护: 仅允许站内路径
  router.replace(safeRedirect(route.query.redirect as string))
}

async function handleSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    let res
    try {
      res = await doLogin()
    }
    catch (err: any) {
      // 一次性密钥过期/已用: 换新密钥重试一次, 仍失败则正常抛出
      if (err?.code !== RSA_CODE_KEY_INVALID) throw err
      res = await doLogin()
    }
    // 开启两步验证: 不发 token, 携带票据进入动态码输入步骤
    if (res.need2fa && res.ticket) {
      totpTicket.value = res.ticket
      totpForm.code = ''
      step.value = 'totp'
      return
    }
    // 未开启两步验证的常规路径; 异常响应缺 token 时兜底报错, 不写坏会话 store
    if (!res.token || !res.userInfo) {
      ElMessage.error('登录响应异常, 请重试')
      return
    }
    await finishLogin(res.token, res.expireAt!, res.userInfo)
  }
  catch {}
  finally {
    loading.value = false
  }
}

/** 两步验证第二步: 票据 + 动态码换 token */
async function handleTotpSubmit() {
  if (!totpFormRef.value) return
  const valid = await totpFormRef.value.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    const res = await api.loginTotp(totpTicket.value, totpForm.code)
    await finishLogin(res.token, res.expireAt, res.userInfo)
  }
  catch (err: any) {
    // 票据过期/失败次数超限: 回到密码登录重新走两步
    if (err?.code === TOTP_CODE_TICKET_INVALID) {
      totpTicket.value = ''
      step.value = 'password'
    }
  }
  finally {
    loading.value = false
  }
}

/** 返回密码登录第一步 */
function backToPassword() {
  totpTicket.value = ''
  totpForm.code = ''
  step.value = 'password'
}
</script>

<template>
  <div class="login-page">
    <!-- 背景装饰: 网格 + 三团彩色光晕 (纯 CSS, 无依赖) -->
    <div class="login-orb login-orb-a" aria-hidden="true" />
    <div class="login-orb login-orb-b" aria-hidden="true" />
    <div class="login-orb login-orb-c" aria-hidden="true" />

    <div class="login-card">
      <div class="login-brand">
        <div class="login-badge">
          <img v-if="siteLogo" :src="siteLogo" class="login-logo-img" alt="logo">
          <el-icon v-else :size="24" class="login-logo-icon"><Histogram /></el-icon>
        </div>
        <div class="login-title">{{ siteName }}</div>
      </div>

      <!-- 第一步: 账号密码 -->
      <template v-if="step === 'password'">
        <div class="login-sub">通用后台管理脚手架</div>
        <el-form
          ref="formRef"
          :model="form"
          :rules="rules"
          label-position="top"
          @keyup.enter="handleSubmit"
        >
          <el-form-item label="用户名" prop="username">
            <el-input v-model="form.username" size="large" placeholder="用户名" :prefix-icon="User" />
          </el-form-item>
          <el-form-item label="密码" prop="password">
            <el-input
              v-model="form.password"
              type="password"
              size="large"
              show-password
              placeholder="密码"
              :prefix-icon="Lock"
            />
          </el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            :icon="Right"
            class="login-btn"
            @click="handleSubmit"
          >
            登录
          </el-button>
        </el-form>
        <!-- 默认凭据提示仅在开发构建渲染, 生产构建(import.meta.dev=false)不输出 -->
        <div v-if="showDefaultHint" class="tips">默认账号: admin / 123456 (仅开发环境)</div>
      </template>

      <!-- 第二步: 两步验证动态码 -->
      <template v-else>
        <div class="login-sub">两步验证 · 请输入验证器 App 中的 6 位动态码</div>
        <el-form
          ref="totpFormRef"
          :model="totpForm"
          :rules="totpRules"
          label-position="top"
          @keyup.enter="handleTotpSubmit"
        >
          <el-form-item prop="code">
            <el-input
              v-model="totpForm.code"
              size="large"
              maxlength="6"
              inputmode="numeric"
              autocomplete="one-time-code"
              placeholder="6 位动态验证码"
              :prefix-icon="Iphone"
              class="totp-code-input"
            />
          </el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            :icon="Right"
            class="login-btn"
            @click="handleTotpSubmit"
          >
            验证并登录
          </el-button>
          <el-button
            size="large"
            :icon="Back"
            class="login-btn login-btn-plain"
            @click="backToPassword"
          >
            返回重新登录
          </el-button>
        </el-form>
        <div class="tips">动态码每 30 秒刷新一次, 与服务器时间偏差过大时会校验失败</div>
      </template>
    </div>
    <div v-if="copyright" class="login-copyright">{{ copyright }}</div>
  </div>
</template>

<style scoped>
.login-page {
  position: relative;
  min-height: 100vh;
  overflow: hidden;
  /* 明亮浅蓝渐变底, 光晕负责色彩层次 */
  background: linear-gradient(180deg, #ecf3fd 0%, #f6f9fe 55%, #f1f5fb 100%);
  display: flex;
  align-items: center;
  justify-content: center;
}
/* 细网格装饰: 径向蒙版让边缘淡出 */
.login-page::before {
  content: '';
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(64, 120, 200, 0.07) 1px, transparent 1px),
    linear-gradient(90deg, rgba(64, 120, 200, 0.07) 1px, transparent 1px);
  background-size: 44px 44px;
  -webkit-mask-image: radial-gradient(ellipse at center, #000 25%, transparent 72%);
  mask-image: radial-gradient(ellipse at center, #000 25%, transparent 72%);
  pointer-events: none;
}
/* 彩色光晕: 径向渐变圆 + 缓慢漂浮 */
.login-orb {
  position: absolute;
  border-radius: 50%;
  pointer-events: none;
  will-change: transform;
}
.login-orb-a {
  width: 560px;
  height: 560px;
  top: -160px;
  right: -120px;
  background: radial-gradient(circle, rgba(64, 158, 255, 0.28), transparent 65%);
  animation: login-orb-float 14s ease-in-out infinite alternate;
}
.login-orb-b {
  width: 640px;
  height: 640px;
  bottom: -220px;
  left: -180px;
  background: radial-gradient(circle, rgba(56, 182, 230, 0.22), transparent 65%);
  animation: login-orb-float 18s ease-in-out infinite alternate-reverse;
}
.login-orb-c {
  width: 380px;
  height: 380px;
  top: 38%;
  left: 56%;
  background: radial-gradient(circle, rgba(135, 120, 255, 0.14), transparent 65%);
  animation: login-orb-float 22s ease-in-out infinite alternate;
}
@keyframes login-orb-float {
  from { transform: translate3d(0, 0, 0); }
  to   { transform: translate3d(-40px, 30px, 0); }
}

.login-card {
  position: relative;
  z-index: 1;
  width: 400px;
  max-width: calc(100vw - 32px);
  background: #fff;
  padding: 40px 36px 32px;
  border-radius: 16px;
  border: 1px solid #e9f0fa;
  box-shadow:
    0 20px 50px rgba(31, 84, 179, 0.1),
    0 4px 16px rgba(31, 84, 179, 0.06);
  animation: login-card-in 0.45s ease-out both;
}
@keyframes login-card-in {
  from { opacity: 0; transform: translateY(24px); }
  to   { opacity: 1; transform: translateY(0); }
}

/* 品牌区: 徽标 + 系统名纵向居中 */
.login-brand {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  user-select: none;
}
.login-badge {
  width: 48px;
  height: 48px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #409eff, #2f6fd8);
  box-shadow: 0 8px 20px rgba(64, 158, 255, 0.35);
}
.login-logo-icon {
  color: #fff;
}
.login-logo-img {
  width: 28px;
  height: 28px;
  object-fit: contain;
  border-radius: 6px;
}
.login-title {
  font-size: 22px;
  font-weight: 600;
  color: #1f2d3d;
  letter-spacing: 0.5px;
}
.login-sub {
  font-size: 13px;
  color: #8a94a6;
  text-align: center;
  margin: 10px 0 28px;
}
.login-copyright {
  position: absolute;
  bottom: 24px;
  left: 0;
  right: 0;
  z-index: 1;
  text-align: center;
  font-size: 12px;
  letter-spacing: 0.5px;
  color: #98a3b5;
}

/* 输入框圆角化 */
.login-card :deep(.el-input__wrapper) {
  border-radius: 10px;
}
/* 主按钮: 满宽居中 + 渐变 + 悬浮微抬升 (提高选择器优先级以覆盖 el-button--primary) */
.login-card :deep(.el-button.login-btn) {
  width: 100%;
  height: 44px;
  font-size: 15px;
  letter-spacing: 4px;
  border: none;
  border-radius: 10px;
  background: linear-gradient(90deg, #409eff, #2f7ce0);
  box-shadow: 0 8px 20px rgba(64, 158, 255, 0.35);
  transition: transform 0.2s, box-shadow 0.2s, filter 0.2s;
}
.login-card :deep(.el-button.login-btn:hover:not(.is-disabled):not(.is-loading)) {
  transform: translateY(-1px);
  filter: brightness(1.05);
  box-shadow: 0 10px 24px rgba(64, 158, 255, 0.45);
}
.login-card :deep(.el-button.login-btn:active:not(.is-disabled):not(.is-loading)) {
  transform: translateY(0);
  box-shadow: 0 6px 16px rgba(64, 158, 255, 0.3);
}
/* 次按钮 (返回重新登录): 描边风格 */
.login-card :deep(.el-button.login-btn-plain) {
  margin-top: 10px;
  margin-left: 0;
  height: 44px;
  font-size: 14px;
  border-radius: 10px;
  background: transparent;
  border: 1px solid var(--el-border-color);
  box-shadow: none;
}
.login-card :deep(.el-button.login-btn-plain:hover) {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.totp-code-input :deep(input) {
  letter-spacing: 8px;
  font-size: 20px;
  text-align: center;
}
.tips {
  margin-top: 16px;
  font-size: 12px;
  color: #909399;
  text-align: center;
}

/* 无障碍: 用户偏好减少动效时关闭装饰动画 */
@media (prefers-reduced-motion: reduce) {
  .login-card,
  .login-orb-a,
  .login-orb-b,
  .login-orb-c {
    animation: none;
  }
}
</style>
