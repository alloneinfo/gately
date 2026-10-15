<template>
  <div class="login">
    <el-card class="login-card" shadow="never">
      <div class="brand">
        <span class="logo">G</span>
        <div>
          <h1>Gately</h1>
          <p>开源的家庭网络网关工具集</p>
        </div>
      </div>
      <el-form :model="form" @submit.prevent="submit" label-position="top">
        <el-form-item label="用户名">
          <el-input v-model="form.username" autocomplete="username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" show-password autocomplete="current-password" />
        </el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" class="submit">登录</el-button>
      </el-form>
      <p class="hint">首次启动时初始密码会打印在程序日志中。</p>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { api } from '../api'

const router = useRouter()
const route = useRoute()
const form = reactive({ username: 'admin', password: '' })
const loading = ref(false)

async function submit() {
  loading.value = true
  try {
    await api.post('/api/login', form)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/overview'
    router.replace(redirect)
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login {
  min-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background:
    radial-gradient(circle at 20% 20%, rgba(14, 124, 102, 0.12), transparent 40%),
    radial-gradient(circle at 80% 80%, rgba(14, 124, 102, 0.08), transparent 40%);
}
.login-card {
  width: 360px;
}
.brand {
  display: flex;
  gap: 12px;
  align-items: center;
  margin-bottom: 20px;
}
.logo {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  background: var(--gately-accent);
  color: #fff;
  font-weight: 700;
  font-size: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
}
h1 {
  margin: 0;
  font-size: 22px;
}
.brand p {
  margin: 2px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.submit {
  width: 100%;
}
.hint {
  margin: 14px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
