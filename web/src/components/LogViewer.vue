<template>
  <div class="log-viewer">
    <div class="toolbar">
      <el-select v-model="level" size="small" style="width: 110px">
        <el-option :label="t('components.logViewer.levelAll')" value="all" />
        <el-option :label="t('components.logViewer.levelInfo')" value="info" />
        <el-option :label="t('components.logViewer.levelWarn')" value="warn" />
        <el-option :label="t('components.logViewer.levelError')" value="error" />
      </el-select>
      <el-checkbox v-model="follow" size="small">{{ t('components.logViewer.autoScroll') }}</el-checkbox>
      <el-button v-if="clearable" size="small" text :icon="Trash2" class="clear" @click="clear">{{ t('common.clear') }}</el-button>
    </div>
    <div ref="box" class="box mono" :style="height ? { height } : undefined">
      <div v-if="hasMore" class="more">
        <el-button size="small" text :loading="loadingMore" @click="loadEarlier">{{ t('components.logViewer.loadEarlier') }}</el-button>
      </div>
      <div v-for="e in visible" :key="e.seq" :class="['line', e.level]">
        <span class="time">{{ e.timeText }}</span>
        <span class="lvl">{{ e.level.toUpperCase() }}</span>
        <span v-if="e.source" class="src">[{{ e.source }}]</span>
        <span>{{ e.message }}</span>
      </div>
      <div v-if="!visible.length" class="empty">{{ t('components.logViewer.empty') }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Trash2 } from 'lucide-vue-next'
import { api, type LogEntry } from '../api'
import { usePolling } from '../composables/usePolling'

const { t } = useI18n()

// height 可覆盖默认高度，例如独立日志页填满可视区域
const props = defineProps<{ url: string; clearable?: boolean; height?: string }>()

// 首次只拉最近 PAGE 条，更早的日志点“加载更早的日志”按页补充；轮询追加后最多保留 MAX 条
const PAGE = 200
const MAX = 2000

// 时间在拉取时格式化一次，轮询刷新列表时不必逐行重复格式化
type Row = LogEntry & { timeText: string }

const entries = ref<Row[]>([])
const level = ref('all')
const follow = ref(true)
const box = ref<HTMLElement>()
// 缓冲中可能还有更早的日志
const hasMore = ref(false)
const loadingMore = ref(false)
let lastSeq = 0
let loaded = false
// 轮询裁剪上限：用户每加载一页历史就放宽相应条数，避免刚加载的旧日志被下一次轮询裁掉
let keep = MAX
// 清空时递增，丢弃清空前发出、清空后才返回的请求结果
let generation = 0

const order = { debug: 0, info: 1, warn: 2, error: 3 }
const visible = computed(() =>
  level.value !== 'all' ? entries.value.filter((e) => order[e.level] >= order[level.value as keyof typeof order]) : entries.value,
)

function toRows(list: LogEntry[]): Row[] {
  return list.map((e) => ({ ...e, timeText: new Date(e.time).toLocaleString() }))
}

async function scrollToEnd() {
  if (follow.value) {
    await nextTick()
    box.value?.scrollTo({ top: box.value.scrollHeight })
  }
}

// 首次拉取最近一页，之后按序号增量拉取，只追加新日志
async function poll() {
  const gen = generation
  try {
    if (!loaded) {
      const list = await api.get<LogEntry[]>(`${props.url}?since=0&limit=${PAGE}`)
      if (gen !== generation) {
        return
      }
      loaded = true
      hasMore.value = list.length >= PAGE
      entries.value = toRows(list)
      if (list.length) {
        lastSeq = list[list.length - 1].seq
      }
      await scrollToEnd()
      return
    }
    const list = await api.get<LogEntry[]>(`${props.url}?since=${lastSeq}`)
    if (gen !== generation) {
      return
    }
    if (list.length) {
      lastSeq = list[list.length - 1].seq
      const next = entries.value.concat(toRows(list))
      if (next.length > keep) {
        // 超出上限时丢掉最旧的部分，仍可通过“加载更早的日志”找回
        hasMore.value = true
      }
      entries.value = next.slice(-keep)
      await scrollToEnd()
    }
  } catch {
    // 网络抖动时下一轮重试
  }
}

// loadEarlier 拉取当前最早一条之前的一页日志，并保持当前可见位置不跳动
async function loadEarlier() {
  const first = entries.value[0]
  if (!first || loadingMore.value) {
    return
  }
  loadingMore.value = true
  const gen = generation
  try {
    const list = await api.get<LogEntry[]>(`${props.url}?before=${first.seq}&limit=${PAGE}`)
    if (gen !== generation) {
      return
    }
    hasMore.value = list.length >= PAGE
    if (!list.length) {
      return
    }
    const el = box.value
    const oldHeight = el?.scrollHeight ?? 0
    // 用户在翻看历史：停止自动滚到底部，并放宽裁剪上限保住刚加载的这一页
    follow.value = false
    keep = Math.max(keep, entries.value.length + list.length)
    entries.value = toRows(list).concat(entries.value)
    await nextTick()
    if (el) {
      el.scrollTop += el.scrollHeight - oldHeight
    }
  } catch {
    // 失败时保留按钮，用户可再次点击
  } finally {
    loadingMore.value = false
  }
}

async function clear() {
  await api.del(props.url)
  generation++
  entries.value = []
  hasMore.value = false
  keep = MAX
}

// 每 2 秒增量拉取，标签页在后台时暂停
const polling = usePolling(poll, 2000)

onMounted(async () => {
  // 首次拉取完成后再开始轮询，避免首屏请求较慢时重复拉取
  await poll()
  polling.start()
})
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
  /* 滚动区域外的行跳过布局与绘制，日志多时首屏与滚动更流畅；auto 让浏览器记住已渲染行的真实高度 */
  content-visibility: auto;
  contain-intrinsic-size: auto 19px;
}
.more {
  text-align: center;
  margin-bottom: 4px;
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
