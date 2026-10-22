<template>
  <!-- 以秒为单位的时长输入：数字加单位（分钟 / 小时 / 天），显示时自动选取能整除的最大单位 -->
  <span class="duration-input">
    <el-input-number v-model="amount" :min="1" :max="maxAmount" :size="size" controls-position="right" class="num" />
    <el-select v-model="unit" :size="size" class="unit">
      <el-option :label="t('components.duration.minutes')" :value="60" />
      <el-option :label="t('components.duration.hours')" :value="3600" />
      <el-option :label="t('components.duration.days')" :value="86400" />
    </el-select>
  </span>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(defineProps<{ modelValue: number; max?: number; size?: 'small' | 'default' }>(), { max: 365 * 86400, size: 'default' })
const emit = defineEmits<{ 'update:modelValue': [seconds: number] }>()
const { t } = useI18n()

const unit = ref(60)
const amount = ref(1)

// 按外部值选取单位：能被天整除用天，其次小时，否则分钟
function sync(sec: number) {
  if (!sec || sec <= 0) {
    unit.value = 3600
    amount.value = 1
    return
  }
  for (const u of [86400, 3600, 60]) {
    if (sec % u === 0) {
      unit.value = u
      amount.value = sec / u
      return
    }
  }
  unit.value = 60
  amount.value = Math.max(1, Math.round(sec / 60))
}

watch(
  () => props.modelValue,
  (v) => {
    if (v !== amount.value * unit.value) {
      sync(v)
    }
  },
  { immediate: true },
)

const maxAmount = computed(() => Math.max(1, Math.floor(props.max / unit.value)))

watch([amount, unit], () => {
  const a = Math.min(amount.value || 1, maxAmount.value)
  if (a !== amount.value) {
    amount.value = a
  }
  const sec = a * unit.value
  if (sec !== props.modelValue) {
    emit('update:modelValue', sec)
  }
})
</script>

<style scoped>
.duration-input {
  display: inline-flex;
  gap: 6px;
  align-items: center;
}
.num {
  width: 120px;
}
.unit {
  width: 90px;
}
</style>
