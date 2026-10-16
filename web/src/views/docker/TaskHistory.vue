<template>
  <div class="g-card">
    <div class="g-card-head">
      <div class="g-card-title"><ListChecks :size="16" />任务历史</div>
      <div class="ops">
        <el-button size="small" :icon="RefreshCw" :loading="loading" @click="load">刷新</el-button>
        <el-button size="small" :icon="Trash2" @click="clear">清除已结束</el-button>
      </div>
    </div>
    <div class="g-card-body">
      <div v-for="t in tasks" :key="t.id" class="t">
        <div class="t-head" @click="toggle(t)">
          <StatusBadge :type="badge(t).type" :text="badge(t).text" />
          <span class="t-title">{{ t.title }}</span>
          <span class="t-time muted">{{ formatTime(t.started) }}</span>
          <ChevronRight :size="16" class="chev" :class="{ open: !!open[t.id] }" />
        </div>
        <div v-if="open[t.id]" class="t-body">
          <div v-if="t.error" class="t-err">{{ t.error }}</div>
          <pre class="t-lines mono">{{ (detail[t.id]?.lines ?? []).join('\n') || '(无输出)' }}</pre>
        </div>
      </div>
      <el-empty v-if="!tasks.length" description="暂无任务" :image-size="50" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { ChevronRight, ListChecks, RefreshCw, Trash2 } from 'lucide-vue-next'
import { api } from '../../api'
import StatusBadge, { type StatusType } from '../../components/StatusBadge.vue'
import { formatTime } from './format'
import type { DockerTask } from './types'

const tasks = ref<DockerTask[]>([])
const loading = ref(false)
const open = reactive<Record<string, boolean>>({})
const detail = reactive<Record<string, DockerTask>>({})

function badge(t: DockerTask): { type: StatusType; text: string } {
  if (t.status === 'running') {
    return { type: 'pending', text: '进行中' }
  }
  return t.status === 'done' ? { type: 'ok', text: '成功' } : { type: 'error', text: '失败' }
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

async function toggle(t: DockerTask) {
  open[t.id] = !open[t.id]
  if (open[t.id]) {
    try {
      detail[t.id] = await api.get<DockerTask>(`/api/docker/tasks/${t.id}`)
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
