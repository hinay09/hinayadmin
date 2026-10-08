<script setup lang="ts">
/**
 * 个人中心
 * - 左侧: 个人信息卡片(头像/昵称/账号/角色/邮箱/手机)
 * - 右侧: tabs - "基础资料" + "修改密码" + "安全设置"
 * 数据接口: GET/PUT /auth/profile, PUT /auth/password, /auth/totp/*
 */
import { reactive, ref, computed, onMounted } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import QRCode from 'qrcode'
import {
  User,
  Message,
  Phone,
  UserFilled,
  Lock,
  Edit,
  Key,
  CircleCheck,
  UploadFilled,
  Clock,
  Location,
} from '@element-plus/icons-vue'
import { useAuthApi } from '~/composables/useApi'
import { useUserStore } from '~/stores/user'
import { useConfigStore } from '~/stores/config'

definePageMeta({ title: '个人中心', layout: 'default' })
defineOptions({ name: 'profile' })

const api = useAuthApi()

// 角色标签: 优先使用后端返回的角色名称 (与角色code解耦), 缺失时回退展示code
const roleTags = computed<string[]>(() =>
  profile.roleNames.length ? profile.roleNames : profile.roles,
)
const userStore = useUserStore()

const route = useRoute()
const configStore = useConfigStore()
// 路由携带 ?tab=password 时直接定位改密页 (强制改密场景由路由守卫跳入)
const activeTab = ref<'basic' | 'password' | 'security'>(route.query.tab === 'password' ? 'password' : 'basic')

/* ---- 密码策略提示 (读取全局配置 sys.password.*, 与后端校验同源) ---- */
const pwdPolicyMinLen = computed(() => configStore.getNumber('sys.password.min_length', 6))
const pwdPolicyMaxLen = computed(() => configStore.getNumber('sys.password.max_length', 32))
const pwdPolicyItems = computed(() => {
  const items = [`长度 ${pwdPolicyMinLen.value}-${pwdPolicyMaxLen.value} 位`]
  if (configStore.getBool('sys.password.require_upper')) items.push('大写字母')
  if (configStore.getBool('sys.password.require_lower')) items.push('小写字母')
  if (configStore.getBool('sys.password.require_digit')) items.push('数字')
  if (configStore.getBool('sys.password.require_special')) items.push('特殊字符')
  return items
})
const loading = ref(false)
const submitting = ref(false)
const avatarUploading = ref(false)

interface ProfileForm {
  nickname: string
  avatar: string
  email: string
  phone: string
}
const profile = reactive<ProfileForm & { username: string; roles: string[]; roleNames: string[]; lastLoginAt: string; lastLoginIp: string }>({
  username: '',
  nickname: '',
  avatar: '',
  email: '',
  phone: '',
  roles: [],
  roleNames: [],
  lastLoginAt: '',
  lastLoginIp: '',
})

interface PasswordForm {
  oldPassword: string
  newPassword: string
  confirmPassword: string
}
const pwdForm = reactive<PasswordForm>({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})

const profileFormRef = ref<FormInstance>()
const pwdFormRef = ref<FormInstance>()

/* ---- 安全设置: TOTP 两步验证 ---- */
const twoFaEnabled = ref(false)
const bindDialog = ref(false)
const disableDialog = ref(false)
const totpSubmitting = ref(false)
const setupLoading = ref(false)
// 绑定二维码数据 (setup 接口返回)
const setupInfo = reactive({ secret: '', otpauth: '', qrDataUrl: '' })
const bindCode = ref('')
const disableCode = ref('')

const codeRule = [
  { required: true, message: '请输入动态验证码', trigger: 'blur' },
  { pattern: /^\d{6}$/, message: '动态验证码为 6 位数字', trigger: 'blur' },
]

/** 开启流程第一步: 拉取绑定密钥并渲染二维码 */
async function handleStartBind() {
  bindDialog.value = true
  setupLoading.value = true
  try {
    const res = await api.totpSetup()
    setupInfo.secret = res.secret
    setupInfo.otpauth = res.otpauth
    setupInfo.qrDataUrl = await QRCode.toDataURL(res.otpauth, { width: 220, margin: 1 })
  }
  catch {
    // 错误提示由 useRequest 统一弹出 (如演示模式禁止绑定), 直接关闭空弹窗
    bindDialog.value = false
  }
  finally {
    setupLoading.value = false
  }
}

/** 开启流程第二步: 动态码确认绑定 */
async function handleConfirmBind() {
  if (!/^\d{6}$/.test(bindCode.value)) {
    ElMessage.warning('请输入 6 位动态验证码')
    return
  }
  totpSubmitting.value = true
  try {
    await api.totpEnable(bindCode.value)
    ElMessage.success('两步验证已开启, 下次登录将要求输入动态码')
    twoFaEnabled.value = true
    bindDialog.value = false
    syncTwoFaStore(true)
  }
  catch {}
  finally {
    totpSubmitting.value = false
  }
}

/** 解绑: 须提供当前有效动态码 */
async function handleConfirmDisable() {
  if (!/^\d{6}$/.test(disableCode.value)) {
    ElMessage.warning('请输入 6 位动态验证码')
    return
  }
  totpSubmitting.value = true
  try {
    await api.totpDisable(disableCode.value)
    ElMessage.success('两步验证已关闭')
    twoFaEnabled.value = false
    disableDialog.value = false
    syncTwoFaStore(false)
  }
  catch {}
  finally {
    totpSubmitting.value = false
  }
}

/** 关闭弹窗时清空已输入的动态码与密钥展示 */
function resetBindDialog() {
  bindCode.value = ''
  setupInfo.secret = ''
  setupInfo.otpauth = ''
  setupInfo.qrDataUrl = ''
}

function resetDisableDialog() {
  disableCode.value = ''
}

/** 复制绑定密钥到剪贴板 (无法扫码时手动输入) */
async function handleCopySecret() {
  if (!setupInfo.secret) return
  try {
    await navigator.clipboard.writeText(setupInfo.secret)
    ElMessage.success('密钥已复制')
  }
  catch {
    ElMessage.warning('复制失败, 请手动选择复制')
  }
}

/** 同步 store 中 userInfo 的两步验证状态 */
function syncTwoFaStore(on: boolean) {
  if (userStore.userInfo) {
    userStore.setUserInfo({ ...userStore.userInfo, twoFaEnabled: on })
  }
}

const profileRules: FormRules = {
  nickname: [{ required: true, message: '请输入昵称', trigger: 'blur' }],
  email: [{ type: 'email', message: '邮箱格式不正确', trigger: 'blur' }],
  phone: [{
    pattern: /^1[3-9]\d{9}$/,
    message: '手机号格式不正确',
    trigger: 'blur',
  }],
}

const pwdRules: FormRules = {
  oldPassword: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, max: 32, message: '长度 6-32', trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    {
      validator: (_r, val, cb) => {
        if (val !== pwdForm.newPassword) cb(new Error('两次密码不一致'))
        else cb()
      },
      trigger: 'blur',
    },
  ],
}

async function loadProfile() {
  loading.value = true
  try {
    const u = await api.profile()
    profile.username = u.username
    profile.nickname = u.nickname
    profile.avatar = u.avatar || ''
    profile.email = u.email || ''
    profile.phone = u.phone || ''
    profile.roles = u.roles || []
    profile.roleNames = u.roleNames || []
    profile.lastLoginAt = u.lastLoginAt || ''
    profile.lastLoginIp = u.lastLoginIp || ''
    twoFaEnabled.value = !!u.twoFaEnabled
  }
  finally {
    loading.value = false
  }
}

async function handleUpdateProfile() {
  if (!profileFormRef.value) return
  const valid = await profileFormRef.value.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    await api.updateProfile({
      nickname: profile.nickname,
      avatar: profile.avatar,
      email: profile.email,
      phone: profile.phone,
    })
    ElMessage.success('个人资料更新成功')
    // 同步 store, 顶部头像/昵称即时刷新
    if (userStore.userInfo) {
      userStore.setUserInfo({
        ...userStore.userInfo,
        nickname: profile.nickname,
        avatar: profile.avatar,
        email: profile.email,
        phone: profile.phone,
      })
    }
  }
  catch (e: any) {
    ElMessage.error(e?.message || '更新失败')
  }
  finally {
    submitting.value = false
  }
}

async function handleChangePassword() {
  if (!pwdFormRef.value) return
  const valid = await pwdFormRef.value.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    await api.changePassword(pwdForm.oldPassword, pwdForm.newPassword)
    ElMessage.success('密码修改成功')
    pwdForm.oldPassword = ''
    pwdForm.newPassword = ''
    pwdForm.confirmPassword = ''
    pwdFormRef.value.resetFields()
    // 强制改密场景: 解锁并回到跳转来源页
    if (userStore.mustChangePwd) {
      userStore.clearMustChangePwd()
      const redirect = route.query.redirect as string
      if (redirect && redirect.startsWith('/') && !redirect.startsWith('//')) {
        await navigateTo(redirect)
      }
    }
  }
  catch (e: any) {
    ElMessage.error(e?.message || '修改失败')
  }
  finally {
    submitting.value = false
  }
}

// 头像上传通道 (FileUploader 的 uploadFn): 前端预校验 + 复用 /auth/avatar 接口
async function uploadAvatar(file: File) {
  const isImage = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'].includes(file.type)
  if (!isImage) {
    ElMessage.warning('仅支持 JPG / PNG / GIF / WEBP 格式')
    throw new Error('仅支持 JPG / PNG / GIF / WEBP 格式')
  }
  if (file.size > 2 * 1024 * 1024) {
    ElMessage.warning('头像大小不能超过 2MB')
    throw new Error('头像大小不能超过 2MB')
  }
  avatarUploading.value = true
  try {
    return await api.uploadAvatar(file)
  }
  finally {
    avatarUploading.value = false
  }
}

// 头像上传成功: 同步本地资料与全局用户信息
function onAvatarUploaded(res: any) {
  profile.avatar = res.url
  if (userStore.userInfo) {
    userStore.setUserInfo({ ...userStore.userInfo, avatar: res.url })
  }
  ElMessage.success('头像更新成功')
}

onMounted(() => {
  loadProfile()
})
</script>

<template>
  <div class="profile-page" v-loading="loading">
    <el-row :gutter="16">
      <el-col :xs="24" :sm="24" :md="8" :lg="7" :xl="6">
        <el-card class="info-card">
          <div class="avatar-wrap">
            <FileUploader
              class="avatar-uploader"
              accept="image/jpeg,image/png,image/gif,image/webp"
              :upload-fn="uploadAvatar"
              @success="onAvatarUploaded"
            >
              <el-avatar :size="96" :src="profile.avatar" class="avatar clickable">
                {{ profile.nickname?.charAt(0) || profile.username?.charAt(0) || 'U' }}
              </el-avatar>
              <div class="avatar-overlay">
                <el-icon :size="20"><UploadFilled /></el-icon>
                <span>更换头像</span>
              </div>
            </FileUploader>
            <el-progress
              v-if="avatarUploading"
              :percentage="100"
              :indeterminate="true"
              :stroke-width="3"
              style="width:96px;margin:4px auto 0"
            />
            <div class="info-name">{{ profile.nickname || profile.username }}</div>
            <div class="info-username">@{{ profile.username }}</div>
            <div class="info-roles">
              <el-tag
                v-for="r in roleTags"
                :key="r"
                :icon="UserFilled"
                size="small"
                effect="plain"
                style="margin-right:4px"
              >{{ r }}</el-tag>
              <el-tag v-if="!roleTags.length" type="info" size="small">无角色</el-tag>
            </div>
          </div>
          <el-divider />
          <ul class="info-list">
            <li>
              <el-icon><User /></el-icon>
              <span class="info-key">账号</span>
              <span class="info-val">{{ profile.username }}</span>
            </li>
            <li>
              <el-icon><Message /></el-icon>
              <span class="info-key">邮箱</span>
              <span class="info-val">{{ profile.email || '-' }}</span>
            </li>
            <li>
              <el-icon><Phone /></el-icon>
              <span class="info-key">手机</span>
              <span class="info-val">{{ profile.phone || '-' }}</span>
            </li>
            <li>
              <el-icon><Clock /></el-icon>
              <span class="info-key">上次登录</span>
              <span class="info-val">{{ profile.lastLoginAt || '首次登录' }}</span>
            </li>
            <li v-if="profile.lastLoginAt">
              <el-icon><Location /></el-icon>
              <span class="info-key">登录 IP</span>
              <span class="info-val">{{ profile.lastLoginIp || '-' }}</span>
            </li>
          </ul>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="24" :md="16" :lg="17" :xl="18">
        <el-card>
          <el-alert
            v-if="userStore.mustChangePwd"
            type="warning"
            :closable="false"
            show-icon
            title="账号密码已被重置或已过期, 请先修改密码后再继续使用"
            style="margin-bottom:12px"
          />
          <el-tabs v-model="activeTab">
            <el-tab-pane name="basic">
              <template #label>
                <span class="tab-label"><el-icon><Edit /></el-icon> 基础资料</span>
              </template>
              <el-form
                ref="profileFormRef"
                :model="profile"
                :rules="profileRules"
                label-width="90px"
                style="max-width:560px"
              >
                <el-form-item label="账号">
                  <el-input :model-value="profile.username" disabled />
                </el-form-item>
                <el-form-item label="昵称" prop="nickname">
                  <el-input v-model="profile.nickname" :prefix-icon="User" placeholder="请输入昵称" />
                </el-form-item>
                <el-form-item label="头像">
                  <div class="form-avatar-row">
                    <FileUploader
                      class="form-avatar-uploader"
                      accept="image/jpeg,image/png,image/gif,image/webp"
                      :upload-fn="uploadAvatar"
                      @success="onAvatarUploaded"
                    >
                      <el-avatar :size="64" :src="profile.avatar" class="clickable">
                        {{ profile.nickname?.charAt(0) || profile.username?.charAt(0) || 'U' }}
                      </el-avatar>
                    </FileUploader>
                    <div class="form-avatar-tip">
                      <el-button size="small" :loading="avatarUploading">
                        <el-icon style="margin-right:4px"><UploadFilled /></el-icon> 上传新头像
                      </el-button>
                      <div class="tip-text">支持 JPG / PNG / GIF / WEBP，不超过 2MB</div>
                    </div>
                  </div>
                </el-form-item>
                <el-form-item label="邮箱" prop="email">
                  <el-input v-model="profile.email" :prefix-icon="Message" placeholder="example@hinay.cn" />
                </el-form-item>
                <el-form-item label="手机" prop="phone">
                  <el-input v-model="profile.phone" :prefix-icon="Phone" placeholder="11位手机号" />
                </el-form-item>
                <el-form-item>
                  <el-button
                    type="primary"
                    :icon="CircleCheck"
                    :loading="submitting"
                    @click="handleUpdateProfile"
                  >保存修改</el-button>
                </el-form-item>
              </el-form>
            </el-tab-pane>
            <el-tab-pane name="password">
              <template #label>
                <span class="tab-label"><el-icon><Key /></el-icon> 修改密码</span>
              </template>
              <el-form
                ref="pwdFormRef"
                :model="pwdForm"
                :rules="pwdRules"
                label-width="100px"
                style="max-width:560px"
              >
                <el-form-item label="原密码" prop="oldPassword">
                  <el-input
                    v-model="pwdForm.oldPassword"
                    type="password"
                    show-password
                    :prefix-icon="Lock"
                  />
                </el-form-item>
                <el-form-item label="新密码" prop="newPassword">
                  <el-input
                    v-model="pwdForm.newPassword"
                    type="password"
                    show-password
                    :prefix-icon="Lock"
                  />
                </el-form-item>
                <el-form-item label="确认新密码" prop="confirmPassword">
                  <el-input
                    v-model="pwdForm.confirmPassword"
                    type="password"
                    show-password
                    :prefix-icon="Lock"
                  />
                </el-form-item>
                <el-form-item>
                  <el-button
                    type="primary"
                    :icon="Key"
                    :loading="submitting"
                    @click="handleChangePassword"
                  >确认修改</el-button>
                </el-form-item>
              </el-form>
            </el-tab-pane>
            <el-tab-pane name="security">
              <template #label>
                <span class="tab-label"><el-icon><Lock /></el-icon> 安全设置</span>
              </template>
              <div class="security-pane">
                <div class="security-item">
                  <div class="security-info">
                    <div class="security-title">
                      两步验证 (TOTP)
                      <el-tag :type="twoFaEnabled ? 'success' : 'info'" size="small">
                        {{ twoFaEnabled ? '已开启' : '未开启' }}
                      </el-tag>
                    </div>
                    <div class="security-desc">
                      开启后登录需在密码之外输入验证器 App (如 Google Authenticator /
                      Microsoft Authenticator / 1Password) 生成的 6 位动态码,
                      即使密码泄露也无法单独登录。
                    </div>
                  </div>
                  <el-button
                    v-if="!twoFaEnabled"
                    type="primary"
                    :icon="Lock"
                    @click="handleStartBind"
                  >开启两步验证</el-button>
                  <el-button
                    v-else
                    type="danger"
                    plain
                    :icon="Lock"
                    @click="disableDialog = true"
                  >解绑两步验证</el-button>
                </div>
              </div>
            </el-tab-pane>
          </el-tabs>
        </el-card>
      </el-col>
    </el-row>

    <!-- 开启两步验证: 扫码 + 动态码确认 -->
    <el-dialog
      v-model="bindDialog"
      title="开启两步验证"
      width="440px"
      :close-on-click-modal="false"
      @closed="resetBindDialog"
    >
      <div v-loading="setupLoading">
        <el-alert
          type="info"
          :closable="false"
          show-icon
          title="用验证器 App 扫描下方二维码; 无法扫码时可手动输入密钥"
          style="margin-bottom:16px"
        />
        <div class="qr-wrap">
          <img v-if="setupInfo.qrDataUrl" :src="setupInfo.qrDataUrl" alt="TOTP 绑定二维码" class="qr-img">
        </div>
        <div v-if="setupInfo.secret" class="secret-row">
          <span class="secret-text">{{ setupInfo.secret }}</span>
          <el-button size="small" text type="primary" @click="handleCopySecret">复制密钥</el-button>
        </div>
        <el-form @submit.prevent>
          <el-form-item label="输入 App 中的 6 位动态码完成绑定" prop="code" :rules="codeRule">
            <el-input
              v-model="bindCode"
              size="large"
              maxlength="6"
              inputmode="numeric"
              autocomplete="one-time-code"
              placeholder="6 位动态验证码"
              class="totp-code-input"
              @keyup.enter="handleConfirmBind"
            />
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="bindDialog = false">取消</el-button>
        <el-button type="primary" :loading="totpSubmitting" @click="handleConfirmBind">
          确认绑定
        </el-button>
      </template>
    </el-dialog>

    <!-- 解绑两步验证 -->
    <el-dialog
      v-model="disableDialog"
      title="解绑两步验证"
      width="420px"
      :close-on-click-modal="false"
      @closed="resetDisableDialog"
    >
      <el-alert
        type="warning"
        :closable="false"
        show-icon
        title="解绑后登录将不再要求动态码, 账号安全等级下降"
        style="margin-bottom:16px"
      />
      <el-form @submit.prevent>
        <el-form-item label="输入当前动态码以确认解绑" prop="code" :rules="codeRule">
          <el-input
            v-model="disableCode"
            size="large"
            maxlength="6"
            inputmode="numeric"
            autocomplete="one-time-code"
            placeholder="6 位动态验证码"
            class="totp-code-input"
            @keyup.enter="handleConfirmDisable"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="disableDialog = false">取消</el-button>
        <el-button type="danger" :loading="totpSubmitting" @click="handleConfirmDisable">
          确认解绑
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.profile-page {
  display: block;
}
.info-card {
  align-self: flex-start;
}
.avatar-wrap {
  text-align: center;
  padding: 12px 0 0;
}
.info-name {
  margin-top: 12px;
  font-size: 18px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.info-username {
  margin-top: 4px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.info-roles {
  margin-top: 10px;
}
.info-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.info-list li {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 0;
  font-size: 14px;
  color: var(--el-text-color-regular);
}
.info-list li .info-key {
  color: var(--el-text-color-secondary);
  width: 56px;
  flex-shrink: 0;
}
.info-list li .info-val {
  color: var(--el-text-color-primary);
  word-break: break-all;
}
.tab-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.avatar-uploader {
  display: inline-block;
  position: relative;
  cursor: pointer;
}
.avatar-uploader :deep(.el-upload) {
  display: inline-block;
  position: relative;
}
.avatar-wrap .avatar.clickable {
  transition: filter 0.2s;
}
.avatar-wrap .avatar-uploader:hover .avatar.clickable {
  filter: brightness(0.7);
}
.avatar-overlay {
  position: absolute;
  top: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 96px;
  height: 96px;
  border-radius: 50%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 12px;
  gap: 2px;
  opacity: 0;
  transition: opacity 0.2s;
  pointer-events: none;
}
.avatar-uploader:hover .avatar-overlay {
  opacity: 1;
}
.form-avatar-row {
  display: flex;
  align-items: center;
  gap: 16px;
}
.form-avatar-uploader :deep(.el-upload) {
  cursor: pointer;
}
.form-avatar-uploader .clickable {
  transition: filter 0.2s;
}
.form-avatar-uploader:hover .clickable {
  filter: brightness(0.8);
}
.form-avatar-tip .tip-text {
  margin-top: 6px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>

<style scoped>
.pwd-policy {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}
.security-pane {
  max-width: 640px;
}
.security-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 0;
}
.security-info {
  flex: 1;
}
.security-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.security-desc {
  margin-top: 6px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}
.qr-wrap {
  display: flex;
  justify-content: center;
  margin-bottom: 12px;
}
.qr-img {
  width: 220px;
  height: 220px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
}
.secret-row {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-bottom: 16px;
}
.secret-text {
  font-family: monospace;
  font-size: 14px;
  letter-spacing: 1px;
  color: var(--el-text-color-primary);
  user-select: all;
}
.totp-code-input :deep(.el-input__inner) {
  letter-spacing: 8px;
  font-size: 20px;
  text-align: center;
}
</style>
