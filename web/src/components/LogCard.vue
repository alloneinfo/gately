<template>
  <!-- 页面底部的日志卡片 -->
  <div class="g-card log-card">
    <div class="g-card-head">
      <div class="g-card-title"><ScrollText :size="16" />{{ title ?? '运行日志' }}</div>
    </div>
    <div class="g-card-body">
      <LogViewer :url="url" :clearable="clearable" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ScrollText } from 'lucide-vue-next'
import LogViewer from './LogViewer.vue'

defineProps<{ url: string; clearable?: boolean; title?: string }>()
</script>

<style scoped>
.log-card {
  margin-top: 16px;
}
</style>
