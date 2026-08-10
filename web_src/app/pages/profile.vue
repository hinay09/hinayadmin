<script setup lang="ts">
/**
 * 个人中心
 * - 左侧: 个人信息卡片(头像/昵称/账号/角色/邮箱/手机)
 * - 右侧: tabs - "基础资料" + "修改密码"
 * 数据接口: GET/PUT /auth/profile, PUT /auth/password
 */
import { reactive, ref, onMounted } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
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
} from '@element-plus/icons-vue'
import { useAuthApi } from '~/composables/useApi'
import { useUserStore } from '~/stores/user'

definePageMeta({ title: '个人中心', layout: 'default' })

const api = useAuthApi()

/**
 * 角色编码 → 中文名映射。
 * 管理员在后台角色管理中新增角色时, 请同步更新此映射。
 */
const ROLE_NAME_MAP: Record<string, string> = {
  admin:  '超级管理员',
  common: '普通用户',
}

const roleName = (code: string) => ROLE_NAME_MAP[code] || code
const userStore = useUserStore()

const activeTab = ref<'basic' | 'password'>('basic')
const loading = ref(false)
const submitting = ref(false)
const avatarUploading = ref(false)

interface ProfileForm {
  nickname: string
  avatar: string
  email: string
  phone: string
}
const profile = reactive<ProfileForm & { username: string; roles: string[] }>({
  username: '',
  nickname: '',
  avatar: '',
  email: '',
  phone: '',
  roles: [],
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
  }
  catch (e: any) {
    ElMessage.error(e?.message || '修改失败')
  }
  finally {
    submitting.value = false
  }
}

async function handleAvatarChange(uploadFile: any) {
  const file = uploadFile.raw || uploadFile.file || uploadFile
  if (!file || !(file instanceof File)) return
  const isImage = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'].includes(file.type)
  if (!isImage) {
    ElMessage.warning('仅支持 JPG / PNG / GIF / WEBP 格式')
    return
  }
  if (file.size > 2 * 1024 * 1024) {
    ElMessage.warning('头像大小不能超过 2MB')
    return
  }
  avatarUploading.value = true
  try {
    const res = await api.uploadAvatar(file)
    profile.avatar = res.url
    if (userStore.userInfo) {
      userStore.setUserInfo({ ...userStore.userInfo, avatar: res.url })
    }
    ElMessage.success('头像更新成功')
  } catch (e: any) {
    ElMessage.error(e?.message || '头像上传失败')
  } finally {
    avatarUploading.value = false
  }
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
            <el-upload
              class="avatar-uploader"
              :show-file-list="false"
              :before-upload="() => false"
              :on-change="handleAvatarChange"
              accept="image/jpeg,image/png,image/gif,image/webp"
            >
              <el-avatar :size="96" :src="profile.avatar" class="avatar clickable">
                {{ profile.nickname?.charAt(0) || profile.username?.charAt(0) || 'U' }}
              </el-avatar>
              <div class="avatar-overlay">
                <el-icon :size="20"><UploadFilled /></el-icon>
                <span>更换头像</span>
              </div>
            </el-upload>
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
                v-for="r in profile.roles"
                :key="r"
                :icon="UserFilled"
                size="small"
                effect="plain"
                style="margin-right:4px"
              >{{ roleName(r) }}</el-tag>
              <el-tag v-if="!profile.roles.length" type="info" size="small">无角色</el-tag>
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
          </ul>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="24" :md="16" :lg="17" :xl="18">
        <el-card>
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
                    <el-upload
                      class="form-avatar-uploader"
                      :show-file-list="false"
                      :before-upload="() => false"
                      :on-change="handleAvatarChange"
                      accept="image/jpeg,image/png,image/gif,image/webp"
                    >
                      <el-avatar :size="64" :src="profile.avatar" class="clickable">
                        {{ profile.nickname?.charAt(0) || profile.username?.charAt(0) || 'U' }}
                      </el-avatar>
                    </el-upload>
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
          </el-tabs>
        </el-card>
      </el-col>
    </el-row>
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
  color: #303133;
}
.info-username {
  margin-top: 4px;
  font-size: 13px;
  color: #909399;
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
  color: #606266;
}
.info-list li .info-key {
  color: #909399;
  width: 56px;
  flex-shrink: 0;
}
.info-list li .info-val {
  color: #303133;
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
  color: #909399;
}
</style>
