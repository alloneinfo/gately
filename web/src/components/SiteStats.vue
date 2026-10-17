<template>
  <el-dialog v-model="visible" :title="`站点统计 · ${title}`" width="720px" :fullscreen="isMobile" append-to-body @closed="stop">
    <div class="range">
      <el-radio-group v-model="minutes" size="small" @change="load">
        <el-radio-button :value="60">1 小时</el-radio-button>
        <el-radio-button :value="360">6 小时</el-radio-button>
        <el-radio-button :value="1440">24 小时</el-radio-button>
      </el-radio-group>
      <span class="meta">每分钟一个点，重启或修改该监听后重新计数</span>
    </div>
    <template v-if="data">
      <div class="kpis">
        <div class="kpi"><div class="v">{{ total.toLocaleString() }}</div><div class="l">请求数</div></div>
        <div class="kpi"><div class="v">{{ formatBytes(data.bytesOut) }}</div><div class="l">累计响应流量</div></div>
        <div class="kpi"><div class="v">{{ data.status['4xx'] }}</div><div class="l">4xx</div></div>
        <div class="kpi"><div class="v" :class="{ bad: data.status['5xx'] > 0 }">{{ data.status['5xx'] }}</div><div class="l">5xx</div></div>
      </div>
      <div class="chart-title">请求数</div>
      <Sparkline :series="[{ values: data.series.map((p) => p.requests), color: 'var(--el-color-primary)' }]" :capacity="data.series.length" :height="70" />
      <div class="chart-title">错误（4xx / 5xx）</div>
      <Sparkline
        :series="[
          { values: data.series.map((p) => p.s4xx), color: 'var(--el-color-warning)' },
          { values: data.series.map((p) => p.s5xx), color: 'var(--el-color-danger)' },
        ]"
        :capacity="data.series.length"
        :height="50"
      />
      <div v-if="data.upstreams?.length" class="chart-title">后端</div>
      <div v-for="u in data.upstreams ?? []" :key="u.url" class="up">
        <StatusBadge :type="u.healthy ? 'running' : 'error'" :text="u.healthy ? '健康' : '不可用'" />
        <span class="mono">{{ u.url }}</span>
        <span class="meta">权重 {{ u.weight }}<template v-if="u.backup"> · 备用</template> · 连接 {{ u.conns }}</span>
      </div>
      <div class="tops">
        <div>
          <div class="chart-title">来源 IP Top 10</div>
          <div v-for="t in data.topIps" :key="t.key" class="top"><span class="mono">{{ t.key }}</span><span>{{ t.count }}</span></div>
          <div v-if="!data.topIps.length" class="meta">暂无数据</div>
        </div>
        <div>
          <div class="chart-title">国家 Top 10</div>
          <div v-for="t in data.topCountries" :key="t.key" class="top"><span>{{ t.key }}</span><span>{{ t.count }}</span></div>
          <div v-if="!data.topCountries.length" class="meta">暂无数据（需导入 GeoIP 数据库）</div>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import Sparkline from './Sparkline.vue'
import StatusBadge from './StatusBadge.vue'
import { useFormLayout } from '../composables/useIsMobile'

interface View {
  series: { t: number; requests: number; bytes: number; s4xx: number; s5xx: number }[]
  status: Record<string, number>
  bytesOut: number
  topIps: { key: string; count: number }[]
  topCountries: { key: string; count: number }[]
  upstreams?: { url: string; weight: number; backup: boolean; healthy: boolean; conns: number }[]
}

const { isMobile } = useFormLayout()
const visible = ref(false)
const title = ref('')
const minutes = ref(60)
const data = ref<View>()
let target = { listener: '', site: '' }
let timer: number | undefined

const total = computed(() => (data.value?.series ?? []).reduce((n, p) => n + p.requests, 0))

function formatBytes(n: number) {
  if (n < 1024) {
    return `${n} B`
  }
  if (n < 1024 ** 2) {
    return `${(n / 1024).toFixed(1)} KB`
  }
  if (n < 1024 ** 3) {
    return `${(n / 1024 ** 2).toFixed(1)} MB`
  }
  return `${(n / 1024 ** 3).toFixed(2)} GB`
}

async function load() {
  try {
    data.value = await api.get<View>(`/api/webservice/stats?listener=${encodeURIComponent(target.listener)}&site=${encodeURIComponent(target.site)}&minutes=${minutes.value}`)
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

function stop() {
  window.clearInterval(timer)
}

function open(listener: string, site: string, name: string) {
  target = { listener, site }
  title.value = name
  data.value = undefined
  visible.value = true
  load()
  stop()
  timer = window.setInterval(load, 10000)
}

defineExpose({ open })
</script>

<style scoped>
.range {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.kpis {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
  margin-bottom: 8px;
}
.kpi {
  background: var(--el-fill-color-light);
  border-radius: 8px;
  padding: 10px 12px;
}
.kpi .v {
  font-size: 18px;
  font-weight: 600;
}
.kpi .v.bad {
  color: var(--el-color-danger);
}
.kpi .l {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.chart-title {
  margin: 12px 0 4px;
  font-size: 13px;
  font-weight: 600;
}
.up {
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 3px 0;
}
.tops {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}
.top {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  padding: 2px 0;
}
</style>
