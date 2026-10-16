<template>
  <div>
    <PageHeader title="STUN 穿透" desc="NAT1（全锥形）网络下，无需公网 IP 即可从外部访问内网服务" :icon="Radar">
      <el-button :icon="SlidersHorizontal" @click="openSettings">全局设置</el-button>
      <el-button type="primary" :icon="Plus" @click="openEditor()">添加规则</el-button>
    </PageHeader>

    <ModuleBanner v-if="moduleOff" title="STUN 穿透模块已停用" :loading="enabling" @enable="enableModule" />

    <el-alert type="info" :closable="false" class="intro" show-icon>
      在通道端口上维持映射并通过 STUN 获取公网地址，外部访问经内置转发到达内网服务。公网端口由运营商分配，变化时会触发 Webhook 与脚本。
    </el-alert>

    <div v-loading="loading" class="card-list">
      <el-empty v-if="!loading && !rules.length" description="暂无规则，可添加或在「设置 → 备份与迁移」中导入 Lucky 配置" />
      <EntityCard v-for="r in rules" :key="r.key" :off="!r.enable || moduleOff">
        <template #lead>
          <el-switch :model-value="r.enable" @change="(v: string | number | boolean) => toggle(r, Boolean(v))" />
        </template>
        <template #title>
          <span>{{ r.name }}</span>
          <span class="tags">
            <el-tag size="small" effect="plain">{{ r.protocol === 'udp4' ? 'UDP' : 'TCP' }}</el-tag>
            <el-tag v-if="r.upnp" size="small" type="info">UPnP</el-tag>
            <el-tag v-if="r.natpmp" size="small" type="info">NAT-PMP</el-tag>
          </span>
        </template>
        <template v-if="r.remark" #sub>{{ r.remark }}</template>
        <template #status>
          <StatusBadge v-bind="ruleStatus(r)" />
        </template>
        <Field label="公网地址" mono>
          <template v-if="r.enable && !moduleOff && r.state.publicAddr">
            <span class="copyable" title="点击复制" @click="copy(r.state.publicAddr)">{{ r.state.publicAddr }}</span>
            <div v-if="r.state.since" class="meta">自 {{ fmtTime(r.state.since) }}</div>
          </template>
          <span v-else class="muted">-</span>
        </Field>
        <Field label="通道" mono>{{ r.state.localAddr || (r.localPort ? ':' + r.localPort : '随机端口') }}</Field>
        <Field label="目标" mono>{{ r.disableForward ? '路由器转发' : `${(r.targetAddresses ?? []).join(' | ')}:${r.targetPort}` }}</Field>
        <Field label="连接 / 流量">
          {{ r.state.stats.activeConns }} 活动 · {{ r.state.stats.totalConns }} 累计
          <div class="meta">↑ {{ formatBytes(r.state.stats.bytesIn) }} · ↓ {{ formatBytes(r.state.stats.bytesOut) }}</div>
        </Field>
        <template v-if="r.enable && !moduleOff && (r.state.error || r.state.mapping)" #extra>
          <div v-if="r.state.error" class="err-line">{{ r.state.error }}</div>
          <div v-if="r.state.mapping" class="meta">{{ r.state.mapping }}</div>
        </template>
        <template #actions>
          <el-button text size="small" :icon="RotateCw" :disabled="!r.enable || moduleOff" @click="restart(r)">重连</el-button>
          <el-button text size="small" :icon="Pencil" @click="openEditor(r)">编辑</el-button>
          <el-popconfirm title="确定删除这条规则？" @confirm="remove(r)">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Trash2">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </EntityCard>
    </div>

    <LogCard url="/api/stun/logs" clearable />

    <el-dialog v-model="editorVisible" :title="editing.key ? '编辑规则' : '添加规则'" width="780px" top="5vh" :fullscreen="isMobile">
      <el-form :model="editing" label-width="140px" :label-position="labelPosition">
        <el-tabs v-model="tab">
          <el-tab-pane label="基本" name="basic">
            <el-form-item label="规则名称" required><el-input v-model="editing.name" placeholder="名称会作为 ruleName 变量传给 Webhook 与脚本" /></el-form-item>
            <el-form-item label="穿透类型">
              <el-radio-group v-model="editing.protocol">
                <el-radio-button value="tcp4">TCP</el-radio-button>
                <el-radio-button value="udp4">UDP</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="通道本地端口">
              <el-input-number v-model="editing.localPort" :min="0" :max="65535" />
              <span class="help">0 为随机；主路由不是本机时需在路由器把该端口转发到本机（或开启 UPnP）</span>
            </el-form-item>
            <el-form-item label="不使用内置转发">
              <el-switch v-model="editing.disableForward" />
              <span class="help">由路由器把通道端口直接转发到目标，性能更好，但需固定通道端口</span>
            </el-form-item>
            <template v-if="!editing.disableForward">
              <el-form-item label="目标地址" required>
                <el-input v-model="targetsText" placeholder="例如 192.168.1.10，多个用逗号分隔（按顺序故障转移）" />
              </el-form-item>
              <el-form-item label="目标端口" required>
                <el-input-number v-model="editing.targetPort" :min="1" :max="65535" />
              </el-form-item>
            </template>
            <el-form-item label="备注"><el-input v-model="editing.remark" /></el-form-item>
            <el-form-item label="启用"><el-switch v-model="editing.enable" /></el-form-item>
          </el-tab-pane>

          <el-tab-pane label="通道" name="channel">
            <el-form-item label="通道绑定">
              <el-radio-group v-model="editing.bind.mode">
                <el-radio-button value="ip">指定 IP</el-radio-button>
                <el-radio-button value="interface">指定网卡</el-radio-button>
              </el-radio-group>
              <span class="help">非多拨或多线路用户保持默认即可</span>
            </el-form-item>
            <el-form-item v-if="editing.bind.mode === 'ip'" label="通道 IP">
              <el-input v-model="editing.bind.ip" placeholder="留空表示所有地址" />
            </el-form-item>
            <template v-else>
              <el-form-item label="网卡">
                <el-select v-model="editing.bind.interface" filterable allow-create style="width: 100%">
                  <el-option v-for="i in interfaces" :key="i.name" :label="i.name" :value="i.name">
                    <span>{{ i.name }}</span><span class="iface">{{ i.addrs.join('  ') }}</span>
                  </el-option>
                </el-select>
              </el-form-item>
              <el-form-item label="IP 选择规则">
                <el-input v-model="editing.bind.regex" placeholder="留空取第 1 个；数字 n 取第 n 个；24* 前缀；*24 后缀；或正则" />
              </el-form-item>
            </template>
            <el-form-item label="使用全局 STUN 服务器">
              <el-switch v-model="editing.useGlobalServers" />
            </el-form-item>
            <el-form-item v-if="!editing.useGlobalServers" label="STUN 服务器">
              <el-input v-model="serversText" type="textarea" :rows="4" class="mono" placeholder="每行一个 host:port；TCP 穿透需服务器支持 TCP" />
            </el-form-item>
            <el-form-item v-if="editing.protocol === 'tcp4'" label="保活服务器">
              <el-input v-model="keepAliveText" type="textarea" :rows="2" class="mono" placeholder="留空使用全局设置；每行一个 host:port（HTTP）" />
            </el-form-item>
          </el-tab-pane>

          <el-tab-pane label="路由器映射" name="mapping">
            <el-form-item label="UPnP"><el-switch v-model="editing.upnp" /></el-form-item>
            <template v-if="editing.upnp">
              <el-form-item label="网关 IP"><el-input v-model="editing.upnpGateway" placeholder="留空自动发现，例如 192.168.1.1" /></el-form-item>
              <el-form-item label="本机局域网 IP"><el-input v-model="editing.upnpLocalHost" placeholder="Docker 桥接网络时填宿主机 IP，其余留空" /></el-form-item>
              <el-form-item label="控制接口地址"><el-input v-model="editing.upnpControlUrl" placeholder="无法自动发现时填写，其余留空" /></el-form-item>
            </template>
            <el-form-item label="NAT-PMP"><el-switch v-model="editing.natpmp" /></el-form-item>
            <el-form-item v-if="editing.natpmp" label="NAT-PMP 网关"><el-input v-model="editing.natpmpGateway" placeholder="路由器 IP，留空使用 UPnP 网关 IP" /></el-form-item>
            <el-form-item v-if="editing.upnp || editing.natpmp" label="内部端口">
              <el-input-number v-model="editing.upnpInternalPort" :min="0" :max="65535" />
              <span class="help">0 表示映射到通道端口；填写后路由器直接转发到本机该端口</span>
            </el-form-item>
          </el-tab-pane>

          <el-tab-pane label="高级" name="advanced">
            <el-form-item label="心跳间隔(毫秒)"><el-input-number v-model="editing.heartbeatInterval" :min="500" :max="60000" :step="500" /></el-form-item>
            <el-form-item label="穿透超时(毫秒)"><el-input-number v-model="editing.stunTimeout" :min="500" :max="30000" :step="500" /></el-form-item>
            <el-form-item label="重试间隔(毫秒)"><el-input-number v-model="editing.retryInterval" :min="500" :max="600000" :step="500" /></el-form-item>
            <el-form-item label="失败自动重试"><el-switch v-model="editing.autoRetry" /></el-form-item>
            <el-form-item label="关闭有效性检测">
              <el-switch v-model="editing.disableCheck" />
              <span class="help">检测会连接自身公网地址确认可达</span>
            </el-form-item>
            <el-form-item label="最大连接数"><el-input-number v-model="editing.maxConnections" :min="0" /></el-form-item>
            <el-form-item v-if="editing.protocol === 'udp4'" label="UDP 会话超时(秒)"><el-input-number v-model="editing.udpSessionTimeout" :min="0" /></el-form-item>
          </el-tab-pane>

          <el-tab-pane label="通知" name="notify">
            <el-form-item label="仅地址变化时通知"><el-switch v-model="editing.webhookOnlyOnChange" /></el-form-item>
            <el-form-item label="使用全局 Webhook"><el-switch v-model="editing.useGlobalWebhook" /></el-form-item>
            <WebhookEditor v-if="!editing.useGlobalWebhook" v-model="editing.webhook" :variables="vars" />
            <el-divider />
            <el-form-item label="执行脚本"><el-switch v-model="editing.scriptEnable" /></el-form-item>
            <el-form-item v-if="editing.scriptEnable" label="脚本内容">
              <el-input v-model="editing.script" type="textarea" :rows="5" class="mono" placeholder='例如 /data/scripts/notify.sh --rule "${ruleName}" --port "${port}"' />
              <div class="form-help">
                以 sh -c 执行，变量通过环境变量传入：<span class="mono">ruleName time ip port ipAddr</span>（同时提供 GATELY_ 前缀版本）
              </div>
            </el-form-item>
          </el-tab-pane>
        </el-tabs>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="settingsVisible" title="STUN 全局设置" width="700px" :fullscreen="isMobile">
      <el-form v-if="settings" :model="settings" label-width="140px" :label-position="labelPosition">
        <el-form-item label="启用模块"><el-switch v-model="settings.enable" /></el-form-item>
        <el-form-item label="STUN 服务器">
          <el-input v-model="settingsServers" type="textarea" :rows="6" class="mono" placeholder="每行一个 host:port" />
        </el-form-item>
        <el-form-item label="保活服务器">
          <el-input v-model="settingsKeepAlive" type="textarea" :rows="2" class="mono" />
        </el-form-item>
        <el-form-item label="地址检测周期(秒)"><el-input-number v-model="settings.addrCheckInterval" :min="10" :max="3600" /></el-form-item>
        <el-divider content-position="left">全局 Webhook</el-divider>
        <WebhookEditor v-model="settings.webhook" :variables="vars" />
      </el-form>
      <template #footer>
        <el-button @click="settingsVisible = false">取消</el-button>
        <el-button type="primary" @click="saveSettings">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Pencil, Plus, Radar, RotateCw, SlidersHorizontal, Trash2 } from 'lucide-vue-next'
import { api, formatBytes } from '../api'
import PageHeader from '../components/PageHeader.vue'
import ModuleBanner from '../components/ModuleBanner.vue'
import EntityCard from '../components/EntityCard.vue'
import Field from '../components/Field.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import { useFormLayout } from '../composables/useIsMobile'
import WebhookEditor, { type WebhookConfig } from '../components/WebhookEditor.vue'
import type { NetInterface } from '../components/IPSourceEditor.vue'

interface Rule {
  key?: string
  name: string
  enable: boolean
  protocol: string
  bind: { mode: string; ip: string; interface: string; regex: string }
  localPort: number
  useGlobalServers: boolean
  stunServers: string[] | null
  keepAliveServers: string[] | null
  disableForward: boolean
  targetAddresses: string[] | null
  targetPort: number
  upnp: boolean
  upnpGateway: string
  upnpLocalHost: string
  upnpControlUrl: string
  upnpInternalPort: number
  natpmp: boolean
  natpmpGateway: string
  heartbeatInterval: number
  stunTimeout: number
  retryInterval: number
  autoRetry: boolean
  disableCheck: boolean
  useGlobalWebhook: boolean
  webhook: WebhookConfig
  webhookOnlyOnChange: boolean
  scriptEnable: boolean
  script: string
  maxConnections: number
  udpSessionTimeout: number
  remark: string
}

interface RuleView extends Rule {
  key: string
  state: {
    status: string
    publicAddr?: string
    localAddr?: string
    since?: string
    error?: string
    mapping?: string
    stats: { activeConns: number; totalConns: number; bytesIn: number; bytesOut: number }
  }
}

interface Settings {
  enable: boolean
  stunServers: string[]
  keepAliveServers: string[]
  webhook: WebhookConfig
  addrCheckInterval: number
}

const vars = ['ruleName', 'time', 'ip', 'port', 'ipAddr']
const { isMobile, labelPosition } = useFormLayout()
// moduleEnabled 为模块全局开关，undefined 表示尚未加载
const moduleEnabled = ref<boolean>()
const moduleOff = computed(() => moduleEnabled.value === false)
const enabling = ref(false)
const rules = ref<RuleView[]>([])
const interfaces = ref<NetInterface[]>([])
const loading = ref(true)
const editorVisible = ref(false)
const saving = ref(false)
const tab = ref('basic')
const editing = ref<Rule>(emptyRule())
const settingsVisible = ref(false)
const settings = ref<Settings>()
let timer: number | undefined

const lines = (v: string) => v.split(/[\n,]/).map((s) => s.trim()).filter(Boolean)
const targetsText = computed({
  get: () => (editing.value.targetAddresses ?? []).join(', '),
  set: (v: string) => (editing.value.targetAddresses = lines(v)),
})
const serversText = computed({
  get: () => (editing.value.stunServers ?? []).join('\n'),
  set: (v: string) => (editing.value.stunServers = lines(v)),
})
const keepAliveText = computed({
  get: () => (editing.value.keepAliveServers ?? []).join('\n'),
  set: (v: string) => (editing.value.keepAliveServers = lines(v)),
})
const settingsServers = computed({
  get: () => (settings.value?.stunServers ?? []).join('\n'),
  set: (v: string) => settings.value && (settings.value.stunServers = lines(v)),
})
const settingsKeepAlive = computed({
  get: () => (settings.value?.keepAliveServers ?? []).join('\n'),
  set: (v: string) => settings.value && (settings.value.keepAliveServers = lines(v)),
})

function emptyRule(): Rule {
  return {
    name: '',
    enable: true,
    protocol: 'tcp4',
    bind: { mode: 'ip', ip: '', interface: '', regex: '' },
    localPort: 0,
    useGlobalServers: true,
    stunServers: [],
    keepAliveServers: [],
    disableForward: false,
    targetAddresses: [],
    targetPort: 0,
    upnp: false,
    upnpGateway: '',
    upnpLocalHost: '',
    upnpControlUrl: '',
    upnpInternalPort: 0,
    natpmp: false,
    natpmpGateway: '',
    heartbeatInterval: 3000,
    stunTimeout: 3000,
    retryInterval: 3000,
    autoRetry: true,
    disableCheck: false,
    useGlobalWebhook: false,
    webhook: { enable: false, url: '', method: 'GET', headers: [], body: '', successContent: [], timeout: 0, insecureSkipVerify: false },
    webhookOnlyOnChange: true,
    scriptEnable: false,
    script: '',
    maxConnections: 0,
    udpSessionTimeout: 0,
    remark: '',
  }
}

// ruleStatus 计算规则展示状态：停用的规则只显示“已停用”
function ruleStatus(r: RuleView): { type: StatusType; text: string; title?: string } {
  if (moduleOff.value) {
    return { type: 'off', text: '模块已停用' }
  }
  if (!r.enable) {
    return { type: 'off', text: '已停用' }
  }
  const map: Record<string, { type: StatusType; text: string }> = {
    ok: { type: 'running', text: '已穿透' },
    connecting: { type: 'pending', text: '连接中' },
    retrying: { type: 'warn', text: '重试中' },
    failed: { type: 'error', text: '失败' },
    stopped: { type: 'warn', text: '未运行' },
  }
  return { ...(map[r.state.status] ?? { type: 'info', text: r.state.status }), title: r.state.error }
}

function fmtTime(t: string) {
  return new Date(t).toLocaleString()
}

async function copy(s: string) {
  await navigator.clipboard?.writeText(s)
  ElMessage.success('已复制 ' + s)
}

async function load() {
  try {
    rules.value = await api.get<RuleView[]>('/api/stun/rules')
  } finally {
    loading.value = false
  }
}

function openEditor(row?: RuleView) {
  if (row) {
    const { state: _s, ...rule } = row
    editing.value = JSON.parse(JSON.stringify(rule))
  } else {
    editing.value = emptyRule()
  }
  tab.value = 'basic'
  editorVisible.value = true
}

async function save() {
  saving.value = true
  try {
    const { key, ...body } = editing.value
    if (key) {
      await api.put(`/api/stun/rules/${key}`, body)
    } else {
      await api.post('/api/stun/rules', body)
    }
    editorVisible.value = false
    ElMessage.success('已保存')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function toggle(row: RuleView, enable: boolean) {
  try {
    await api.put(`/api/stun/rules/${row.key}/enable`, { enable })
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function restart(row: RuleView) {
  try {
    await api.post(`/api/stun/rules/${row.key}/restart`)
    ElMessage.success('已重新连接')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function remove(row: RuleView) {
  try {
    await api.del(`/api/stun/rules/${row.key}`)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function loadModule() {
  const s = await api.get<Settings>('/api/stun/settings')
  moduleEnabled.value = s.enable
  return s
}

async function openSettings() {
  settings.value = await loadModule()
  settingsVisible.value = true
}

// enableModule 在横幅中一键启用模块
async function enableModule() {
  enabling.value = true
  try {
    const s = await loadModule()
    await api.put('/api/stun/settings', { ...s, enable: true })
    ElMessage.success('STUN 穿透模块已启用')
    await Promise.all([loadModule(), load()])
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    enabling.value = false
  }
}

async function saveSettings() {
  try {
    await api.put('/api/stun/settings', settings.value)
    settingsVisible.value = false
    ElMessage.success('已保存，规则已重启')
    await Promise.all([loadModule(), load()])
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(async () => {
  loadModule()
  interfaces.value = await api.get<NetInterface[]>('/api/stun/netinterfaces')
  await load()
  timer = window.setInterval(load, 3000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.intro {
  margin-bottom: 16px;
}
.err-line {
  font-size: 12px;
  color: var(--el-color-danger);
  word-break: break-all;
}
.iface {
  float: right;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-left: 16px;
}
</style>
