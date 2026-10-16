<template>
  <!-- 模块全局停用时的提示横幅 -->
  <div class="module-banner">
    <div class="icon"><PowerOff :size="18" /></div>
    <div class="text">
      <div class="title">{{ title }}</div>
      <div class="desc">{{ desc ?? '模块停用期间，下列条目全部不会运行；启用模块后，各条目按自身开关生效。' }}</div>
    </div>
    <el-button type="primary" :icon="Power" :loading="loading" @click="$emit('enable')">{{ action ?? '启用模块' }}</el-button>
  </div>
</template>

<script setup lang="ts">
import { Power, PowerOff } from 'lucide-vue-next'

defineProps<{ title: string; desc?: string; action?: string; loading?: boolean }>()
defineEmits<{ enable: [] }>()
</script>

<style scoped>
.module-banner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  margin-bottom: 16px;
  border-radius: var(--g-radius);
  border: 1px solid var(--el-color-warning-light-5);
  background: var(--el-color-warning-light-9);
}
.icon {
  width: 34px;
  height: 34px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-color-warning);
  background: var(--el-bg-color);
  flex-shrink: 0;
}
.text {
  flex: 1;
  min-width: 0;
}
.title {
  font-weight: 600;
  font-size: 14px;
}
.desc {
  font-size: 12px;
  color: var(--el-text-color-regular);
  margin-top: 2px;
  line-height: 1.5;
}
@media (max-width: 767.98px) {
  .module-banner {
    flex-wrap: wrap;
  }
  .module-banner .el-button {
    width: 100%;
  }
}
</style>
