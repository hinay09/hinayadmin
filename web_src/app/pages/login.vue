<script setup lang="ts">
/**
 * 登录页
 */
import { ref, reactive } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { User, Lock, Histogram, Right } from '@element-plus/icons-vue'
import { JSEncrypt } from 'jsencrypt'
import { useUserStore, safeRedirect } from '~/stores/user'
import { useAuthApi } from '~/composables/useApi'

definePageMeta({ layout: 'blank', title: '登录' })

const userStore = useUserStore()
const route = useRoute()
const router = useRouter()
const api = useAuthApi()

const formRef = ref<FormInstance>()
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
    userStore.setToken(res.token, res.expireAt)
    userStore.setUserInfo(res.userInfo)
    // 拉取菜单
    const menusRes = await api.menus()
    userStore.setMenus(menusRes.menus, menusRes.permissions)
    ElMessage.success('登录成功')
    // 开放跳转防护: 仅允许站内路径
    router.replace(safeRedirect(route.query.redirect as string))
  }
  catch {}
  finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-title">
        <el-icon class="login-logo"><Histogram /></el-icon>
        <span>Hinay Admin</span>
      </div>
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
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
  display: flex;
  align-items: center;
  justify-content: center;
}
.login-card {
  width: 380px;
  background: #fff;
  padding: 36px 32px 28px;
  border-radius: 8px;
  box-shadow: 0 8px 32px rgba(0,0,0,0.12);
}
.login-title {
  font-size: 22px;
  font-weight: 600;
  color: #303133;
  text-align: center;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}
.login-logo {
  font-size: 26px;
  color: #1890ff;
}
.login-sub {
  font-size: 13px;
  color: #909399;
  text-align: center;
  margin: 6px 0 24px;
}
.login-btn {
  width: 100%;
}
.tips {
  margin-top: 16px;
  font-size: 12px;
  color: #909399;
  text-align: center;
}
</style>
