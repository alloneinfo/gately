<template>
  <div>
    <PageHeader title="网络诊断" desc="Ping、路由追踪、端口探测、DNS 查询、测速与局域网扫描" :icon="Activity">
      <el-button :icon="ShieldBan" @click="openSettings">禁止探测列表</el-button>
    </PageHeader>

    <el-card shadow="never" class="card">
      <el-tabs v-model="tool" @tab-change="onToolChange">
        <el-tab-pane label="Ping" name="ping" />
        <el-tab-pane label="路由追踪" name="traceroute" />
        <el-tab-pane label="端口探测" name="tcp" />
        <el-tab-pane label="DNS 查询" name="dns" />
        <el-tab-pane label="测速" name="speed" />
        <el-tab-pane label="局域网扫描" name="scan" />
      </el-tabs>

      <el-form :model="form" inline class="form" @submit.prevent="run">
        <el-form-item v-if="['ping', 'traceroute', 'tcp', 'dns'].includes(tool)" :label="tool === 'dns' ? '域名 / IP' : '目标'">
          <el-input v-model="form.target" :placeholder="tool === 'dns' ? 'example.com' : '主机名或 IP'" class="w-260" clearable />
        </el-form-item>
        <el-form-item v-if="tool === 'ping'" label="次数"><el-input-number v-model="form.count" :min="1" :max="20" /></el-form-item>
        <el-form-item v-if="tool === 'tcp'" label="端口">
          <el-input v-model="form.ports" placeholder="22,80,443,8000-8100" class="w-260 mono" />
        </el-form-item>
        <template v-if="tool === 'dns'">
          <el-form-item label="类型">
            <el-select v-model="form.type" class="w-110">
              <el-option v-for="t in ['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS', 'PTR']" :key="t" :label="t" :value="t" />
            </el-select>
          </el-form-item>
          <el-form-item label="DNS 服务器">
            <el-input v-model="form.server" placeholder="留空用系统；如 223.5.5.5 或 https://dns.alidns.com/dns-query" class="w-360" clearable />
          </el-form-item>
        </template>
        <template v-if="tool === 'speed'">
          <el-form-item label="下载地址"><el-input v-model="form.url" placeholder="留空使用 Cloudflare 测速节点" class="w-360" clearable /></el-form-item>
          <el-form-item label="时长(秒)"><el-input-number v-model="form.seconds" :min="2" :max="30" /></el-form-item>
        </template>
        <el-form-item v-if="tool === 'scan'" label="网段">
          <el-select v-model="form.cidr" filterable allow-create default-first-option clearable placeholder="默认本机网段" class="w-260">
            <el-option v-for="n in networks" :key="n" :label="n" :value="n" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" native-type="submit" :icon="Play" :loading="starting" :disabled="running">开始</el-button>
          <el-button v-if="running" :icon="Square" @click="cancel">停止</el-button>
        </el-form-item>
      </el-form>
      <div v-if="tool === 'scan'" class="form-help">仅扫描私有网段（最多 1024 个地址）。通过 TCP 探测判断在线，MAC 来自系统邻居表。</div>
    </el-card>

    <el-card v-if="current" shadow="never" class="card">
      <template #header>
        <div class="head">
          <span>{{ current.title }}</span>
          <StatusBadge :type="badge.type" :text="badge.text" />
        </div>
      </template>
      <pre ref="outEl" class="out mono">{{ current.lines.join('\n') }}</pre>
      <div v-if="current.error" class="err">{{ current.error }}</div>

      <el-table v-if="current.tool === 'scan' && hosts.length" :data="hosts" size="small" stripe class="tbl">
        <el-table-column prop="ip" label="IP" width="140" />
        <el-table-column prop="mac" label="MAC" width="160"><template #default="{ row }"><span class="mono">{{ row.mac || '-' }}</span></template></el-table-column>
        <el-table-column prop="hostname" label="主机名" min-width="160" show-overflow-tooltip />
        <el-table-column label="开放端口" min-width="140"><template #default="{ row }">{{ (row.ports ?? []).join(', ') || '-' }}</template></el-table-column>
        <el-table-column label="" width="120">
          <template #default="{ row }">
            <el-button v-if="row.mac" text size="small" :icon="Power" @click="addWol(row)">加入唤醒</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card v-if="history.length" shadow="never" class="card">
      <template #header>最近任务</template>
      <div v-for="t in history" :key="t.id" class="hist" @click="select(t.id)">
        <StatusBadge :type="statusType(t.status)" :text="statusText(t.status)" />
        <span class="t-title">{{ t.title }}</span>
        <span class="meta">{{ new Date(t.started).toLocaleTimeString() }}</span>
      </div>
    </el-card>

    <el-dialog v-model="settingsVisible" title="禁止探测列表" width="480px" :fullscreen="isMobile">
      <p class="form-help">云厂商元数据地址（169.254.169.254 等）始终被禁止。可追加不允许诊断工具访问的 IP 或网段，每行一个。</p>
      <el-input v-model="denyText" type="textarea" :rows="5" class="mono" placeholder="10.0.0.0/8" />
      <template #footer>
        <el-button @click="settingsVisible = false">取消</el-button>
        <el-button type="primary" @click="saveSettings">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Activity, Play, Power, ShieldBan, Square } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import { useFormLayout } from '../composables/useIsMobile'

interface Host {
  ip: string
  mac?: string
  hostname?: string
  ports?: number[]
}
interface Task {
  id: string
  tool: string
  title: string
  status: string
  error?: string
  lines: string[]
  result?: unknown
  started: string
}

const { isMobile } = useFormLayout()
const tool = ref('ping')
const form = reactive({ target: '', count: 4, ports: '22,80,443', type: 'A', server: '', url: '', seconds: 8, cidr: '' })
const networks = ref<string[]>([])
const starting = ref(false)
const current = ref<Task>()
const history = ref<Task[]>([])
const outEl = ref<HTMLElement>()
const settingsVisible = ref(false)
const denyText = ref('')
let timer: number | undefined

const running = computed(() => current.value?.status === 'running')
const hosts = computed(() => (current.value?.tool === 'scan' ? ((current.value.result as Host[] | undefined) ?? []) : []))
const badge = computed(() => ({ type: statusType(current.value?.status ?? ''), text: statusText(current.value?.status ?? '') }))

function statusType(s: string): StatusType {
  return s === 'running' ? 'pending' : s === 'done' ? 'running' : s === 'error' ? 'error' : 'off'
}
function statusText(s: string) {
  return ({ running: '运行中', done: '完成', error: '失败', canceled: '已取消' } as Record<string, string>)[s] ?? s
}

function onToolChange() {
  current.value = undefined
}

async function run() {
  starting.value = true
  try {
    const t = await api.post<Task>('/api/nettools/run', { tool: tool.value, ...form })
    current.value = t
    poll(t.id)
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    starting.value = false
  }
}

function poll(id: string) {
  window.clearInterval(timer)
  const tick = async () => {
    try {
      const t = await api.get<Task>(`/api/nettools/tasks/${id}`)
      current.value = t
      if (t.status !== 'running') {
        window.clearInterval(timer)
        loadHistory()
      }
    } catch {
      window.clearInterval(timer)
    }
  }
  timer = window.setInterval(tick, 700)
}

async function cancel() {
  if (current.value) {
    await api.del(`/api/nettools/tasks/${current.value.id}`)
  }
}

async function loadHistory() {
  history.value = await api.get<Task[]>('/api/nettools/tasks')
}

async function select(id: string) {
  const t = await api.get<Task>(`/api/nettools/tasks/${id}`)
  tool.value = t.tool
  current.value = t
  if (t.status === 'running') {
    poll(id)
  }
}

async function addWol(h: Host) {
  try {
    await api.post('/api/wol/devices', { name: h.hostname || h.ip, macs: [h.mac], broadcastIps: [], port: 9, repeat: 3, checkAddr: '', remark: `局域网扫描添加（${h.ip}）`, enable: true })
    ElMessage.success('已加入网络唤醒设备')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function openSettings() {
  const s = await api.get<{ deny: string[] }>('/api/nettools/settings')
  denyText.value = (s.deny ?? []).join('\n')
  settingsVisible.value = true
}

async function saveSettings() {
  try {
    await api.put('/api/nettools/settings', { deny: denyText.value.split('\n').map((s) => s.trim()).filter(Boolean) })
    settingsVisible.value = false
    ElMessage.success('已保存')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

// 输出更新时滚动到底部
watch(
  () => current.value?.lines.length,
  async () => {
    await nextTick()
    if (outEl.value) {
      outEl.value.scrollTop = outEl.value.scrollHeight
    }
  },
)

onMounted(async () => {
  networks.value = await api.get<string[]>('/api/nettools/networks')
  await loadHistory()
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.card {
  margin-bottom: 14px;
}
.form {
  margin-top: 4px;
}
.w-260 {
  width: 260px;
}
.w-360 {
  width: 360px;
  max-width: 100%;
}
.w-110 {
  width: 110px;
}
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.out {
  margin: 0;
  max-height: 340px;
  overflow: auto;
  padding: 10px 12px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  font-size: 12.5px;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-all;
}
.err {
  margin-top: 8px;
  color: var(--el-color-danger);
  font-size: 13px;
}
.tbl {
  margin-top: 12px;
}
.hist {
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 5px 0;
  cursor: pointer;
}
.t-title {
  flex: 1;
}
.meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
