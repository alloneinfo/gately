<template>
  <el-dialog
    :model-value="modelValue"
    title="会话列表"
    width="760px"
    :fullscreen="isMobile"
    class="term-dialog"
    @update:model-value="(v: boolean) => $emit('update:modelValue', v)"
    @open="onOpen"
    @closed="stopTimer"
  >
    <div class="top">
      <span class="meta">服务端保留的终端会话，共 {{ list.length }} 个。刷新页面或断线后可在此附着回来。</span>
      <el-button size="small" :icon="RefreshCw" :loading="loading" @click="load">刷新</el-button>
    </div>
    <el-empty v-if="!list.length && !loading" description="暂无会话" :image-size="60" />
    <div v-for="s in list" :key="s.id" class="item">
      <div class="info">
        <div class="line1">
          <span class="name ellipsis">{{ s.connName || s.connKey }}</span>
          <el-tag size="small" effect="plain">{{ typeLabel[s.type] ?? s.type }}</el-tag>
          <el-tag v-if="openIds.includes(s.id)" size="small" type="success">已打开</el-tag>
          <el-tag v-else-if="s.attached > 0" size="small" type="warning">{{ s.attached }} 端附着</el-tag>
          <el-tag v-else size="small" type="info">已脱离</el-tag>
        </div>
        <div v-if="s.remark" class="remark">备注：{{ s.remark }}</div>
        <div class="meta">
          创建 {{ fmtTime(s.createdAt) }} · 最近活动 {{ fmtTime(s.lastActive) }} · {{ s.cols }}×{{ s.rows }} · 输出 {{ formatBytes(s.bytesOut) }} / 输入 {{ formatBytes(s.bytesIn) }}
        </div>
        <div v-if="stats[s.id]" class="stats meta">
          已运行 {{ formatDuration(stats[s.id].durationSec) }} · 空闲 {{ formatDuration(stats[s.id].idleSec) }} · 当前附着 {{ stats[s.id].attached }} 端
        </div>
      </div>
      <div class="ops">
        <el-button size="small" type="primary" plain :icon="Link2" @click="$emit('attach', s)">{{ openIds.includes(s.id) ? '切换' : '附着' }}</el-button>
        <el-button size="small" :icon="StickyNote" @click="editRemark(s)">备注</el-button>
        <el-button size="small" :icon="BarChart3" @click="showStats(s)">统计</el-button>
        <el-button size="small" type="danger" plain :icon="Power" @click="kill(s)">结束</el-button>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { BarChart3, Link2, Power, RefreshCw, StickyNote } from 'lucide-vue-next'
import { api, formatBytes, formatDuration } from '../../api'
import { useIsMobile } from '../../composables/useIsMobile'
import { fmtTime, typeLabel, type SessionInfo, type SessionStats } from './types'

defineProps<{ modelValue: boolean; openIds: string[] }>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean]; attach: [s: SessionInfo]; killed: [id: string]; remark: [id: string, remark: string] }>()

const isMobile = useIsMobile()
const list = ref<SessionInfo[]>([])
const stats = ref<Record<string, SessionStats>>({})
const loading = ref(false)
let timer: ReturnType<typeof setInterval> | undefined

async function load() {
  loading.value = true
  try {
    list.value = (await api.get<SessionInfo[] | null>('/api/webterminal/sessions')) ?? []
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

function stopTimer() {
  if (timer) {
    clearInterval(timer)
    timer = undefined
  }
}

// 打开期间每 5 秒自动刷新一次
function onOpen() {
  stats.value = {}
  load()
  stopTimer()
  timer = setInterval(load, 5000)
}

async function showStats(s: SessionInfo) {
  try {
    stats.value[s.id] = await api.get<SessionStats>(`/api/webterminal/sessions/${encodeURIComponent(s.id)}/stats`)
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function editRemark(s: SessionInfo) {
  try {
    const { value } = await ElMessageBox.prompt('会话备注', '编辑备注', { inputValue: s.remark, inputPlaceholder: '例如：部署任务' })
    await api.put(`/api/webterminal/sessions/${encodeURIComponent(s.id)}/remark`, { remark: value ?? '' })
    s.remark = value ?? ''
    emit('remark', s.id, s.remark)
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error((e as Error).message)
    }
  }
}

async function kill(s: SessionInfo) {
  try {
    await ElMessageBox.confirm(`结束会话「${s.connName}」？会话中正在运行的程序会被终止。`, '确认', { type: 'warning' })
    await api.del(`/api/webterminal/sessions/${encodeURIComponent(s.id)}`)
    emit('killed', s.id)
    await load()
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error((e as Error).message)
    }
  }
}

onBeforeUnmount(stopTimer)
</script>

<style scoped>
.top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}
.item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 0;
  border-top: 1px solid var(--g-border);
}
.info {
  flex: 1;
  min-width: 0;
}
.line1 {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.name {
  max-width: 220px;
  font-weight: 500;
}
.remark {
  margin-top: 2px;
  font-size: 13px;
  color: var(--el-text-color-regular);
}
.stats {
  margin-top: 2px;
  color: var(--el-color-primary);
}
.ops {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px;
  max-width: 250px;
}
.ops .el-button + .el-button {
  margin-left: 0;
}
@media (max-width: 767.98px) {
  .item {
    flex-direction: column;
    align-items: stretch;
  }
  .ops {
    max-width: none;
    justify-content: flex-start;
  }
}
</style>
