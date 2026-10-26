<template>
  <!-- 恢复模块默认配置（frp、cloudflared、Tailscale、File Browser 共用），安装的程序保留 -->
  <el-button type="danger" plain :icon="RotateCcw" :loading="busy" @click="reset">{{ t('components.resetConfig.button') }}</el-button>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { RotateCcw } from 'lucide-vue-next'
import { api } from '../api'

const props = defineProps<{
  // base 为模块接口前缀，例如 /api/frp
  base: string
  // desc 说明会被清除的内容
  desc: string
}>()
const emit = defineEmits<{ done: [] }>()

const { t } = useI18n()
const busy = ref(false)

async function reset() {
  try {
    await ElMessageBox.confirm(`${props.desc} ${t('components.resetConfig.common')}`, t('components.resetConfig.title'), {
      type: 'warning',
      confirmButtonText: t('components.resetConfig.button'),
      confirmButtonClass: 'el-button--danger',
    })
  } catch {
    return
  }
  busy.value = true
  try {
    await api.post(`${props.base}/reset`)
    ElMessage.success(t('components.resetConfig.done'))
    emit('done')
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    busy.value = false
  }
}
</script>
