<template>
  <el-input v-bind="$attrs" :model-value="modelValue" :type="shown ? 'text' : 'password'" @update:model-value="onInput">
    <template #suffix>
      <span
        class="secret-eye"
        role="button"
        tabindex="0"
        :title="shown ? t('components.secretInput.hide') : t('components.secretInput.show')"
        @click="toggle"
        @keydown.enter.prevent="toggle"
      >
        <Loader2 v-if="loading" :size="14" class="spin" />
        <EyeOff v-else-if="shown" :size="14" />
        <Eye v-else :size="14" />
      </span>
    </template>
  </el-input>
</template>

<script setup lang="ts">
// SecretInput 是密码 / 令牌输入框。后端返回的已保存密钥是脱敏占位符，点击“显示”时
// 通过 reveal 向后端取回明文填入输入框再显示；取回的明文原样提交与提交占位符等价，后端都保留原值。
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { Eye, EyeOff, Loader2 } from 'lucide-vue-next'
import { SECRET_MASK } from '../secrets'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  modelValue?: string | null
  // reveal 返回已保存密钥的明文；不提供时只切换显示，适用于尚未保存过的新密钥
  reveal?: () => Promise<string | undefined>
}>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const { t } = useI18n()
const shown = ref(false)
const loading = ref(false)

function onInput(v: string) {
  emit('update:modelValue', v)
}

async function toggle() {
  if (loading.value) {
    return
  }
  if (shown.value) {
    shown.value = false
    return
  }
  if (props.reveal && props.modelValue === SECRET_MASK) {
    loading.value = true
    try {
      const v = await props.reveal()
      if (v !== undefined) {
        emit('update:modelValue', v)
      }
    } catch (e) {
      ElMessage.error((e as Error).message)
      return
    } finally {
      loading.value = false
    }
  }
  shown.value = true
}
</script>

<style scoped>
.secret-eye {
  display: inline-flex;
  align-items: center;
  cursor: pointer;
  color: var(--el-input-icon-color, var(--el-text-color-placeholder));
}
.secret-eye:hover {
  color: var(--el-text-color-regular);
}
.spin {
  animation: secret-spin 1s linear infinite;
}
@keyframes secret-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
