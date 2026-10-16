<template>
  <!-- 统一的状态徽标：ok 正常 / running 运行中 / warn 警告 / error 错误 / pending 等待 / off 已停用 / info 普通信息 -->
  <span class="status-badge" :class="`is-${type}`" :title="title">
    <span class="dot" />
    <span class="text"><slot>{{ text }}</slot></span>
  </span>
</template>

<script setup lang="ts">
export type StatusType = 'ok' | 'running' | 'warn' | 'error' | 'pending' | 'off' | 'info'

withDefaults(defineProps<{ type?: StatusType; text?: string; title?: string }>(), { type: 'info', text: '', title: undefined })
</script>

<style scoped>
.status-badge {
  --c: var(--el-color-info);
  --bg: var(--el-fill-color-light);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  height: 22px;
  padding: 0 9px;
  border-radius: 999px;
  font-size: 12px;
  line-height: 22px;
  color: var(--c);
  background: var(--bg);
  white-space: nowrap;
  vertical-align: middle;
  box-sizing: border-box;
}
.text {
  overflow: hidden;
  text-overflow: ellipsis;
}
.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--c);
  flex-shrink: 0;
}
.is-ok,
.is-running {
  --c: var(--el-color-success);
  --bg: var(--el-color-success-light-9);
}
.is-running .dot {
  box-shadow: 0 0 0 0 var(--el-color-success);
  animation: pulse 2s infinite;
}
.is-warn {
  --c: var(--el-color-warning);
  --bg: var(--el-color-warning-light-9);
}
.is-error {
  --c: var(--el-color-danger);
  --bg: var(--el-color-danger-light-9);
}
.is-pending {
  --c: var(--el-color-primary);
  --bg: var(--el-color-primary-light-9);
}
.is-pending .dot {
  background: transparent;
  border: 1.5px solid var(--c);
  width: 5px;
  height: 5px;
}
.is-off {
  --c: var(--el-text-color-secondary);
  --bg: var(--el-fill-color);
}
.is-off .dot {
  background: var(--el-text-color-placeholder);
}
@keyframes pulse {
  0% {
    box-shadow: 0 0 0 0 color-mix(in srgb, var(--el-color-success) 60%, transparent);
  }
  70% {
    box-shadow: 0 0 0 5px transparent;
  }
  100% {
    box-shadow: 0 0 0 0 transparent;
  }
}
</style>
