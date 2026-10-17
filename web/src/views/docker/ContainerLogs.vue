<template>
  <div class="logs">
    <div class="bar">
      <el-select v-model="tail" class="b-tail" @change="load">
        <el-option v-for="n in [100, 300, 1000, 5000]" :key="n" :label="t('docker.container.logs.lastLines', { count: n })" :value="n" />
      </el-select>
      <el-input v-model="keyword" :placeholder="t('docker.container.logs.filter')" clearable :prefix-icon="Search" class="b-search" />
      <el-checkbox v-model="timestamps" @change="load">{{ t('docker.container.logs.timestamps') }}</el-checkbox>
      <el-checkbox v-model="auto">{{ t('docker.container.logs.autoRefresh') }}</el-checkbox>
      <div class="b-ops">
        <el-button :icon="RefreshCw" :loading="loading" @click="load">{{ t('common.refresh') }}</el-button>
        <el-button :icon="ArrowDownToLine" @click="toBottom">{{ t('docker.container.logs.toBottom') }}</el-button>
        <el-button :icon="Copy" @click="copy">{{ t('common.copy') }}</el-button>
      </div>
    </div>
    <pre ref="box" class="box mono">{{ shown || t('docker.container.logs.noLogs') }}</pre>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { ArrowDownToLine, Copy, RefreshCw, Search } from 'lucide-vue-next'
import { api } from '../../api'

const { t } = useI18n()

const props = defineProps<{ containerId: string }>()

const tail = ref(300)
const timestamps = ref(true)
const auto = ref(false)
const keyword = ref('')
const logs = ref('')
const loading = ref(false)
const box = ref<HTMLElement>()
let timer: number | undefined

const shown = computed(() => {
  const k = keyword.value.trim().toLowerCase()
  if (!k) {
    return logs.value
  }
  return logs.value
    .split('\n')
    .filter((l) => l.toLowerCase().includes(k))
    .join('\n')
})

async function load() {
  loading.value = true
  try {
    const el = box.value
    // 用户停留在底部时刷新后继续跟随
    const atBottom = !el || el.scrollHeight - el.scrollTop - el.clientHeight < 40
    const r = await api.get<{ logs: string }>(`/api/docker/containers/${props.containerId}/logs?tail=${tail.value}&timestamps=${timestamps.value ? 1 : 0}`)
    logs.value = r.logs ?? ''
    if (atBottom) {
      await nextTick()
      toBottom()
    }
  } catch (e) {
    ElMessage.error((e as Error).message)
    auto.value = false
  } finally {
    loading.value = false
  }
}

function toBottom() {
  box.value?.scrollTo({ top: box.value.scrollHeight })
}

async function copy() {
  try {
    await navigator.clipboard.writeText(shown.value)
    ElMessage.success(t('common.copied'))
  } catch {
    ElMessage.error(t('docker.container.logs.copyFailed'))
  }
}

function stop() {
  window.clearInterval(timer)
  timer = undefined
}

watch(auto, (v) => {
  stop()
  if (v) {
    timer = window.setInterval(load, 3000)
  }
})

onMounted(load)
onBeforeUnmount(stop)
</script>

<style scoped>
.bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  margin-bottom: 10px;
}
.b-tail {
  width: 140px;
}
.b-search {
  width: 180px;
}
.b-ops {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-left: auto;
}
.b-ops .el-button + .el-button {
  margin-left: 0;
}
.box {
  margin: 0;
  height: 56vh;
  min-height: 240px;
  overflow: auto;
  padding: 10px;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--g-border);
  border-radius: var(--g-radius-sm);
}
@media (max-width: 767.98px) {
  .b-tail,
  .b-search {
    flex: 1 1 120px;
    width: auto;
  }
  .b-ops {
    width: 100%;
    margin-left: 0;
  }
  .b-ops .el-button {
    flex: 1;
    min-height: 40px;
  }
  .bar :deep(input) {
    font-size: 16px;
  }
  .box {
    height: calc(100dvh - 340px);
  }
}
</style>
