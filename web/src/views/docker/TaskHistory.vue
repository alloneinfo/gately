<template>
  <div class="g-card">
    <div class="g-card-head">
      <div class="g-card-title"><ListChecks :size="16" />{{ t('docker.taskHistory.title') }}</div>
      <div class="ops">
        <el-button size="small" :icon="RefreshCw" :loading="loading" @click="load">{{ t('common.refresh') }}</el-button>
        <el-button size="small" :icon="Trash2" @click="clear">{{ t('docker.taskHistory.clearFinished') }}</el-button>
      </div>
    </div>
    <div class="g-card-body">
      <div v-for="task in tasks" :key="task.id" class="t">
        <div class="t-head" @click="toggle(task)">
          <StatusBadge :type="badge(task).type" :text="badge(task).text" />
          <span class="t-title">{{ task.title }}</span>
          <span class="t-time muted">{{ formatTime(task.started) }}</span>
          <ChevronRight :size="16" class="chev" :class="{ open: !!open[task.id] }" />
        </div>
        <div v-if="open[task.id]" class="t-body">
          <div v-if="task.error" class="t-err">{{ task.error }}</div>
          <pre class="t-lines mono">{{ (detail[task.id]?.lines ?? []).join('\n') || t('docker.taskHistory.noOutput') }}</pre>
        </div>
      </div>
      <el-empty v-if="!tasks.length" :description="t('docker.taskHistory.empty')" :image-size="50" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { ChevronRight, ListChecks, RefreshCw, Trash2 } from 'lucide-vue-next'
import { api } from '../../api'
import StatusBadge, { type StatusType } from '../../components/StatusBadge.vue'
import { formatTime } from './format'
import type { DockerTask } from './types'

const { t } = useI18n()

const tasks = ref<DockerTask[]>([])
const loading = ref(false)
const open = reactive<Record<string, boolean>>({})
const detail = reactive<Record<string, DockerTask>>({})

function badge(task: DockerTask): { type: StatusType; text: string } {
  if (task.status === 'running') {
    return { type: 'pending', text: t('docker.taskHistory.running') }
  }
  return task.status === 'done' ? { type: 'ok', text: t('docker.taskHistory.success') } : { type: 'error', text: t('docker.taskHistory.failed') }
}

async function load() {
  loading.value = true
  try {
    tasks.value = (await api.get<DockerTask[]>('/api/docker/tasks')) ?? []
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

async function toggle(task: DockerTask) {
  open[task.id] = !open[task.id]
  if (open[task.id]) {
    try {
      detail[task.id] = await api.get<DockerTask>(`/api/docker/tasks/${task.id}`)
    } catch (e) {
      ElMessage.error((e as Error).message)
    }
  }
}

async function clear() {
  try {
    await api.del('/api/docker/tasks')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

defineExpose({ reload: load })
onMounted(load)
</script>

<style scoped>
.ops {
  display: flex;
  gap: 8px;
}
.t + .t {
  border-top: 1px solid var(--g-border);
}
.t-head {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px 10px;
  min-height: 44px;
  cursor: pointer;
}
.t-title {
  flex: 1;
  min-width: 120px;
  font-size: 13px;
  word-break: break-all;
}
.t-time {
  font-size: 12px;
}
.chev {
  transition: transform 0.15s;
  color: var(--el-text-color-secondary);
}
.chev.open {
  transform: rotate(90deg);
}
.t-body {
  padding-bottom: 10px;
}
.t-err {
  color: var(--el-color-danger);
  font-size: 12px;
  margin-bottom: 6px;
  word-break: break-all;
}
.t-lines {
  margin: 0;
  max-height: 240px;
  overflow: auto;
  padding: 8px;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
  background: var(--el-fill-color-lighter);
  border-radius: 6px;
}
</style>
