<template>
  <el-dialog
    :model-value="modelValue"
    title="Web 终端设置"
    width="600px"
    :fullscreen="isMobile"
    class="term-dialog"
    @update:model-value="(v: boolean) => $emit('update:modelValue', v)"
    @open="load"
  >
    <el-form v-if="form" label-width="140px" :label-position="labelPosition">
      <el-form-item label="允许本机 Shell">
        <el-switch v-model="form.allowLocalShell" />
        <span class="help">开启后才能创建和使用「本机 Shell」类型的连接</span>
      </el-form-item>
      <el-alert v-if="form.allowLocalShell" type="warning" :closable="false" class="gap">
        本机 Shell 以 Gately 的运行用户执行命令，等同于服务器的完全控制权。请确保管理界面不暴露在公网，并使用强密码、开启二次验证。
      </el-alert>
      <el-form-item label="最大并发会话">
        <el-input-number v-model="form.maxSessions" :min="1" :max="500" />
        <span class="help">同时存在的终端会话数量上限，包含断线后保留的会话</span>
      </el-form-item>
      <el-form-item label="空闲超时(分钟)">
        <el-input-number v-model="form.idleTimeout" :min="0" :max="100000" />
        <span class="help">会话无输入输出超过该时间自动结束，0 表示不限制</span>
      </el-form-item>
      <el-form-item label="回放缓冲(KB)">
        <el-input-number v-model="form.bufferSize" :min="16" :max="65536" :step="64" />
        <span class="help">每个会话保留的最近输出大小，断线重连或刷新页面后用于恢复屏幕内容</span>
      </el-form-item>
      <el-form-item label="心跳间隔(秒)">
        <el-input-number v-model="form.heartbeatInterval" :min="5" :max="600" />
        <span class="help">浏览器与服务端互发心跳的间隔，用于保持连接并及早发现断线</span>
      </el-form-item>
      <el-form-item label="会话保留(分钟)">
        <el-input-number v-model="form.sessionKeepAlive" :min="0" :max="100000" />
        <span class="help">浏览器断开后会话继续运行的时间，期间可重新附着；0 表示断开即结束会话。要做到刷新页面后找回会话，请设置大于 0 的值</span>
      </el-form-item>
      <el-form-item label="安全">
        <el-button :icon="ShieldCheck" @click="securityVisible = true">用户协议与二次验证</el-button>
      </el-form-item>
      <div v-if="active !== null" class="meta">当前活动会话：{{ active }} 个</div>
    </el-form>
    <el-skeleton v-else :rows="5" animated />
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="saving" :disabled="!form" @click="save">保存</el-button>
    </template>
    <SecurityDialog v-model="securityVisible" />
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { ShieldCheck } from 'lucide-vue-next'
import { api } from '../../api'
import { useFormLayout } from '../../composables/useIsMobile'
import SecurityDialog from './SecurityDialog.vue'
import type { TermSettings } from './types'

defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean]; saved: [s: TermSettings] }>()

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
    ElMessage.success('已保存')
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
