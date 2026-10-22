<template>
  <!-- 封禁时长：沿用默认（0）、自定义秒数或永久（-1） -->
  <span class="ban-time">
    <el-select :model-value="mode" class="mode" @update:model-value="setMode">
      <el-option :label="t('components.banTime.default')" value="default" />
      <el-option :label="t('components.banTime.custom')" value="custom" />
      <el-option :label="t('components.banTime.permanent')" value="permanent" />
    </el-select>
    <DurationInput v-if="mode === 'custom'" :model-value="modelValue" @update:model-value="(v: number) => emit('update:modelValue', v)" />
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import DurationInput from './DurationInput.vue'

const props = defineProps<{ modelValue: number }>()
const emit = defineEmits<{ 'update:modelValue': [seconds: number] }>()
const { t } = useI18n()

const mode = computed(() => (props.modelValue === 0 ? 'default' : props.modelValue < 0 ? 'permanent' : 'custom'))

function setMode(v: string) {
  // 切到自定义时先给一天，之后由时长输入框调整
  emit('update:modelValue', v === 'default' ? 0 : v === 'permanent' ? -1 : 86400)
}
</script>

<style scoped>
.ban-time {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}
.mode {
  width: 130px;
}
</style>
