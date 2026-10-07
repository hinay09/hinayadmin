<script setup lang="ts">
/**
 * 注册页
 * - 受全局配置 sys.allow_register 开关控制: 进入页面先查 /auth/register/status,
 *   未开放时展示关闭提示, 开放时展示注册表单。
 * - 密码与登录同通道: 取一次性 RSA 公钥加密后提交, 明文不出本机。
 * - 注册成功不自动登录, 跳转登录页由用户自行登录 (完整走两步验证等流程)。
 */
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { User, Lock, EditPen, Right, Back } from '@element-plus/icons-vue'
import { JSEncrypt } from 'jsencrypt'
import { storeToRefs } from 'pinia'
import { useConfigStore } from '~/stores/config'
import { useAuthApi } from '~/composables/useApi'

definePageMeta({ layout: 'blank', title: '注册' })

const configStore = useConfigStore()
const { siteName, siteLogo } = storeToRefs(configStore)
const router = useRouter()
const api = useAuthApi()

/** RSA_CODE_KEY_INVALID 后端一次性密钥失效的错误码, 此时静默换新密钥重试一次 */
const RSA_CODE_KEY_INVALID = 50005

/* ---- 注册开关 ---- */
const allowRegister = ref<boolean | null>(null) // null = 查询中

onMounted(async () => {
  configStore.restore()
  try {
    const res = await api.registerStatus()
    allowRegister.value = !!res.allowRegister
  }
  catch {
    // 查询失败按未开放处理 (提交侧后端仍会强校验开关)
    allowRegister.value = false
  }
})

/* ---- 表单 ---- */
const formRef = ref<FormInstance>()
const loading = ref(false)
const form = reactive({
  username: '',
  nickname: '',
  password: '',
  confirmPassword: '',
})

const rules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 32, message: '用户名长度 2-32 个字符', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 32, message: '密码长度 6-32 位', trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入密码', trigger: 'blur' },
    {
      validator: (_rule: any, value: string, callback: (e?: Error) => void) => {
        if (value !== form.password) callback(new Error('两次输入的密码不一致'))
        else callback()
      },
      trigger: 'blur',
    },
  ],
}

/** 取一次性公钥并加密密码, 返回密文与 keyId */
async function encryptPassword(plain: string) {
  const { keyId, publicKey } = await api.publicKey()
  const encryptor = new JSEncrypt()
  encryptor.setPublicKey(publicKey)
  const cipher = encryptor.encrypt(plain)
  if (!cipher) throw new Error('密码加密失败')
  return { keyId, cipher }
}

async function doRegister() {
  const { keyId, cipher } = await encryptPassword(form.password)
  return api.register(form.username, cipher, keyId, form.nickname || undefined)
}

async function handleSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    try {
      await doRegister()
    }
    catch (err: any) {
      // 一次性密钥过期/已用: 换新密钥重试一次, 仍失败则正常抛出
      if (err?.code !== RSA_CODE_KEY_INVALID) throw err
      await doRegister()
    }
    ElMessage.success('注册成功, 请登录')
    router.replace('/login')
  }
  catch {}
  finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="register-page">
    <div class="register-box">
      <!-- 品牌标识: 与登录页同源 (sys.name / sys.logo) -->
      <div class="brand-head">
        <div class="brand-badge">
          <img v-if="siteLogo" :src="siteLogo" class="brand-logo-img" alt="logo">
          <el-icon v-else :size="22"><User /></el-icon>
        </div>
        <span class="brand-name">{{ siteName }}</span>
      </div>

      <!-- 查询开关中 -->
      <div v-if="allowRegister === null" class="register-state" v-loading="true" />

      <!-- 未开放 -->
      <template v-else-if="!allowRegister">
        <h2 class="form-title">注册</h2>
        <el-result icon="info" title="注册功能未开放" sub-title="请联系管理员开通或直接登录">
          <template #extra>
            <el-button type="primary" :icon="Back" @click="router.replace('/login')">返回登录</el-button>
          </template>
        </el-result>
      </template>

      <!-- 开放: 注册表单 -->
      <template v-else>
        <h2 class="form-title">注册账号</h2>
        <p class="form-sub">创建一个新账号, 注册成功后即可登录</p>
        <el-form
          ref="formRef"
          :model="form"
          :rules="rules"
          label-position="top"
          @keyup.enter="handleSubmit"
        >
          <el-form-item label="用户名" prop="username">
            <el-input v-model="form.username" size="large" placeholder="用户名" :prefix-icon="User" maxlength="32" />
          </el-form-item>
          <el-form-item label="昵称" prop="nickname">
            <el-input v-model="form.nickname" size="large" placeholder="昵称 (可选, 默认同用户名)" :prefix-icon="EditPen" maxlength="32" />
          </el-form-item>
          <el-form-item label="密码" prop="password">
            <el-input
              v-model="form.password"
              type="password"
              size="large"
              show-password
              placeholder="密码 (6-32 位)"
              :prefix-icon="Lock"
            />
          </el-form-item>
          <el-form-item label="确认密码" prop="confirmPassword">
            <el-input
              v-model="form.confirmPassword"
              type="password"
              size="large"
              show-password
              placeholder="再次输入密码"
              :prefix-icon="Lock"
            />
          </el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            :icon="Right"
            class="register-btn"
            @click="handleSubmit"
          >
            注册
          </el-button>
          <div class="to-login">
            已有账号? <el-link type="primary" :underline="false" @click="router.replace('/login')">去登录</el-link>
          </div>
        </el-form>
      </template>
    </div>
  </div>
</template>

<style scoped>
.register-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: #f6f8fc;
  padding: 24px;
}

.register-box {
  width: 100%;
  max-width: 420px;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 12px;
  box-shadow: 0 6px 24px rgba(31, 45, 61, 0.06);
  padding: 32px 36px 28px;
}

.brand-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 20px;
}

.brand-badge {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border-radius: 10px;
  background: linear-gradient(135deg, #409eff, #337ecc);
  color: #fff;
  flex-shrink: 0;
}

.brand-logo-img {
  width: 22px;
  height: 22px;
  object-fit: contain;
}

.brand-name {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.form-title {
  margin: 0 0 4px;
  font-size: 22px;
  font-weight: 600;
  color: #303133;
}

.form-sub {
  margin: 0 0 18px;
  font-size: 13px;
  color: #909399;
}

.register-state {
  min-height: 240px;
}

.register-btn {
  width: 100%;
  margin-top: 4px;
}

.to-login {
  margin-top: 14px;
  text-align: center;
  font-size: 13px;
  color: #909399;
}
</style>
