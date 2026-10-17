<template>
  <el-config-provider :locale="elLocale">
    <router-view />
  </el-config-provider>
</template>

<script setup lang="ts">
import { shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Language } from 'element-plus/es/locale'
import { loadElementLocale } from './i18n/elementLocale'

const { locale } = useI18n()
const elLocale = shallowRef<Language | undefined>()

// 界面语言变化时同步组件库自带文案
watch(
  locale,
  async (code) => {
    elLocale.value = await loadElementLocale(code)
  },
  { immediate: true },
)
</script>
