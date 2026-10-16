<template>
  <!-- 后台任务浮条：展示最近的任务进度，结束后可手动关闭 -->
  <div v-if="visible.length" class="task-panel">
    <div v-for="t in visible" :key="t.id" class="task" :class="`is-${t.status}`">
      <div class="task-head">
        <Loader v-if="t.status === 'running'" :size="16" class="spin" />
        <CircleCheck v-else-if="t.status === 'done'" :size="16" class="ok" />
        <CircleX v-else :size="16" class="bad" />
        <span class="task-title">{{ t.title || '后台任务' }}</span>
        <el-button text size="small" class="task-btn" @click="toggle(t.id)">{{ open[t.id] ? '收起' : '详情' }}</el-button>
        <el-button v-if="t.status !== 'running'" text size="small" :icon="X" class="task-btn" aria-label="关闭" @click="t.dismissed = true" />
      </div>
      <div v-if="t.status === 'error' && t.error" class="task-err">{{ t.error }}</div>
      <pre v-else-if="lastLine(t)" class="task-line mono">{{ lastLine(t) }}</pre>
      <pre v-if="open[t.id]" class="task-lines mono">{{ (t.lines ?? []).join('\n') || '(暂无输出)' }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive } from 'vue'
import { CircleCheck, CircleX, Loader, X } from 'lucide-vue-next'
import { trackedTasks, type TrackedTask } from './tasks'

const open = reactive<Record<string, boolean>>({})
const visible = computed(() => trackedTasks.filter((t) => !t.dismissed).slice(0, 3))

function toggle(id: string) {
  open[id] = !open[id]
}

function lastLine(t: TrackedTask): string {
  const l = t.lines ?? []
  return l.length ? l[l.length - 1] : ''
}
</script>

<style scoped>
.task-panel {
  position: fixed;
  right: 16px;
  bottom: 16px;
  z-index: 2100;
  width: 380px;
  max-width: calc(100vw - 32px);
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.task {
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--g-border);
  border-radius: var(--g-radius);
  box-shadow: 0 8px 24px -12px rgba(0, 0, 0, 0.35);
  padding: 8px 8px 8px 12px;
  min-width: 0;
}
.task.is-error {
  border-color: var(--el-color-danger-light-5);
}
.task-head {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 28px;
}
.task-title {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.task-btn {
  flex-shrink: 0;
}
.ok {
  color: var(--el-color-success);
}
.bad {
  color: var(--el-color-danger);
}
.spin {
  color: var(--el-color-primary);
  animation: spin 1s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
.task-line,
.task-lines {
  margin: 4px 4px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  white-space: pre-wrap;
  word-break: break-all;
}
.task-line {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.task-lines {
  max-height: 200px;
  overflow: auto;
  background: var(--el-fill-color-lighter);
  border-radius: 6px;
  padding: 8px;
}
.task-err {
  margin: 4px 4px 0 0;
  font-size: 12px;
  color: var(--el-color-danger);
  word-break: break-all;
}
@media (max-width: 767.98px) {
  .task-panel {
    left: 8px;
    right: 8px;
    width: auto;
    max-width: none;
    bottom: calc(8px + env(safe-area-inset-bottom));
  }
  .task-btn {
    min-height: 36px;
  }
}
</style>
