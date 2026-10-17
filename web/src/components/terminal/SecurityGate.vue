<template>
  <!-- 全局门禁对话框：用户协议确认与二次验证码输入，由 security.ts 的状态驱动 -->
  <el-dialog
    v-model="gate.agreement"
    :title="t('terminal.security.gateAgreementTitle')"
    width="560px"
    :fullscreen="isMobile"
    :close-on-click-modal="false"
    class="term-dialog"
    append-to-body
    @close="settleAgreement(false)"
  >
    <pre class="agreement">{{ agreementText() }}</pre>
    <el-checkbox v-model="agreed" class="agree-check">{{ t('terminal.security.agreeCheck') }}</el-checkbox>
    <template #footer>
      <el-button @click="settleAgreement(false)">{{ t('common.cancel') }}</el-button>
      <el-button type="primary" :disabled="!agreed" :loading="busy" @click="accept">{{ t('terminal.security.agreeContinue') }}</el-button>
    </template>
  </el-dialog>

  <el-dialog
    v-model="gate.twofa"
    :title="t('terminal.security.gateTwofaTitle')"
    width="400px"
    :fullscreen="isMobile"
    :close-on-click-modal="false"
    class="term-dialog"
    append-to-body
    @close="settleTwoFA(false)"
    @opened="focusCode"
  >
    <div class="tip">{{ t('terminal.security.gateTip') }}</div>
    <el-input
      ref="codeInput"
      v-model="code"
      maxlength="6"
      inputmode="numeric"
      autocomplete="one-time-code"
      :placeholder="t('terminal.security.codePlaceholder')"
      size="large"
      @keyup.enter="verify"
    />
    <div v-if="err" class="err">{{ err }}</div>
    <template #footer>
      <el-button @click="settleTwoFA(false)">{{ t('common.cancel') }}</el-button>
      <el-button type="primary" :loading="busy" :disabled="code.length < 6" @click="verify">{{ t('terminal.security.verify') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../api'
import { useIsMobile } from '../../composables/useIsMobile'
import { agreementText, gate, settleAgreement, settleTwoFA } from './security'

const { t } = useI18n()
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
