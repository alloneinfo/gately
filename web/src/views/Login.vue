<template>
  <div class="login">
    <div class="theme-btn">
      <LocaleSwitcher />
      <ThemePicker />
    </div>
    <div class="panel">
      <section class="hero">
        <div class="hero-brand">
          <AppLogo :size="40" />
          <span>Gately</span>
        </div>
        <h1>{{ t('login.heroLine1') }}<br />{{ t('login.heroLine2') }}</h1>
        <ul>
          <li><ArrowLeftRight :size="16" />{{ t('login.feature1') }}</li>
          <li><Server :size="16" />{{ t('login.feature2') }}</li>
          <li><ShieldCheck :size="16" />{{ t('login.feature3') }}</li>
        </ul>
      </section>

      <section class="form-side">
        <div class="mobile-brand">
          <AppLogo :size="44" />
          <div>
            <div class="mb-name">Gately</div>
            <div class="mb-sub">{{ t('login.mobileSub') }}</div>
          </div>
        </div>

        <template v-if="!needCode">
          <h2>{{ t('login.title') }}</h2>
          <p class="sub">{{ t('login.subtitle') }}</p>
          <el-form :model="form" label-position="top" size="large" @submit.prevent="submit">
            <el-form-item :label="t('login.username')">
              <el-input v-model="form.username" autocomplete="username" :prefix-icon="UserRound" />
            </el-form-item>
            <el-form-item :label="t('login.password')">
              <el-input v-model="form.password" type="password" show-password autocomplete="current-password" :prefix-icon="LockKeyhole" />
            </el-form-item>
            <el-button type="primary" native-type="submit" :loading="loading" class="submit">
              {{ t('login.submit') }}
              <ArrowRight :size="16" class="submit-icon" />
            </el-button>
          </el-form>
          <div v-if="providers.length" class="third">
            <div class="third-title"><span>{{ t('login.thirdParty') }}</span></div>
            <a
              v-for="p in providers"
              :key="p.key"
              :href="p.url"
              class="third-btn"
              :class="{ busy: redirecting === p.key, disabled: redirecting && redirecting !== p.key }"
              :aria-busy="redirecting === p.key"
              @click="startThird($event, p)"
            >
              <LoaderCircle v-if="redirecting === p.key" :size="15" class="spin" />
              {{ redirecting === p.key ? t('login.redirecting', { name: p.name }) : t('login.loginWith', { name: p.name }) }}
            </a>
          </div>
          <p class="hint"><Info :size="14" />{{ t('login.initialPasswordHint') }}</p>
        </template>

        <template v-else>
          <div class="step-icon"><ShieldCheck :size="24" /></div>
          <h2>{{ t('login.totpTitle') }}</h2>
          <i18n-t keypath="login.totpPrompt" tag="p" class="sub">
            <template #user><b>{{ form.username }}</b></template>
          </i18n-t>
          <el-form :model="form" label-position="top" size="large" @submit.prevent="submit">
            <el-form-item :label="t('login.code')">
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
            <el-button type="primary" native-type="submit" :loading="loading" class="submit" :disabled="form.code.length < 6">{{ t('login.verify') }}</el-button>
            <el-button class="back" text @click="back"><ArrowLeft :size="14" />{{ t('login.backToAccount') }}</el-button>
          </el-form>
        </template>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage, type InputInstance } from 'element-plus'
import { LoaderCircle, ArrowLeft, ArrowLeftRight, ArrowRight, Info, LockKeyhole, Server, ShieldCheck, UserRound } from 'lucide-vue-next'
import { api, ApiError } from '../api'
import AppLogo from '../components/AppLogo.vue'
import ThemePicker from '../components/ThemePicker.vue'
import LocaleSwitcher from '../components/LocaleSwitcher.vue'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const form = reactive({ username: 'admin', password: '', code: '' })
const loading = ref(false)
const needCode = ref(false)
const codeInput = ref<InputInstance>()
const providers = ref<{ key: string; name: string; type: string; url: string }[]>([])

// 正在跳转的第三方登录方式；非空时按钮显示进度并禁止重复点击
const redirecting = ref('')

function startThird(e: MouseEvent, p: { key: string }) {
  if (redirecting.value) {
    e.preventDefault()
    return
  }
  redirecting.value = p.key
}

// 从授权页点浏览器返回时页面可能来自缓存，需复位按钮状态
window.addEventListener('pageshow', (e) => {
  if (e.persisted) {
    redirecting.value = ''
  }
})

// 第三方登录失败时，后端带着原因跳回登录页
if (typeof route.query.oauthError === 'string') {
  ElMessage.error(route.query.oauthError)
  router.replace({ path: '/login' })
}

onMounted(async () => {
  try {
    const r = await api.get<{ providers: typeof providers.value }>('/api/login/providers')
    providers.value = r.providers ?? []
  } catch {
    // 登录方式获取失败不影响密码登录
  }
})

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
.third {
  margin-top: 18px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.third-title {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--g-text-3, #909399);
  font-size: 12px;
}
.third-title::before,
.third-title::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--g-border);
}
.third-btn {
  display: block;
  text-align: center;
  padding: 9px 12px;
  border: 1px solid var(--g-border);
  border-radius: 8px;
  color: inherit;
  text-decoration: none;
  font-size: 14px;
  transition: background 0.15s;
}
.third-btn.busy {
  opacity: 0.8;
  pointer-events: none;
}
.third-btn.disabled {
  opacity: 0.5;
  pointer-events: none;
}
.third-btn .spin {
  margin-right: 6px;
  vertical-align: -2px;
  animation: third-spin 1s linear infinite;
}
@keyframes third-spin {
  to {
    transform: rotate(360deg);
  }
}
.third-btn:hover {
  background: var(--g-hover, rgba(127, 127, 127, 0.1));
}
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
  display: flex;
  gap: 8px;
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
