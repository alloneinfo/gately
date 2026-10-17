<template>
  <el-dialog
    :model-value="modelValue"
    :title="t('terminal.quick.title', { name: conn?.name ?? '' })"
    width="480px"
    :fullscreen="isMobile"
    class="term-dialog"
    @update:model-value="(v: boolean) => $emit('update:modelValue', v)"
    @open="text = (conn?.quickAccessDirs ?? []).join('\n')"
  >
    <el-input v-model="text" type="textarea" :rows="8" class="mono" :placeholder="t('terminal.quick.placeholder')" />
    <div class="form-help">{{ t('terminal.quick.help') }}</div>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">{{ t('common.cancel') }}</el-button>
      <el-button type="primary" :loading="saving" @click="save">{{ t('common.save') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { api } from '../../api'
import { useIsMobile } from '../../composables/useIsMobile'
import type { Conn } from './types'

const props = defineProps<{ modelValue: boolean; conn: Conn | null }>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean]; saved: [] }>()

const { t } = useI18n()
const isMobile = useIsMobile()
const text = ref('')
const saving = ref(false)

async function save() {
  if (!props.conn) {
    return
  }
  saving.value = true
  try {
    const dirs = text.value
      .split('\n')
      .map((s) => s.trim())
      .filter((s) => s)
    await api.put(`/api/webterminal/connections/${encodeURIComponent(props.conn.key)}/quickaccess`, { dirs })
    emit('update:modelValue', false)
    emit('saved')
    ElMessage.success(t('common.saved'))
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}
</script>
