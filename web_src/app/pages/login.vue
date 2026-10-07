<script setup lang="ts">
/**
 * 登录页
 * - 第一步: 用户名 + 密码 (RSA 加密传输)
 * - 第二步: 开启了两步验证 (TOTP) 的用户, 输入验证器 App 的 6 位动态码
 */
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { User, Lock, Histogram, Right, Back, Iphone, CircleCheckFilled } from '@element-plus/icons-vue'
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
onMounted(() => {
  configStore.restore()
  startCountUp()
})

/* ---- 品牌区插画: "今日登录"数字滚动计数 ---- */
const statNum = ref(0)
const STAT_TARGET = 1284

function startCountUp() {
  // 减少动效偏好: 直接展示最终值
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    statNum.value = STAT_TARGET
    return
  }
  const duration = 1300
  const t0 = performance.now()
  const tick = (t: number) => {
    const p = Math.min(1, (t - t0) / duration)
    // easeOutCubic 缓动
    statNum.value = Math.round(STAT_TARGET * (1 - Math.pow(1 - p, 3)))
    if (p < 1) requestAnimationFrame(tick)
  }
  requestAnimationFrame(tick)
}

const statText = computed(() => statNum.value.toLocaleString())

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
    <!-- 品牌区: 蓝色渐变面板 + 纯 CSS 迷你仪表盘插画 (题材取自产品本身) -->
    <aside class="brand-panel">
      <div class="brand-head">
        <div class="brand-badge">
          <img v-if="siteLogo" :src="siteLogo" class="brand-logo-img" alt="logo">
          <el-icon v-else :size="20" class="brand-logo-icon"><Histogram /></el-icon>
        </div>
        <span class="brand-name">{{ siteName }}</span>
      </div>

      <div class="brand-hero">
        <h1 class="brand-title">让繁琐的后台事务,<br>清爽有序</h1>
        <p class="brand-desc">基于 Go + Nuxt 的全栈管理脚手架, RBAC 权限、多标签页、代码生成开箱即用</p>

        <div class="mini-dash" aria-hidden="true">
          <div class="mini-card">
            <div class="mini-card-head">
              <span class="mini-title">本周动态</span>
              <span class="mini-live"><i />实时</span>
            </div>
            <div class="mini-bars">
              <i
                v-for="(h, i) in [42, 68, 50, 84, 60, 95, 74]"
                :key="i"
                :style="{ '--h': h + '%', '--i': i }"
              />
            </div>
            <div class="mini-card-foot">
              <span class="mini-note">权限管得清, 表格查得快</span>
            </div>
          </div>

          <!-- 悬浮消息 toast: 倾斜俏皮感 (内容取社区版共有的消息中心场景) -->
          <div class="mini-toast">
            <el-icon :size="22" class="mini-toast-icon"><CircleCheckFilled /></el-icon>
            <div class="mini-toast-text">
              <b>公告已推送</b>
              <span>全员通知 · 2 分钟前</span>
            </div>
          </div>

          <div class="mini-stat">
            <div class="mini-stat-text">
              <span class="mini-stat-num">{{ statText }}</span>
              <span class="mini-stat-label">今日登录</span>
            </div>
            <svg class="mini-spark" viewBox="0 0 96 32" fill="none">
              <polyline
                points="0,25 14,19 28,22 42,11 56,15 70,7 84,11 96,4"
                stroke="#409eff"
                stroke-width="2.5"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
              <circle class="mini-spark-dot" r="3.5" fill="#409eff">
                <animateMotion
                  dur="3s"
                  repeatCount="indefinite"
                  path="M0,25 L14,19 L28,22 L42,11 L56,15 L70,7 L84,11 L96,4"
                />
              </circle>
            </svg>
          </div>
        </div>
      </div>

      <!-- 功能词滚动带: 慢速, 低透明度, 纯装饰 -->
      <div class="brand-marquee" aria-hidden="true">
        <div class="marquee-track">
          <template v-for="n in 2" :key="n">
            <span v-for="w in ['RBAC 权限', '多标签页', '代码生成', '操作日志', '数据字典', '定时任务', '消息中心', '在线用户']" :key="n + '-' + w" class="marquee-word">{{ w }}</span>
          </template>
        </div>
      </div>

      <div v-if="copyright" class="brand-foot">{{ copyright }}</div>
    </aside>

    <!-- 表单区: 浅色面板, 表单直接落地不套卡片 -->
    <main class="form-panel">
      <div class="form-box">
        <!-- 窄屏时品牌区隐藏, 在表单上方补一行品牌标识 -->
        <div class="form-brand">
          <div class="brand-badge">
            <img v-if="siteLogo" :src="siteLogo" class="brand-logo-img" alt="logo">
            <el-icon v-else :size="20" class="brand-logo-icon"><Histogram /></el-icon>
          </div>
          <span class="form-brand-name">{{ siteName }}</span>
        </div>

        <!-- 第一步: 账号密码 -->
        <template v-if="step === 'password'">
          <h2 class="form-title">登录</h2>
          <p class="form-sub">欢迎回来, 请登录你的账号</p>
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
          <h2 class="form-title">两步验证</h2>
          <p class="form-sub">打开验证器 App, 输入 6 位动态码</p>
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

      <div v-if="copyright" class="form-copyright">{{ copyright }}</div>
    </main>
  </div>
</template>

<style scoped>
.login-page {
  display: grid;
  grid-template-columns: minmax(460px, 46%) 1fr;
  min-height: 100vh;
  background: #f6f8fc;
}

/* ---------- 品牌面板 ---------- */
.brand-panel {
  position: relative;
  display: flex;
  flex-direction: column;
  padding: 40px 48px 32px;
  overflow: hidden;
  color: #fff;
  background: linear-gradient(165deg, #3a86f5 0%, #2264d4 55%, #1a4db0 100%);
  user-select: none;
}
/* 两团克制的微光: 上亮下暗, 拉开蓝色渐变的层次 */
.brand-panel::before,
.brand-panel::after {
  content: '';
  position: absolute;
  border-radius: 50%;
  pointer-events: none;
}
.brand-panel::before {
  width: 520px;
  height: 520px;
  top: -180px;
  left: -140px;
  background: radial-gradient(circle, rgba(170, 215, 255, 0.28), transparent 65%);
}
.brand-panel::after {
  width: 460px;
  height: 460px;
  bottom: -160px;
  right: -120px;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.09), transparent 65%);
}

.brand-head {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 12px;
}
.brand-badge {
  width: 38px;
  height: 38px;
  flex-shrink: 0;
  border-radius: 11px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #6fbaff, #2f7ce0);
  box-shadow: 0 6px 16px rgba(20, 60, 140, 0.35);
}
.brand-logo-icon {
  color: #fff;
}
.brand-logo-img {
  width: 24px;
  height: 24px;
  object-fit: contain;
  border-radius: 5px;
}
.brand-name {
  font-size: 17px;
  font-weight: 600;
  letter-spacing: 0.5px;
}

.brand-hero {
  position: relative;
  z-index: 1;
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 32px 0;
}
.brand-title {
  margin: 0;
  font-size: 34px;
  line-height: 1.45;
  font-weight: 700;
  letter-spacing: 1px;
}
.brand-desc {
  margin: 14px 0 0;
  max-width: 26em;
  font-size: 14px;
  line-height: 1.9;
  color: rgba(255, 255, 255, 0.62);
}

/* ---------- 迷你仪表盘插画 ---------- */
.mini-dash {
  position: relative;
  margin-top: 44px;
  width: 340px;
  max-width: 100%;
  /* 给右上 toast 卡与右下统计卡留出溢出空间 */
  padding: 44px 36px 30px 0;
}
.mini-card {
  border-radius: 14px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(6px);
  padding: 18px 20px 16px;
}
.mini-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.mini-title {
  font-size: 13px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.9);
}
.mini-live {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.85);
}
/* 实时徽标的呼吸点 (与曲线游走点是全页仅有的两组循环动效) */
.mini-live i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #a5d8ff;
  animation: live-pulse 2s ease-in-out infinite;
}
@keyframes live-pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(165, 216, 255, 0.45); }
  50%      { box-shadow: 0 0 0 6px rgba(165, 216, 255, 0); }
}
.mini-bars {
  display: flex;
  align-items: flex-end;
  gap: 10px;
  height: 72px;
}
.mini-bars i {
  flex: 1;
  height: var(--h);
  border-radius: 4px 4px 2px 2px;
  background: linear-gradient(180deg, #cfe4ff, #8fb8f5);
  transform-origin: bottom;
  animation: bar-in 0.7s cubic-bezier(0.22, 1.2, 0.36, 1) calc(0.35s + var(--i) * 75ms) both;
}
.mini-bars i:nth-child(4) {
  background: linear-gradient(180deg, #ffffff, #d8e9ff);
}
.mini-bars i:nth-child(6) {
  background: linear-gradient(180deg, #8fc0ff, #3d7ef0);
}
@keyframes bar-in {
  from { transform: scaleY(0); }
  to   { transform: scaleY(1); }
}
.mini-card-foot {
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px dashed rgba(255, 255, 255, 0.12);
}
.mini-note {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.45);
}

/* 悬浮消息 toast: 白卡 + 绿勾, 微倾斜 */
.mini-toast {
  position: absolute;
  top: 0;
  right: 8px;
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 10px 14px 10px 10px;
  border-radius: 12px;
  background: #fff;
  color: #1f2d3d;
  box-shadow: 0 12px 28px rgba(5, 15, 35, 0.32);
  transform: rotate(2.5deg);
  animation: toast-in 0.55s cubic-bezier(0.22, 1.2, 0.36, 1) 1.25s both;
}
@keyframes toast-in {
  from { opacity: 0; transform: rotate(2.5deg) translateY(-12px) scale(0.9); }
  to   { opacity: 1; transform: rotate(2.5deg) translateY(0) scale(1); }
}
.mini-toast-icon {
  color: #2fbf71;
}
.mini-toast-text {
  display: flex;
  flex-direction: column;
  line-height: 1.4;
}
.mini-toast-text b {
  font-size: 13px;
}
.mini-toast-text span {
  font-size: 11px;
  color: #8a94a6;
}

/* 悬浮统计卡: 白卡压在主卡右下角, 制造层次 */
.mini-stat {
  position: absolute;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 12px;
  background: #fff;
  color: #1f2d3d;
  box-shadow: 0 14px 32px rgba(5, 15, 35, 0.35);
  animation: stat-in 0.55s cubic-bezier(0.22, 1.2, 0.36, 1) 0.9s both;
}
@keyframes stat-in {
  from { opacity: 0; transform: translateY(14px) scale(0.94); }
  to   { opacity: 1; transform: translateY(0) scale(1); }
}
.mini-stat-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.mini-stat-num {
  font-size: 20px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.3px;
}
.mini-stat-label {
  font-size: 12px;
  color: #8a94a6;
}
.mini-spark {
  width: 96px;
  height: 32px;
}
.mini-spark polyline {
  stroke-dasharray: 200;
  stroke-dashoffset: 200;
  animation: spark-draw 1.4s ease-out 1.1s forwards;
}
@keyframes spark-draw {
  to { stroke-dashoffset: 0; }
}
/* 曲线上游走的蓝光点 (SMIL animateMotion, 跟随 polyline 轨迹) */
.mini-spark-dot {
  filter: drop-shadow(0 0 4px rgba(64, 158, 255, 0.9));
}

/* ---------- 功能词滚动带 ---------- */
.brand-marquee {
  position: relative;
  z-index: 1;
  overflow: hidden;
  padding: 10px 0;
  margin-bottom: 8px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  -webkit-mask-image: linear-gradient(90deg, transparent, #000 12%, #000 88%, transparent);
  mask-image: linear-gradient(90deg, transparent, #000 12%, #000 88%, transparent);
}
.marquee-track {
  display: inline-flex;
  white-space: nowrap;
  animation: marquee 36s linear infinite;
}
@keyframes marquee {
  from { transform: translateX(0); }
  to   { transform: translateX(-50%); }
}
.marquee-word {
  display: inline-flex;
  align-items: center;
  font-size: 12px;
  letter-spacing: 1px;
  color: rgba(255, 255, 255, 0.38);
}
/* 词间小圆点分隔 */
.marquee-word::after {
  content: '';
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.35);
  margin: 0 18px;
}

.brand-foot {
  position: relative;
  z-index: 1;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.35);
}

/* ---------- 表单面板 ---------- */
.form-panel {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 24px 72px;
}
.form-box {
  width: 368px;
  max-width: 100%;
  animation: form-in 0.5s ease-out 0.15s both;
}
@keyframes form-in {
  from { opacity: 0; transform: translateY(18px); }
  to   { opacity: 1; transform: translateY(0); }
}

/* 窄屏备用品牌行 (宽屏隐藏) */
.form-brand {
  display: none;
  align-items: center;
  gap: 10px;
  margin-bottom: 28px;
}
.form-brand-name {
  font-size: 16px;
  font-weight: 600;
  color: #1f2d3d;
}

.form-title {
  margin: 0;
  font-size: 28px;
  font-weight: 700;
  color: #16233c;
  letter-spacing: 0.5px;
}
.form-sub {
  margin: 10px 0 30px;
  font-size: 14px;
  color: #7a8699;
}

/* 输入框: 圆角 + 白底, 聚焦时青色光晕 */
.form-box :deep(.el-input__wrapper) {
  border-radius: 10px;
  background: #fff;
  transition: box-shadow 0.2s;
}
.form-box :deep(.el-input__wrapper.is-focus) {
  box-shadow:
    0 0 0 1px var(--el-color-primary) inset,
    0 0 0 4px rgba(64, 158, 255, 0.15);
}
.form-box :deep(.el-form-item__label) {
  font-weight: 500;
  color: #3d4a5f;
}

/* 主按钮: 实心主色, hover 加深而非提亮, 去掉渐变 */
.form-box :deep(.el-button.login-btn) {
  width: 100%;
  height: 46px;
  margin-top: 4px;
  font-size: 15px;
  letter-spacing: 2px;
  border: none;
  border-radius: 10px;
  background: var(--el-color-primary);
  box-shadow: 0 6px 16px rgba(64, 158, 255, 0.28);
  transition: transform 0.15s, background-color 0.2s, box-shadow 0.2s;
}
.form-box :deep(.el-button.login-btn:hover:not(.is-disabled):not(.is-loading)) {
  background: #3375d6;
  box-shadow: 0 8px 20px rgba(64, 158, 255, 0.36);
  transform: translateY(-1px);
}
.form-box :deep(.el-button.login-btn:active:not(.is-disabled):not(.is-loading)) {
  background: #2b68c2;
  transform: translateY(0);
  box-shadow: 0 4px 12px rgba(64, 158, 255, 0.24);
}
/* hover 时按钮箭头右滑 */
.form-box :deep(.el-button.login-btn .el-icon) {
  transition: transform 0.2s;
}
.form-box :deep(.el-button.login-btn:hover:not(.is-disabled):not(.is-loading) .el-icon) {
  transform: translateX(3px);
}
/* 次按钮 (返回重新登录): 描边风格 */
.form-box :deep(.el-button.login-btn-plain) {
  margin-top: 10px;
  margin-left: 0;
  height: 44px;
  font-size: 14px;
  border-radius: 10px;
  background: transparent;
  border: 1px solid var(--el-border-color);
  box-shadow: none;
}
.form-box :deep(.el-button.login-btn-plain:hover) {
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
.form-copyright {
  position: absolute;
  bottom: 22px;
  left: 0;
  right: 0;
  text-align: center;
  font-size: 12px;
  color: #98a3b5;
  /* 宽屏由品牌面板底部展示版权, 此处只作窄屏兜底 */
  display: none;
}

/* ---------- 矮视口: 压缩标题与插画, 避免品牌面板底部被裁 ----------
   面板是 overflow:hidden 的 grid 项, 高度锁在视口高, 内容超出不会触发页面滚动 */
@media (max-height: 760px) {
  .brand-panel {
    padding: 28px 40px 20px;
  }
  .brand-hero {
    padding: 20px 0;
  }
  .brand-title {
    font-size: 28px;
  }
  .brand-desc {
    margin-top: 10px;
    line-height: 1.7;
  }
  .mini-dash {
    margin-top: 26px;
    padding: 38px 32px 24px 0;
  }
  .mini-card {
    padding: 14px 18px 12px;
  }
  .mini-card-head {
    margin-bottom: 12px;
  }
  .mini-bars {
    height: 58px;
  }
  .mini-card-foot {
    margin-top: 10px;
    padding-top: 9px;
  }
}
/* 极矮视口: 收起插画保底 */
@media (max-height: 600px) {
  .mini-dash {
    display: none;
  }
}

/* ---------- 窄屏: 收掉品牌面板, 表单居中 ---------- */
@media (max-width: 960px) {
  .login-page {
    grid-template-columns: 1fr;
  }
  .brand-panel {
    display: none;
  }
  .form-brand {
    display: flex;
  }
  .form-brand .brand-badge {
    box-shadow: 0 4px 12px rgba(64, 158, 255, 0.25);
  }
  .form-copyright {
    display: block;
  }
}

/* 无障碍: 用户偏好减少动效时关闭全部装饰动画 */
@media (prefers-reduced-motion: reduce) {
  .form-box,
  .mini-bars i,
  .mini-stat,
  .mini-toast,
  .mini-live i,
  .marquee-track {
    animation: none;
  }
  .mini-toast {
    transform: rotate(2.5deg);
  }
  .mini-spark polyline {
    stroke-dashoffset: 0;
  }
  .mini-spark-dot {
    display: none;
  }
}
</style>
