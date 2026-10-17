<template>
  <!-- 页面底部的日志卡片 -->
  <div class="g-card log-card">
    <div class="g-card-head">
      <div class="g-card-title"><ScrollText :size="16" />{{ title ?? t('components.logCard.title') }}</div>
    </div>
    <div class="g-card-body">
      <LogViewer :url="url" :clearable="clearable" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ScrollText } from 'lucide-vue-next'
import LogViewer from './LogViewer.vue'

const { t } = useI18n()

defineProps<{ url: string; clearable?: boolean; title?: string }>()
</script>

<style scoped>
.log-card {
  margin-top: 16px;
}
</style>
