<template>
  <el-dialog
    :model-value="modelValue"
    :title="`快速访问目录 - ${conn?.name ?? ''}`"
    width="480px"
    :fullscreen="isMobile"
    class="term-dialog"
    @update:model-value="(v: boolean) => $emit('update:modelValue', v)"
    @open="text = (conn?.quickAccessDirs ?? []).join('\n')"
  >
    <el-input v-model="text" type="textarea" :rows="8" class="mono" placeholder="每行一个目录，例如&#10;/etc&#10;/var/log" />
    <div class="form-help">这些目录会出现在文件管理（SFTP）面板的「快速访问」菜单中。</div>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="saving" @click="save">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../../api'
import { useIsMobile } from '../../composables/useIsMobile'
import type { Conn } from './types'

const props = defineProps<{ modelValue: boolean; conn: Conn | null }>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean]; saved: [] }>()

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
    ElMessage.success('已保存')
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}
</script>
