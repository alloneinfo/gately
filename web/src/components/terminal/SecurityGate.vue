<template>
  <!-- 全局门禁对话框：用户协议确认与二次验证码输入，由 security.ts 的状态驱动 -->
  <el-dialog
    v-model="gate.agreement"
    title="Web 终端用户协议"
    width="560px"
    :fullscreen="isMobile"
    :close-on-click-modal="false"
    class="term-dialog"
    append-to-body
    @close="settleAgreement(false)"
  >
    <pre class="agreement">{{ AGREEMENT_TEXT }}</pre>
    <el-checkbox v-model="agreed" class="agree-check">我已阅读并同意上述内容</el-checkbox>
    <template #footer>
      <el-button @click="settleAgreement(false)">取消</el-button>
      <el-button type="primary" :disabled="!agreed" :loading="busy" @click="accept">同意并继续</el-button>
    </template>
  </el-dialog>

  <el-dialog
    v-model="gate.twofa"
    title="二次验证"
    width="400px"
    :fullscreen="isMobile"
    :close-on-click-modal="false"
    class="term-dialog"
    append-to-body
    @close="settleTwoFA(false)"
    @opened="focusCode"
  >
    <div class="tip">请输入验证器应用（如 Google Authenticator）中显示的 6 位验证码。验证通过后 30 分钟内无需重复验证。</div>
    <el-input
      ref="codeInput"
      v-model="code"
      maxlength="6"
      inputmode="numeric"
      autocomplete="one-time-code"
      placeholder="6 位验证码"
      size="large"
      @keyup.enter="verify"
    />
    <div v-if="err" class="err">{{ err }}</div>
    <template #footer>
      <el-button @click="settleTwoFA(false)">取消</el-button>
      <el-button type="primary" :loading="busy" :disabled="code.length < 6" @click="verify">验证</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { api } from '../../api'
import { useIsMobile } from '../../composables/useIsMobile'
import { AGREEMENT_TEXT, gate, settleAgreement, settleTwoFA } from './security'

const isMobile = useIsMobile()
const agreed = ref(false)
const busy = ref(false)
const code = ref('')
const err = ref('')
const codeInput = ref<{ focus: () => void }>()

watch(
  () => gate.agreement,
  (v) => {
    if (v) {
      agreed.value = false
    }
  },
)
watch(
  () => gate.twofa,
  (v) => {
    if (v) {
      code.value = ''
      err.value = ''
    }
  },
)

function focusCode() {
  nextTick(() => codeInput.value?.focus())
}

async function accept() {
  busy.value = true
  try {
    await api.post('/api/webterminal/security/agreement')
    settleAgreement(true)
  } catch (e) {
    err.value = (e as Error).message
  } finally {
    busy.value = false
  }
}

async function verify() {
  if (code.value.length < 6) {
    return
  }
  busy.value = true
  err.value = ''
  try {
    await api.post('/api/webterminal/security/verify2fa', { code: code.value })
    settleTwoFA(true)
  } catch (e) {
    err.value = (e as Error).message
    code.value = ''
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.agreement {
  margin: 0 0 12px;
  padding: 12px;
  max-height: 46vh;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: var(--g-font);
  font-size: 13px;
  line-height: 1.7;
  background: var(--el-fill-color-light);
  border-radius: var(--g-radius-sm);
}
.agree-check {
  height: 40px;
}
.tip {
  margin-bottom: 12px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}
.err {
  margin-top: 8px;
  font-size: 13px;
  color: var(--el-color-danger);
}
</style>
