<template>
  <div class="login">
    <div class="theme-btn">
      <ThemePicker />
    </div>
    <div class="panel">
      <section class="hero">
        <div class="hero-brand">
          <AppLogo :size="40" />
          <span>Gately</span>
        </div>
        <h1>开源、透明的<br />家庭网络网关</h1>
        <ul>
          <li><ArrowLeftRight :size="16" />端口转发、STUN 穿透与动态域名</li>
          <li><Server :size="16" />反向代理、WAF 与证书自动续签</li>
          <li><ShieldCheck :size="16" />不连接项目方服务器，配置明文可审计</li>
        </ul>
      </section>

      <section class="form-side">
        <div class="mobile-brand">
          <AppLogo :size="44" />
          <div>
            <div class="mb-name">Gately</div>
            <div class="mb-sub">开源的家庭网络网关工具集</div>
          </div>
        </div>

        <template v-if="!needCode">
          <h2>登录管理界面</h2>
          <p class="sub">请输入管理员账号与密码</p>
          <el-form :model="form" label-position="top" size="large" @submit.prevent="submit">
            <el-form-item label="用户名">
              <el-input v-model="form.username" autocomplete="username" :prefix-icon="UserRound" />
            </el-form-item>
            <el-form-item label="密码">
              <el-input v-model="form.password" type="password" show-password autocomplete="current-password" :prefix-icon="LockKeyhole" />
            </el-form-item>
            <el-button type="primary" native-type="submit" :loading="loading" class="submit">
              登录
              <ArrowRight :size="16" class="submit-icon" />
            </el-button>
          </el-form>
          <p class="hint"><Info :size="14" />首次启动时初始密码会打印在程序日志中。</p>
        </template>

        <template v-else>
          <div class="step-icon"><ShieldCheck :size="24" /></div>
          <h2>两步验证</h2>
          <p class="sub">打开验证器 App，输入 <b>{{ form.username }}</b> 的 6 位动态验证码</p>
          <el-form :model="form" label-position="top" size="large" @submit.prevent="submit">
            <el-form-item label="验证码">
              <el-input
                ref="codeInput"
                v-model="form.code"
                inputmode="numeric"
                maxlength="6"
                autocomplete="one-time-code"
                placeholder="000000"
                class="code-input mono"
              />
            </el-form-item>
            <el-button type="primary" native-type="submit" :loading="loading" class="submit" :disabled="form.code.length < 6">验证并登录</el-button>
            <el-button class="back" text @click="back"><ArrowLeft :size="14" />返回修改账号</el-button>
          </el-form>
        </template>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, type InputInstance } from 'element-plus'
import { ArrowLeft, ArrowLeftRight, ArrowRight, Info, LockKeyhole, Server, ShieldCheck, UserRound } from 'lucide-vue-next'
import { api, ApiError } from '../api'
import AppLogo from '../components/AppLogo.vue'
import ThemePicker from '../components/ThemePicker.vue'

const router = useRouter()
const route = useRoute()
const form = reactive({ username: 'admin', password: '', code: '' })
const loading = ref(false)
const needCode = ref(false)
const codeInput = ref<InputInstance>()

async function submit() {
  loading.value = true
  try {
    await api.post('/api/login', form)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/overview'
    router.replace(redirect)
  } catch (e) {
    // 密码正确但需要两步验证码时切换到验证码步骤
    if (e instanceof ApiError && e.data?.totpRequired) {
      const first = !needCode.value
      needCode.value = true
      if (first && !form.code) {
        await nextTick()
        codeInput.value?.focus()
        return
      }
    }
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

function back() {
  needCode.value = false
  form.code = ''
}
</script>

<style scoped>
.login {
  min-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background:
    radial-gradient(1200px 600px at 10% -10%, color-mix(in srgb, var(--el-color-primary) 16%, transparent), transparent 60%),
    radial-gradient(900px 500px at 110% 110%, color-mix(in srgb, var(--el-color-primary) 12%, transparent), transparent 60%),
    var(--el-bg-color-page);
  position: relative;
}
.theme-btn {
  position: absolute;
  top: 16px;
  right: 16px;
  z-index: 1;
}
.panel {
  width: 100%;
  max-width: 880px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  background: var(--el-bg-color);
  border: 1px solid var(--g-border);
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 20px 50px -20px color-mix(in srgb, var(--g-hero-3) 45%, transparent);
}
.hero {
  padding: 40px;
  color: #f2fbf8;
  background:
    radial-gradient(circle at 85% 15%, color-mix(in srgb, var(--g-logo-dot) 28%, transparent), transparent 40%),
    linear-gradient(150deg, var(--g-hero-1) 0%, var(--g-hero-2) 60%, var(--g-hero-3) 100%);
  display: flex;
  flex-direction: column;
}
.hero-brand {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 20px;
  font-weight: 700;
  color: #fff;
}
.hero-brand :deep(rect) {
  fill: rgba(255, 255, 255, 0.16);
}
.hero h1 {
  margin: 48px 0 24px;
  font-size: 28px;
  line-height: 1.35;
  color: #fff;
  font-weight: 700;
}
.hero ul {
  list-style: none;
  padding: 0;
  margin: auto 0 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  font-size: 14px;
}
.hero li {
  display: flex;
  align-items: center;
  gap: 10px;
  opacity: 0.92;
}
.form-side {
  padding: 44px 40px;
}
.mobile-brand {
  display: none;
}
h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
}
.sub {
  margin: 6px 0 24px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.submit {
  width: 100%;
  margin-top: 6px;
}
.submit-icon {
  margin-left: 6px;
}
.back {
  width: 100%;
  margin: 8px 0 0 !important;
  gap: 4px;
}
.hint {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 18px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.step-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 16px;
}
.code-input :deep(input) {
  font-size: 22px;
  letter-spacing: 8px;
  text-align: center;
  font-family: var(--g-mono);
}
@media (max-width: 767.98px) {
  .login {
    padding: 16px;
    align-items: flex-start;
    padding-top: 12vh;
  }
  .panel {
    grid-template-columns: 1fr;
    max-width: 420px;
    border-radius: 14px;
  }
  .hero {
    display: none;
  }
  .form-side {
    padding: 28px 22px;
  }
  .mobile-brand {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 24px;
  }
  .mb-name {
    font-size: 20px;
    font-weight: 700;
  }
  .mb-sub {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
  h2 {
    font-size: 19px;
  }
}
</style>
