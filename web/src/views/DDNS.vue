<template>
  <div>
    <PageHeader title="动态域名" desc="定期获取公网 IP，变化时自动更新 DNS 解析记录" :icon="Globe">
      <div class="module-toggle">
        <span class="meta">模块</span>
        <el-switch v-model="settings.enable" inline-prompt active-text="开" inactive-text="关" @change="saveSettings" />
      </div>
      <el-button type="primary" :icon="Plus" @click="openEditor()">添加任务</el-button>
    </PageHeader>

    <ModuleBanner v-if="moduleOff" title="动态域名模块已停用" desc="模块停用期间不会检查 IP，也不会更新任何解析记录；启用后各任务按自身开关运行。" :loading="enabling" @enable="enableModule" />

    <div v-loading="loading" class="card-list">
      <el-empty v-if="!loading && !tasks.length" description="暂无任务，可添加或在「设置 → 备份与迁移」中导入 Lucky 配置" />
      <EntityCard v-for="t in tasks" :key="t.key" :off="!t.enable || moduleOff">
        <template #lead>
          <el-switch :model-value="t.enable" @change="(v: string | number | boolean) => toggle(t, Boolean(v))" />
        </template>
        <template #title>
          <span>{{ t.name || '未命名任务' }}</span>
          <el-tag size="small" effect="plain">{{ providerTitle(t.provider) }}</el-tag>
        </template>
        <template v-if="t.remark" #sub>{{ t.remark }}</template>
        <template #status>
          <StatusBadge v-bind="taskStatus(t)" />
        </template>
        <Field label="IPv4" mono>{{ t.ipv4.enable ? t.state.ipv4 || (taskActive(t) ? '获取中…' : '-') : '未启用' }}</Field>
        <Field label="IPv6" mono>{{ t.ipv6.enable ? t.state.ipv6 || (taskActive(t) ? '获取中…' : '-') : '未启用' }}</Field>
        <Field label="上次检查">{{ t.state.lastRun ? fmtTime(t.state.lastRun) : '-' }}</Field>
        <Field label="下次检查">{{ taskActive(t) && t.state.nextRun ? fmtTime(t.state.nextRun) : '-' }}</Field>
        <template #extra>
          <el-alert v-if="t.state.ipError && taskActive(t)" type="error" :title="t.state.ipError" :closable="false" show-icon class="ip-error" />
          <div class="records">
            <div v-for="rec in t.records" :key="rec.key" class="rec" :class="{ 'is-off': !rec.enable }">
              <div class="rec-name">
                <span class="mono rec-domain">{{ fqdn(rec) }}</span>
                <el-tag size="small" type="info" class="mono">{{ rec.type }}</el-tag>
              </div>
              <div class="rec-value mono">{{ (rec.key && t.state.records?.[rec.key]?.value) || '-' }}</div>
              <div class="rec-status">
                <StatusBadge v-bind="recordStatus(t, rec)" />
                <span v-if="recordSync(t, rec)" class="meta">{{ recordSync(t, rec) }}</span>
              </div>
            </div>
            <div v-if="!t.records.length" class="meta">没有解析记录</div>
          </div>
        </template>
        <template #actions>
          <el-button text size="small" :icon="RefreshCw" :disabled="!taskActive(t) || !t.state.running" @click="sync(t, false)">检查</el-button>
          <el-button text size="small" :icon="Zap" :disabled="!taskActive(t) || !t.state.running" @click="sync(t, true)">强制同步</el-button>
          <el-button text size="small" :icon="Pencil" @click="openEditor(t)">编辑</el-button>
          <el-popconfirm title="确定删除该任务？不会删除服务商处的解析记录" width="240" @confirm="remove(t)">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Trash2">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </EntityCard>
    </div>

    <LogCard url="/api/ddns/logs" clearable />

    <el-dialog v-model="editorVisible" :title="editing.key ? '编辑任务' : '添加任务'" width="900px" top="5vh" :fullscreen="isMobile">
      <el-form :model="editing" label-width="120px" :label-position="labelPosition">
        <el-tabs v-model="tab">
          <el-tab-pane label="基本" name="basic">
            <el-form-item label="任务名称"><el-input v-model="editing.name" /></el-form-item>
            <el-form-item label="DNS 服务商">
              <el-select v-model="editing.provider" filterable class="w-260" @change="onProviderChange">
                <el-option v-for="p in providers" :key="p.name" :label="p.title" :value="p.name" />
              </el-select>
              <el-link v-if="currentProvider?.docUrl" :href="currentProvider.docUrl" target="_blank" type="primary" class="doc">获取凭据</el-link>
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
            <div class="rec-edit-head">
              <span>启用</span><span>主机记录</span><span>主域名</span><span>类型</span><span>记录值</span><span>TTL</span>
              <span>{{ currentProvider?.supportsLine ? '线路' : '' }}</span><span />
            </div>
            <div v-for="(row, i) in editing.records" :key="i" class="rec-edit">
              <div class="re-cell re-enable"><label class="re-label">启用</label><el-switch v-model="row.enable" size="small" /></div>
              <div class="re-cell"><label class="re-label">主机记录</label><el-input v-model="row.sub" size="small" placeholder="@" /></div>
              <div class="re-cell"><label class="re-label">主域名</label><el-input v-model="row.zone" size="small" placeholder="example.com" /></div>
              <div class="re-cell">
                <label class="re-label">类型</label>
                <el-select v-model="row.type" size="small" @change="onTypeChange(row)">
                  <el-option v-for="ty in ['A', 'AAAA', 'CNAME', 'TXT']" :key="ty" :label="ty" :value="ty" />
                </el-select>
              </div>
              <div class="re-cell"><label class="re-label">记录值</label><el-input v-model="row.value" size="small" class="mono" /></div>
              <div class="re-cell"><label class="re-label">TTL</label><el-input-number v-model="row.ttl" size="small" :min="0" :controls="false" class="re-ttl" /></div>
              <div class="re-cell">
                <template v-if="currentProvider?.supportsLine">
                  <label class="re-label">线路</label>
                  <el-input v-model="row.line" size="small" placeholder="默认" />
                </template>
              </div>
              <div class="re-cell re-del">
                <el-button text type="danger" size="small" :icon="Trash2" aria-label="删除记录" @click="editing.records.splice(i, 1)" />
              </div>
            </div>
            <el-empty v-if="!editing.records.length" description="还没有解析记录" :image-size="60" />
            <div class="records-ops">
              <el-button size="small" :icon="Plus" @click="addRecord('A')">IPv4 记录</el-button>
              <el-button size="small" :icon="Plus" @click="addRecord('AAAA')">IPv6 记录</el-button>
            </div>
            <div class="form-help">记录值可使用 <span class="mono">{ipv4Addr}</span> <span class="mono">{ipv6Addr}</span>，TTL 为 0 时使用服务商默认值</div>
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
import { Globe, Pencil, Plus, RefreshCw, Trash2, Zap } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import ModuleBanner from '../components/ModuleBanner.vue'
import EntityCard from '../components/EntityCard.vue'
import Field from '../components/Field.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import WebhookEditor, { type WebhookConfig } from '../components/WebhookEditor.vue'
import IPSourceEditor, { type IPSource, type NetInterface } from '../components/IPSourceEditor.vue'
import { useFormLayout } from '../composables/useIsMobile'

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
    records: Record<string, RecordState> | null
  }
}

type Badge = { type: StatusType; text: string; title?: string }

const { isMobile, labelPosition } = useFormLayout()
const tasks = ref<TaskView[]>([])
const providers = ref<ProviderMeta[]>([])
const interfaces = ref<NetInterface[]>([])
const defaults = reactive({ ipv4URLs: [] as string[], ipv6URLs: [] as string[] })
const settings = reactive({ enable: true })
const settingsLoaded = ref(false)
const loading = ref(true)
const editorVisible = ref(false)
const saving = ref(false)
const enabling = ref(false)
const tab = ref('basic')
const editing = ref<Task>(emptyTask())
const extra = ref<Record<string, string>>({})
let timer: number | undefined

const currentProvider = computed(() => providers.value.find((p) => p.name === editing.value.provider))
const moduleOff = computed(() => settingsLoaded.value && !settings.enable)

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
  const d = new Date(t)
  // 当天只显示时间，其余显示日期与时间
  return d.toDateString() === new Date().toDateString() ? d.toLocaleTimeString() : d.toLocaleString()
}

function actionText(a?: string) {
  return ({ created: '已创建', updated: '已更新', unchanged: '已是最新' } as Record<string, string>)[a ?? ''] ?? '已同步'
}

// taskActive 表示任务实际处于运行条件：模块与任务均已启用
function taskActive(t: TaskView) {
  return !moduleOff.value && t.enable
}

function recState(t: TaskView, r: DNSRecord): RecordState | undefined {
  return r.key ? t.state.records?.[r.key] : undefined
}

// taskStatus 汇总任务状态。停用的任务只显示“已停用”，不会显示“等待检查”
function taskStatus(t: TaskView): Badge {
  if (moduleOff.value) {
    return { type: 'off', text: '模块已停用' }
  }
  if (!t.enable) {
    return { type: 'off', text: '已停用' }
  }
  if (t.state.ipError) {
    return { type: 'error', text: '获取 IP 失败', title: t.state.ipError }
  }
  const enabled = t.records.filter((r) => r.enable)
  const failed = enabled.filter((r) => recState(t, r)?.lastError).length
  if (failed) {
    return { type: 'error', text: `${failed} 条同步失败` }
  }
  if (!t.state.running) {
    return { type: 'pending', text: '启动中' }
  }
  if (!enabled.length) {
    return { type: 'warn', text: '无启用的记录' }
  }
  if (enabled.every((r) => recState(t, r))) {
    return { type: 'ok', text: '正常' }
  }
  return { type: 'pending', text: '等待检查' }
}

// recordStatus 计算单条记录状态：模块停用、任务停用、记录停用都显示为“已停用”
function recordStatus(t: TaskView, r: DNSRecord): Badge {
  if (moduleOff.value) {
    return { type: 'off', text: '模块已停用' }
  }
  if (!t.enable) {
    return { type: 'off', text: '已停用', title: '所属任务已停用' }
  }
  if (!r.enable) {
    return { type: 'off', text: '已停用', title: '该记录已停用' }
  }
  const st = recState(t, r)
  if (st?.lastError) {
    return { type: 'error', text: st.lastError, title: st.lastError }
  }
  if (st) {
    return { type: 'ok', text: actionText(st.lastAction) }
  }
  return { type: 'pending', text: '等待检查' }
}

function recordSync(t: TaskView, r: DNSRecord) {
  const st = recState(t, r)
  if (!taskActive(t) || !r.enable || !st?.lastSync) {
    return ''
  }
  return fmtTime(st.lastSync)
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
    ElMessage.success(body.enable && settings.enable ? '已保存，正在检查' : '已保存')
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
    settings.enable = !settings.enable
    ElMessage.error((e as Error).message)
  }
}

async function enableModule() {
  enabling.value = true
  settings.enable = true
  await saveSettings()
  enabling.value = false
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
  settingsLoaded.value = true
  await load()
  timer = window.setInterval(load, 5000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.module-toggle {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 32px;
  padding: 0 10px;
  border: 1px solid var(--g-border);
  border-radius: 8px;
  background: var(--el-bg-color);
}
.ip-error {
  margin-bottom: 10px;
}
.records {
  border: 1px solid var(--g-border);
  border-radius: 8px;
  overflow: hidden;
}
.records > .meta {
  padding: 10px 12px;
}
.rec {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(0, 1.4fr) minmax(0, 1.6fr);
  gap: 6px 16px;
  align-items: center;
  padding: 9px 12px;
  font-size: 13px;
}
.rec + .rec {
  border-top: 1px solid var(--g-border);
}
.rec.is-off .rec-name,
.rec.is-off .rec-value {
  opacity: 0.55;
}
.rec-name {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.rec-domain {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rec-value {
  color: var(--el-text-color-regular);
  word-break: break-all;
}
.rec-status {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.rec-status :deep(.status-badge) {
  max-width: 220px;
}
.doc {
  margin-left: 12px;
}
.w-260 {
  width: 260px;
}
/* 记录编辑行：桌面为表格式网格，移动端为卡片 */
.rec-edit-head,
.rec-edit {
  display: grid;
  grid-template-columns: 44px 110px minmax(0, 1.3fr) 90px minmax(0, 1.2fr) 70px 90px 36px;
  gap: 8px;
  align-items: center;
}
.rec-edit-head {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  padding: 0 4px 6px;
}
.rec-edit {
  padding: 6px 4px;
  border-top: 1px solid var(--g-border);
}
.re-label {
  display: none;
}
.re-ttl {
  width: 100%;
}
.records-ops {
  margin-top: 12px;
}
@media (max-width: 767.98px) {
  .rec {
    grid-template-columns: 1fr;
    gap: 4px;
  }
  .w-260 {
    width: 100%;
  }
  .doc {
    margin: 6px 0 0;
  }
  .rec-edit-head {
    display: none;
  }
  .rec-edit {
    grid-template-columns: 1fr 1fr;
    gap: 8px 10px;
    padding: 12px;
    margin-bottom: 10px;
    border: 1px solid var(--g-border);
    border-radius: 8px;
    position: relative;
  }
  .re-label {
    display: block;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin-bottom: 2px;
  }
  .re-cell:empty {
    display: none;
  }
  .re-enable {
    grid-column: 1 / -1;
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .re-enable .re-label {
    margin: 0;
  }
  .re-del {
    position: absolute;
    top: 6px;
    right: 6px;
  }
  .re-del + * {
    display: none;
  }
}
</style>
