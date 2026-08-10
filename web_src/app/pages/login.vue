<script setup lang="ts">
/**
 * 登录页
 */
import { ref, reactive } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { User, Lock, Histogram, Right } from '@element-plus/icons-vue'
import { useUserStore } from '~/stores/user'
import { useAuthApi } from '~/composables/useApi'

definePageMeta({ layout: 'blank', title: '登录' })

const userStore = useUserStore()
const route = useRoute()
const router = useRouter()
const api = useAuthApi()

const formRef = ref<FormInstance>()
const loading = ref(false)
const form = reactive({
  username: 'admin',
  password: '123456',
})
const rules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function handleSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    const res = await api.login(form.username, form.password)
    userStore.setToken(res.token, res.expireAt)
    userStore.setUserInfo(res.userInfo)
    // 拉取菜单
    const menusRes = await api.menus()
    userStore.setMenus(menusRes.menus, menusRes.permissions)
    ElMessage.success('登录成功')
    const redirect = (route.query.redirect as string) || '/'
    router.replace(redirect)
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
      <div class="tips">默认账号: admin / 123456</div>
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
