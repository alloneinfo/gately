<template>
  <!-- 卡片中的一个“标签 + 值”字段 -->
  <div class="field" :class="{ wide }">
    <div class="field-label">{{ label }}</div>
    <div class="field-value" :class="{ mono }"><slot /></div>
  </div>
</template>

<script setup lang="ts">
// wide 为 true 时占满整行
defineProps<{ label: string; mono?: boolean; wide?: boolean }>()
</script>

<style scoped>
.field {
  min-width: 0;
}
.field.wide {
  grid-column: 1 / -1;
}
.field-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 3px;
  line-height: 1.4;
}
.field-value {
  font-size: 13px;
  line-height: 1.6;
  color: var(--el-text-color-primary);
  word-break: break-all;
}
</style>
