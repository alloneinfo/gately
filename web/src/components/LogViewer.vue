<template>
  <div class="log-viewer">
    <div class="toolbar">
      <el-select v-model="level" size="small" style="width: 110px">
        <el-option label="全部级别" value="all" />
        <el-option label="信息及以上" value="info" />
        <el-option label="警告及以上" value="warn" />
        <el-option label="仅错误" value="error" />
      </el-select>
      <el-checkbox v-model="follow" size="small">自动滚动</el-checkbox>
      <el-button v-if="clearable" size="small" text :icon="Trash2" class="clear" @click="clear">清空</el-button>
    </div>
    <div ref="box" class="box mono">
      <div v-for="e in visible" :key="e.seq" :class="['line', e.level]">
        <span class="time">{{ formatTime(e.time) }}</span>
        <span class="lvl">{{ e.level.toUpperCase() }}</span>
        <span v-if="e.source" class="src">[{{ e.source }}]</span>
        <span>{{ e.message }}</span>
      </div>
      <div v-if="!visible.length" class="empty">暂无日志</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { Trash2 } from 'lucide-vue-next'
import { api, type LogEntry } from '../api'

const props = defineProps<{ url: string; clearable?: boolean }>()

const entries = ref<LogEntry[]>([])
const level = ref('all')
const follow = ref(true)
const box = ref<HTMLElement>()
let lastSeq = 0
let timer: number | undefined

const order = { debug: 0, info: 1, warn: 2, error: 3 }
const visible = computed(() =>
  level.value !== 'all' ? entries.value.filter((e) => order[e.level] >= order[level.value as keyof typeof order]) : entries.value,
)

function formatTime(t: string) {
  return new Date(t).toLocaleString()
}

// 按序号增量拉取，只追加新日志
async function poll() {
  try {
    const list = await api.get<LogEntry[]>(`${props.url}?since=${lastSeq}`)
    if (list.length) {
      lastSeq = list[list.length - 1].seq
      entries.value = entries.value.concat(list).slice(-2000)
      if (follow.value) {
        await nextTick()
        box.value?.scrollTo({ top: box.value.scrollHeight })
      }
    }
  } catch {
    // 网络抖动时下一轮重试
  }
}

async function clear() {
  await api.del(props.url)
  entries.value = []
}

onMounted(() => {
  poll()
  timer = window.setInterval(poll, 2000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 12px;
  align-items: center;
  margin-bottom: 10px;
}
.clear {
  margin-left: auto;
}
.box {
  height: 300px;
  overflow: auto;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
  border: 1px solid var(--g-border);
  padding: 8px 10px;
  font-size: 12px;
  line-height: 1.6;
}
.line {
  white-space: pre-wrap;
  word-break: break-all;
}
.time {
  color: var(--el-text-color-secondary);
  margin-right: 8px;
}
.lvl {
  display: inline-block;
  width: 48px;
}
.src {
  color: var(--gately-accent);
  margin-right: 6px;
}
.warn .lvl {
  color: var(--el-color-warning);
}
.error .lvl,
.error {
  color: var(--el-color-danger);
}
.debug {
  opacity: 0.7;
}
.empty {
  color: var(--el-text-color-secondary);
}
@media (max-width: 767.98px) {
  .box {
    height: 240px;
    font-size: 11px;
  }
  .time {
    display: block;
    margin-right: 0;
  }
}
</style>
