<template>
  <div class="page-head">
    <div class="page-head-main">
      <div v-if="icon" class="page-head-icon">
        <component :is="icon" :size="20" :stroke-width="2" />
      </div>
      <div class="page-head-text">
        <h2>{{ title }}</h2>
        <p v-if="desc">{{ desc }}</p>
      </div>
    </div>
    <div v-if="$slots.default" class="page-head-actions">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'

defineProps<{ title: string; desc?: string; icon?: Component }>()
</script>

<style scoped>
.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px 16px;
  flex-wrap: wrap;
  margin-bottom: 16px;
}
.page-head-main {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.page-head-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  flex-shrink: 0;
}
.page-head-text {
  min-width: 0;
}
h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  line-height: 1.3;
}
p {
  margin: 2px 0 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}
.page-head-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.page-head-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}
@media (max-width: 767.98px) {
  .page-head {
    align-items: flex-start;
    flex-direction: column;
  }
  .page-head-icon {
    width: 36px;
    height: 36px;
  }
  .page-head-actions {
    width: 100%;
  }
}
</style>
