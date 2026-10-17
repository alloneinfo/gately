<template>
  <el-dialog
    :model-value="modelValue"
    :title="t('terminal.settings.title')"
    width="600px"
    :fullscreen="isMobile"
    class="term-dialog"
    @update:model-value="(v: boolean) => $emit('update:modelValue', v)"
    @open="load"
  >
    <el-form v-if="form" label-width="140px" :label-position="labelPosition">
      <el-form-item :label="t('terminal.settings.allowLocalShell')">
        <el-switch v-model="form.allowLocalShell" />
        <span class="help">{{ t('terminal.settings.allowLocalShellHelp') }}</span>
      </el-form-item>
      <el-alert v-if="form.allowLocalShell" type="warning" :closable="false" class="gap">
        {{ t('terminal.settings.localShellWarn') }}
      </el-alert>
      <el-form-item :label="t('terminal.settings.maxSessions')">
        <el-input-number v-model="form.maxSessions" :min="1" :max="500" />
        <span class="help">{{ t('terminal.settings.maxSessionsHelp') }}</span>
      </el-form-item>
      <el-form-item :label="t('terminal.settings.idleTimeout')">
        <el-input-number v-model="form.idleTimeout" :min="0" :max="100000" />
        <span class="help">{{ t('terminal.settings.idleTimeoutHelp') }}</span>
      </el-form-item>
      <el-form-item :label="t('terminal.settings.bufferSize')">
        <el-input-number v-model="form.bufferSize" :min="16" :max="65536" :step="64" />
        <span class="help">{{ t('terminal.settings.bufferSizeHelp') }}</span>
      </el-form-item>
      <el-form-item :label="t('terminal.settings.heartbeat')">
        <el-input-number v-model="form.heartbeatInterval" :min="5" :max="600" />
        <span class="help">{{ t('terminal.settings.heartbeatHelp') }}</span>
      </el-form-item>
      <el-form-item :label="t('terminal.settings.keepAlive')">
        <el-input-number v-model="form.sessionKeepAlive" :min="0" :max="100000" />
        <span class="help">{{ t('terminal.settings.keepAliveHelp') }}</span>
      </el-form-item>
      <el-form-item :label="t('terminal.settings.security')">
        <el-button :icon="ShieldCheck" @click="securityVisible = true">{{ t('terminal.settings.securityBtn') }}</el-button>
      </el-form-item>
      <div v-if="active !== null" class="meta">{{ t('terminal.settings.activeSessions', { count: active }) }}</div>
    </el-form>
    <el-skeleton v-else :rows="5" animated />
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">{{ t('common.cancel') }}</el-button>
      <el-button type="primary" :loading="saving" :disabled="!form" @click="save">{{ t('common.save') }}</el-button>
    </template>
    <SecurityDialog v-model="securityVisible" />
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { ShieldCheck } from 'lucide-vue-next'
import { api } from '../../api'
import { useFormLayout } from '../../composables/useIsMobile'
import SecurityDialog from './SecurityDialog.vue'
import type { TermSettings } from './types'

defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean]; saved: [s: TermSettings] }>()

const { t } = useI18n()
const { isMobile, labelPosition } = useFormLayout()
const form = ref<TermSettings>()
const active = ref<number | null>(null)
const saving = ref(false)
const securityVisible = ref(false)

async function load() {
  form.value = undefined
  try {
    const r = await api.get<{ settings: TermSettings; activeSessions: number }>('/api/webterminal/settings')
    form.value = { ...r.settings }
    active.value = r.activeSessions ?? null
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function save() {
  if (!form.value) {
    return
  }
  saving.value = true
  try {
    await api.put('/api/webterminal/settings', form.value)
    emit('saved', { ...form.value })
    emit('update:modelValue', false)
    ElMessage.success(t('common.saved'))
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.gap {
  margin-bottom: 14px;
}
</style>
