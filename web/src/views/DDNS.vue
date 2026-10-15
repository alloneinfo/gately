<template>
  <div>
    <div class="page-header">
      <h2>动态域名</h2>
      <div>
        <el-switch v-model="settings.enable" active-text="启用模块" @change="saveSettings" class="module-switch" />
        <el-button type="primary" @click="openEditor()">添加任务</el-button>
      </div>
    </div>

    <el-empty v-if="!loading && !tasks.length" description="暂无任务，可添加或在设置中导入 Lucky 配置" />

    <el-card v-for="t in tasks" :key="t.key" shadow="never" class="task">
      <template #header>
        <div class="task-head">
          <div class="task-title">
            <el-switch :model-value="t.enable" size="small" @change="(v: string | number | boolean) => toggle(t, Boolean(v))" />
            <strong>{{ t.name || '未命名任务' }}</strong>
            <el-tag size="small" effect="plain">{{ providerTitle(t.provider) }}</el-tag>
            <el-tag v-if="t.state.ipv4" size="small" type="info" class="mono">v4 {{ t.state.ipv4 }}</el-tag>
            <el-tag v-if="t.state.ipv6" size="small" type="info" class="mono">v6 {{ t.state.ipv6 }}</el-tag>
          </div>
          <div>
            <span class="meta" v-if="t.state.lastRun">上次 {{ fmtTime(t.state.lastRun) }}</span>
            <span class="meta" v-if="t.state.nextRun">下次 {{ fmtTime(t.state.nextRun) }}</span>
            <el-button size="small" :disabled="!t.state.running" @click="sync(t, false)">检查</el-button>
            <el-button size="small" :disabled="!t.state.running" @click="sync(t, true)">强制同步</el-button>
            <el-button size="small" @click="openEditor(t)">编辑</el-button>
            <el-popconfirm title="确定删除该任务？不会删除服务商处的解析记录" @confirm="remove(t)">
              <template #reference>
                <el-button size="small" type="danger" plain>删除</el-button>
              </template>
            </el-popconfirm>
          </div>
        </div>
      </template>
      <el-alert v-if="t.state.ipError" type="error" :title="t.state.ipError" :closable="false" class="ip-error" />
      <el-table :data="t.records" size="small">
        <el-table-column label="" width="40">
          <template #default="{ row }">
            <el-icon v-if="!row.enable" title="已停用"><CircleClose /></el-icon>
          </template>
        </el-table-column>
        <el-table-column label="域名" min-width="200">
          <template #default="{ row }">
            <span class="mono">{{ fqdn(row) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="type" label="类型" width="70" />
        <el-table-column label="当前值" min-width="200">
          <template #default="{ row }">
            <span class="mono">{{ t.state.records[row.key]?.value || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="220">
          <template #default="{ row }">
            <template v-if="t.state.records[row.key]">
              <el-tag v-if="t.state.records[row.key].lastError" type="danger" size="small" :title="t.state.records[row.key].lastError">
                {{ shorten(t.state.records[row.key].lastError ?? '') }}
              </el-tag>
              <el-tag v-else type="success" size="small">{{ actionText(t.state.records[row.key].lastAction) }}</el-tag>
              <span class="meta" v-if="t.state.records[row.key].lastSync">{{ fmtTime(t.state.records[row.key].lastSync ?? "") }}</span>
            </template>
            <span v-else class="meta">等待检查</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never" header="日志" class="logs">
      <LogViewer url="/api/ddns/logs" clearable />
    </el-card>

    <el-dialog v-model="editorVisible" :title="editing.key ? '编辑任务' : '添加任务'" width="860px" top="5vh">
      <el-form :model="editing" label-width="120px">
        <el-tabs v-model="tab">
          <el-tab-pane label="基本" name="basic">
            <el-form-item label="任务名称"><el-input v-model="editing.name" /></el-form-item>
            <el-form-item label="DNS 服务商">
              <el-select v-model="editing.provider" filterable style="width: 260px" @change="onProviderChange">
                <el-option v-for="p in providers" :key="p.name" :label="p.title" :value="p.name" />
              </el-select>
              <el-link v-if="currentProvider?.docUrl" :href="currentProvider.docUrl" target="_blank" class="doc">获取凭据</el-link>
            </el-form-item>
            <el-form-item v-if="currentProvider?.idLabel" :label="currentProvider.idLabel">
              <el-input v-model="editing.credentials.id" />
            </el-form-item>
            <el-form-item v-if="currentProvider?.secretLabel" :label="currentProvider.secretLabel">
              <el-input v-model="editing.credentials.secret" type="password" show-password placeholder="已保存的密钥显示为 ********，不修改则保持原值" />
            </el-form-item>
            <el-form-item v-for="f in currentProvider?.extra ?? []" :key="f.key" :label="f.label" :required="f.required">
              <el-input v-model="extra[f.key]" :type="f.multiline ? 'textarea' : 'text'" :rows="3" :placeholder="f.placeholder" />
            </el-form-item>
            <el-form-item label="检查周期(秒)">
              <el-input-number v-model="editing.interval" :min="30" :max="86400" />
            </el-form-item>
            <el-form-item label="强制同步(秒)">
              <el-input-number v-model="editing.forceInterval" :min="0" :max="604800" />
              <span class="help">即使 IP 未变化也定期向服务商核对，0 表示关闭</span>
            </el-form-item>
            <el-form-item label="请求超时(秒)">
              <el-input-number v-model="editing.httpTimeout" :min="1" :max="120" />
            </el-form-item>
            <el-form-item label="备注"><el-input v-model="editing.remark" /></el-form-item>
            <el-form-item label="启用"><el-switch v-model="editing.enable" /></el-form-item>
          </el-tab-pane>

          <el-tab-pane label="解析记录" name="records">
            <el-table :data="editing.records" size="small">
              <el-table-column label="启用" width="60">
                <template #default="{ row }"><el-switch v-model="row.enable" size="small" /></template>
              </el-table-column>
              <el-table-column label="主机记录" width="120">
                <template #default="{ row }"><el-input v-model="row.sub" size="small" placeholder="@" /></template>
              </el-table-column>
              <el-table-column label="主域名" min-width="150">
                <template #default="{ row }"><el-input v-model="row.zone" size="small" placeholder="example.com" /></template>
              </el-table-column>
              <el-table-column label="类型" width="100">
                <template #default="{ row }">
                  <el-select v-model="row.type" size="small" @change="onTypeChange(row)">
                    <el-option v-for="ty in ['A', 'AAAA', 'CNAME', 'TXT']" :key="ty" :label="ty" :value="ty" />
                  </el-select>
                </template>
              </el-table-column>
              <el-table-column label="记录值" min-width="150">
                <template #default="{ row }"><el-input v-model="row.value" size="small" class="mono" /></template>
              </el-table-column>
              <el-table-column label="TTL" width="90">
                <template #default="{ row }"><el-input-number v-model="row.ttl" size="small" :min="0" :controls="false" style="width: 70px" /></template>
              </el-table-column>
              <el-table-column v-if="currentProvider?.supportsLine" label="线路" width="90">
                <template #default="{ row }"><el-input v-model="row.line" size="small" placeholder="默认" /></template>
              </el-table-column>
              <el-table-column width="50">
                <template #default="{ $index }">
                  <el-button link type="danger" @click="editing.records.splice($index, 1)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
            <div class="records-ops">
              <el-button size="small" @click="addRecord('A')">添加 IPv4 记录</el-button>
              <el-button size="small" @click="addRecord('AAAA')">添加 IPv6 记录</el-button>
              <span class="help">记录值可使用 <span class="mono">{ipv4Addr}</span> <span class="mono">{ipv6Addr}</span>，TTL 为 0 时使用服务商默认值</span>
            </div>
          </el-tab-pane>

          <el-tab-pane label="IP 来源" name="ip">
            <IPSourceEditor v-model="editing.ipv4" :family="4" :interfaces="interfaces" />
            <el-divider />
            <IPSourceEditor v-model="editing.ipv6" :family="6" :interfaces="interfaces" />
          </el-tab-pane>

          <el-tab-pane label="Webhook" name="webhook">
            <WebhookEditor
              v-model="editing.webhook"
              :variables="['ipAddr', 'ipv4Addr', 'ipv6Addr', 'successDomains', 'failedDomains', 'successDomainsLine', 'failedDomainsLine', 'taskName', 'time']"
            />
          </el-tab-pane>
        </el-tabs>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { CircleClose } from '@element-plus/icons-vue'
import { api } from '../api'
import LogViewer from '../components/LogViewer.vue'
import WebhookEditor, { type WebhookConfig } from '../components/WebhookEditor.vue'
import IPSourceEditor, { type IPSource, type NetInterface } from '../components/IPSourceEditor.vue'

interface ProviderMeta {
  name: string
  title: string
  idLabel?: string
  secretLabel?: string
  supportsLine?: boolean
  docUrl?: string
  extra?: { key: string; label: string; placeholder?: string; multiline?: boolean; required?: boolean }[]
}

interface DNSRecord {
  key?: string
  enable: boolean
  zone: string
  sub: string
  type: string
  value: string
  ttl: number
  line: string
  remark: string
}

interface Task {
  key?: string
  name: string
  enable: boolean
  ipv4: IPSource
  ipv6: IPSource
  provider: string
  credentials: { id: string; secret: string; extra?: Record<string, string> }
  records: DNSRecord[]
  interval: number
  forceInterval: number
  httpTimeout: number
  webhook: WebhookConfig
  remark: string
}

interface RecordState {
  value?: string
  lastSync?: string
  lastAction?: string
  lastError?: string
}

interface TaskView extends Task {
  key: string
  state: {
    ipv4?: string
    ipv6?: string
    ipError?: string
    lastRun?: string
    nextRun?: string
    running: boolean
    records: Record<string, RecordState>
  }
}

const tasks = ref<TaskView[]>([])
const providers = ref<ProviderMeta[]>([])
const interfaces = ref<NetInterface[]>([])
const defaults = reactive({ ipv4URLs: [] as string[], ipv6URLs: [] as string[] })
const settings = reactive({ enable: true })
const loading = ref(true)
const editorVisible = ref(false)
const saving = ref(false)
const tab = ref('basic')
const editing = ref<Task>(emptyTask())
const extra = ref<Record<string, string>>({})
let timer: number | undefined

const currentProvider = computed(() => providers.value.find((p) => p.name === editing.value.provider))

function emptySource(urls: string[]): IPSource {
  return { enable: false, type: 'url', urls: [...urls], interface: '', regex: '', command: '' }
}

function emptyTask(): Task {
  return {
    name: '',
    enable: true,
    ipv4: { ...emptySource(defaults.ipv4URLs), enable: true },
    ipv6: emptySource(defaults.ipv6URLs),
    provider: 'alidns',
    credentials: { id: '', secret: '' },
    records: [],
    interval: 300,
    forceInterval: 3600,
    httpTimeout: 20,
    webhook: { enable: false, url: '', method: 'GET', headers: [], body: '', successContent: [], timeout: 0, insecureSkipVerify: false },
    remark: '',
  }
}

function providerTitle(name: string) {
  return providers.value.find((p) => p.name === name)?.title ?? name
}

function fqdn(r: DNSRecord) {
  return !r.sub || r.sub === '@' ? r.zone : `${r.sub}.${r.zone}`
}

function fmtTime(t: string) {
  return new Date(t).toLocaleTimeString()
}

function shorten(s: string) {
  return s.length > 40 ? s.slice(0, 40) + '…' : s
}

function actionText(a?: string) {
  return ({ created: '已创建', updated: '已更新', unchanged: '已是最新' } as Record<string, string>)[a ?? ''] ?? '已同步'
}

async function load() {
  try {
    tasks.value = await api.get<TaskView[]>('/api/ddns/tasks')
  } finally {
    loading.value = false
  }
}

function openEditor(t?: TaskView) {
  if (t) {
    const { state: _state, ...task } = t
    editing.value = JSON.parse(JSON.stringify(task))
  } else {
    editing.value = emptyTask()
    addRecord('A')
  }
  extra.value = { ...(editing.value.credentials.extra ?? {}) }
  tab.value = 'basic'
  editorVisible.value = true
}

function onProviderChange() {
  extra.value = {}
}

function addRecord(type: 'A' | 'AAAA') {
  const last = editing.value.records[editing.value.records.length - 1]
  editing.value.records.push({
    enable: true,
    zone: last?.zone ?? '',
    sub: '',
    type,
    value: type === 'A' ? '{ipv4Addr}' : '{ipv6Addr}',
    ttl: 0,
    line: '',
    remark: '',
  })
  if (type === 'AAAA') {
    editing.value.ipv6.enable = true
  } else {
    editing.value.ipv4.enable = true
  }
}

function onTypeChange(r: DNSRecord) {
  if (r.type === 'A' && (r.value === '' || r.value === '{ipv6Addr}')) {
    r.value = '{ipv4Addr}'
  } else if (r.type === 'AAAA' && (r.value === '' || r.value === '{ipv4Addr}')) {
    r.value = '{ipv6Addr}'
  }
}

async function save() {
  editing.value.credentials.extra = { ...extra.value }
  saving.value = true
  try {
    const { key, ...body } = editing.value
    if (key) {
      await api.put(`/api/ddns/tasks/${key}`, body)
    } else {
      await api.post('/api/ddns/tasks', body)
    }
    editorVisible.value = false
    ElMessage.success('已保存，正在检查')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function toggle(t: TaskView, enable: boolean) {
  try {
    await api.put(`/api/ddns/tasks/${t.key}/enable`, { enable })
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function sync(t: TaskView, force: boolean) {
  try {
    await api.post(`/api/ddns/tasks/${t.key}/sync${force ? '?force=1' : ''}`)
    ElMessage.success('已触发')
    window.setTimeout(load, 1500)
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function remove(t: TaskView) {
  try {
    await api.del(`/api/ddns/tasks/${t.key}`)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function saveSettings() {
  try {
    await api.put('/api/ddns/settings', settings)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(async () => {
  const [p, i, d, s] = await Promise.all([
    api.get<ProviderMeta[]>('/api/ddns/providers'),
    api.get<NetInterface[]>('/api/ddns/netinterfaces'),
    api.get<{ ipv4URLs: string[]; ipv6URLs: string[] }>('/api/ddns/defaults'),
    api.get<{ enable: boolean }>('/api/ddns/settings'),
  ])
  providers.value = p
  interfaces.value = i
  Object.assign(defaults, d)
  settings.enable = s.enable
  await load()
  timer = window.setInterval(load, 5000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.module-switch {
  margin-right: 16px;
}
.task {
  margin-bottom: 14px;
}
.task-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.task-title {
  display: flex;
  align-items: center;
  gap: 8px;
}
.meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin: 0 8px;
}
.ip-error {
  margin-bottom: 10px;
}
.doc {
  margin-left: 12px;
}
.help {
  margin-left: 10px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.records-ops {
  margin-top: 10px;
}
.logs {
  margin-top: 16px;
}
</style>
