<template>
  <el-dialog
    :model-value="modelValue"
    title="用户协议与二次验证"
    width="520px"
    :fullscreen="isMobile"
    class="term-dialog"
    append-to-body
    @update:model-value="(v: boolean) => $emit('update:modelValue', v)"
    @open="load"
  >
    <template v-if="state">
      <div class="block">
        <div class="block-head">
          <span class="block-title">用户协议</span>
          <el-tag :type="state.agreementAccepted ? 'success' : 'warning'" size="small">{{ state.agreementAccepted ? '已确认' : '未确认' }}</el-tag>
        </div>
        <div class="block-desc">首次使用 Web 终端前需要确认用户协议。</div>
        <el-button v-if="!state.agreementAccepted" :loading="busy" @click="acceptAgreement">阅读并确认协议</el-button>
        <el-button v-else text @click="showText = !showText">{{ showText ? '收起协议' : '查看协议' }}</el-button>
        <pre v-if="showText || !state.agreementAccepted" class="agreement">{{ AGREEMENT_TEXT }}</pre>
      </div>

      <div class="block">
        <div class="block-head">
          <span class="block-title">二次验证（TOTP）</span>
          <el-tag :type="state.twoFAEnable ? 'success' : 'info'" size="small">{{ state.twoFAEnable ? '已启用' : '未启用' }}</el-tag>
        </div>
        <div class="block-desc">启用后，新建会话、附着会话与文件管理需要先输入验证器应用中的 6 位验证码，验证通过后 30 分钟内有效。</div>

        <template v-if="!state.twoFAEnable">
          <el-button v-if="!setup" :loading="busy" @click="startSetup">启用二次验证</el-button>
          <div v-else class="setup">
            <div class="step">1. 使用验证器应用（Google Authenticator、Microsoft Authenticator 等）扫描二维码：</div>
            <img v-if="qr" :src="qr" class="qr" alt="二维码" />
            <div class="step">无法扫码时，手动输入密钥：<span class="secret mono">{{ setup.secret }}</span></div>
            <div class="step">2. 输入应用中显示的 6 位验证码完成启用：</div>
            <div class="code-row">
              <el-input v-model="code" maxlength="6" inputmode="numeric" placeholder="6 位验证码" @keyup.enter="enable" />
              <el-button type="primary" :loading="busy" :disabled="code.length < 6" @click="enable">确认启用</el-button>
              <el-button @click="cancelSetup">取消</el-button>
            </div>
          </div>
        </template>
        <div v-else class="code-row">
          <el-input v-model="code" maxlength="6" inputmode="numeric" placeholder="输入验证码以关闭" @keyup.enter="disable" />
          <el-button type="danger" plain :loading="busy" :disabled="code.length < 6" @click="disable">关闭二次验证</el-button>
        </div>
      </div>
    </template>
    <el-skeleton v-else :rows="4" animated />
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import QRCode from 'qrcode'
import { api } from '../../api'
import { useIsMobile } from '../../composables/useIsMobile'
import { AGREEMENT_TEXT } from './security'
import type { SecurityState } from './types'

defineProps<{ modelValue: boolean }>()
defineEmits<{ 'update:modelValue': [v: boolean] }>()

const isMobile = useIsMobile()
const state = ref<SecurityState>()
const setup = ref<{ secret: string; url: string }>()
const qr = ref('')
const code = ref('')
const busy = ref(false)
const showText = ref(false)

async function load() {
  state.value = undefined
  setup.value = undefined
  code.value = ''
  showText.value = false
  try {
    state.value = await api.get<SecurityState>('/api/webterminal/security')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

// run 统一处理忙碌状态与错误提示。
async function run(fn: () => Promise<void>) {
  busy.value = true
  try {
    await fn()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    busy.value = false
  }
}

function acceptAgreement() {
  return run(async () => {
    await api.post('/api/webterminal/security/agreement')
    ElMessage.success('已确认用户协议')
    await load()
  })
}

function startSetup() {
  return run(async () => {
    setup.value = await api.post<{ secret: string; url: string }>('/api/webterminal/security/2fa/setup')
    qr.value = await QRCode.toDataURL(setup.value.url, { width: 200, margin: 1 })
    code.value = ''
  })
}

function cancelSetup() {
  setup.value = undefined
  code.value = ''
}

function enable() {
  if (!setup.value || code.value.length < 6) {
    return
  }
  const secret = setup.value.secret
  return run(async () => {
    await api.post('/api/webterminal/security/2fa/enable', { secret, code: code.value })
    ElMessage.success('二次验证已启用')
    await load()
  })
}

function disable() {
  if (code.value.length < 6) {
    return
  }
  return run(async () => {
    await api.post('/api/webterminal/security/2fa/disable', { code: code.value })
    ElMessage.success('二次验证已关闭')
    await load()
  })
}
</script>

<style scoped>
.block {
  padding: 14px;
  margin-bottom: 14px;
  border: 1px solid var(--g-border);
  border-radius: var(--g-radius);
}
.block-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}
.block-title {
  font-weight: 600;
}
.block-desc {
  margin-bottom: 10px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}
.agreement {
  margin: 10px 0 0;
  padding: 10px;
  max-height: 200px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: var(--g-font);
  font-size: 12px;
  line-height: 1.7;
  background: var(--el-fill-color-light);
  border-radius: var(--g-radius-sm);
}
.setup .step {
  margin: 8px 0;
  font-size: 13px;
  line-height: 1.6;
}
.qr {
  display: block;
  width: 200px;
  height: 200px;
  background: #fff;
  border-radius: 8px;
}
.secret {
  word-break: break-all;
  user-select: all;
  color: var(--el-color-primary);
}
.code-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.code-row .el-input {
  flex: 1 1 160px;
  min-width: 0;
}
</style>
