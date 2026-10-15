<template>
  <div>
    <div class="page-header">
      <h2>STUN 穿透</h2>
      <div>
        <el-button @click="openSettings">全局设置</el-button>
        <el-button type="primary" @click="openEditor()">添加规则</el-button>
      </div>
    </div>

    <el-alert type="info" :closable="false" class="intro">
      适用于 NAT1（全锥形）网络：在通道端口上维持映射并通过 STUN 获取公网地址，外部访问经内置转发到达内网服务。公网端口由运营商分配，变化时会触发 Webhook 与脚本。
    </el-alert>

    <el-card shadow="never">
      <el-table :data="rules" v-loading="loading" empty-text="暂无规则，可添加或在设置中导入 Lucky 配置">
        <el-table-column label="启用" width="70">
          <template #default="{ row }">
            <el-switch :model-value="row.enable" size="small" @change="(v: string | number | boolean) => toggle(row, Boolean(v))" />
          </template>
        </el-table-column>
        <el-table-column label="名称" min-width="160">
          <template #default="{ row }">
            <div>{{ row.name }}</div>
            <el-tag size="small" effect="plain">{{ row.protocol.toUpperCase() }}</el-tag>
            <el-tag v-if="row.upnp" size="small" effect="plain" type="info" class="tag">UPnP</el-tag>
            <el-tag v-if="row.natpmp" size="small" effect="plain" type="info" class="tag">NAT-PMP</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="公网地址" min-width="190">
          <template #default="{ row }">
            <template v-if="row.state.publicAddr">
              <span class="mono addr" @click="copy(row.state.publicAddr)" title="点击复制">{{ row.state.publicAddr }}</span>
              <div class="remark" v-if="row.state.since">自 {{ fmtTime(row.state.since) }}</div>
            </template>
            <span v-else class="remark">-</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="200">
          <template #default="{ row }">
            <el-tag :type="statusType(row.state.status)" size="small">{{ statusText(row.state.status) }}</el-tag>
            <div v-if="row.state.error" class="error" :title="row.state.error">{{ row.state.error }}</div>
            <div v-if="row.state.mapping" class="remark">{{ row.state.mapping }}</div>
          </template>
        </el-table-column>
        <el-table-column label="通道 → 目标" min-width="200">
          <template #default="{ row }">
            <div class="mono">{{ row.state.localAddr || (row.localPort ? ':' + row.localPort : '随机端口') }}</div>
            <div class="mono remark">
              → {{ row.disableForward ? '路由器转发' : `${(row.targetAddresses ?? []).join('|')}:${row.targetPort}` }}
            </div>
          </template>
        </el-table-column>
        <el-table-column label="连接 / 流量" min-width="150">
          <template #default="{ row }">
            <div>{{ row.state.stats.activeConns }} 活动 · {{ row.state.stats.totalConns }} 累计</div>
            <div class="remark">↑ {{ formatBytes(row.state.stats.bytesIn) }} ↓ {{ formatBytes(row.state.stats.bytesOut) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="170" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" :disabled="!row.enable" @click="restart(row)">重连</el-button>
            <el-button link type="primary" @click="openEditor(row)">编辑</el-button>
            <el-popconfirm title="确定删除这条规则？" @confirm="remove(row)">
              <template #reference>
                <el-button link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never" header="日志" class="logs">
      <LogViewer url="/api/stun/logs" clearable />
    </el-card>

    <el-dialog v-model="editorVisible" :title="editing.key ? '编辑规则' : '添加规则'" width="780px" top="5vh">
      <el-form :model="editing" label-width="140px">
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
              <div class="help block">
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

    <el-dialog v-model="settingsVisible" title="STUN 全局设置" width="700px">
      <el-form v-if="settings" :model="settings" label-width="140px">
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
import { api, formatBytes } from '../api'
import LogViewer from '../components/LogViewer.vue'
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

function statusType(s: string) {
  return ({ ok: 'success', connecting: 'warning', retrying: 'warning', failed: 'danger' } as Record<string, string>)[s] ?? 'info'
}

function statusText(s: string) {
  return ({ ok: '已穿透', connecting: '连接中', retrying: '重试中', failed: '失败', stopped: '已停止' } as Record<string, string>)[s] ?? s
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
  await api.post(`/api/stun/rules/${row.key}/restart`)
  await load()
}

async function remove(row: RuleView) {
  try {
    await api.del(`/api/stun/rules/${row.key}`)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function openSettings() {
  settings.value = await api.get<Settings>('/api/stun/settings')
  settingsVisible.value = true
}

async function saveSettings() {
  try {
    await api.put('/api/stun/settings', settings.value)
    settingsVisible.value = false
    ElMessage.success('已保存，规则已重启')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(async () => {
  interfaces.value = await api.get<NetInterface[]>('/api/stun/netinterfaces')
  await load()
  timer = window.setInterval(load, 3000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.intro {
  margin-bottom: 14px;
}
.tag {
  margin-left: 4px;
}
.addr {
  cursor: pointer;
  color: var(--el-color-primary);
}
.remark {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.error {
  font-size: 12px;
  color: var(--el-color-danger);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 260px;
}
.help {
  margin-left: 10px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.help.block {
  display: block;
  margin-left: 0;
}
.iface {
  float: right;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-left: 16px;
}
.logs {
  margin-top: 16px;
}
</style>
