<template>
  <div>
    <PageHeader :title="t('frp.title')" :desc="t('frp.desc')" :icon="Waypoints">
      <el-button :icon="FileUp" @click="openImport">{{ t('frp.importConfig') }}</el-button>
    </PageHeader>

    <el-tabs v-model="tab">
      <el-tab-pane :label="t('frp.clientTab')" name="client">
        <template v-if="client">
          <el-alert v-if="client.error && client.settings.enable" type="error" :title="client.error" :closable="false" show-icon class="block-gap" />
          <div class="g-card">
            <div class="g-card-head">
              <div class="g-card-title"><Plug :size="16" />{{ t('frp.connection') }} <span v-if="client.version" class="g-card-desc">{{ client.version }}</span></div>
              <div class="head-actions">
                <StatusBadge v-bind="roleBadge(client)" />
                <el-button type="primary" size="small" :icon="Save" :loading="savingClient" @click="saveClient">{{ t('common.save') }}</el-button>
              </div>
            </div>
            <div class="g-card-body">
              <el-form label-width="110px" :label-position="labelPosition" class="svc-form">
                <el-form-item :label="t('common.enable')"><el-switch v-model="client.settings.enable" /></el-form-item>
                <el-form-item :label="t('frp.serverAddr')"><el-input v-model="client.settings.serverAddr" :placeholder="t('frp.serverAddrPlaceholder')" /></el-form-item>
                <el-form-item :label="t('frp.serverPort')"><el-input-number v-model="client.settings.serverPort" :min="1" :max="65535" /></el-form-item>
                <el-form-item :label="t('frp.token')"><el-input v-model="client.settings.token" type="password" show-password :placeholder="t('frp.tokenClientPlaceholder')" /></el-form-item>
                <el-form-item :label="t('frp.user')"><el-input v-model="client.settings.user" :placeholder="t('frp.userPlaceholder')" /></el-form-item>
                <el-form-item :label="t('frp.protocol')">
                  <el-select v-model="client.settings.protocol" class="w-200">
                    <el-option :label="t('frp.tcpDefault')" value="" />
                    <el-option label="kcp" value="kcp" />
                    <el-option label="quic" value="quic" />
                    <el-option label="websocket" value="websocket" />
                    <el-option label="wss" value="wss" />
                  </el-select>
                </el-form-item>
                <el-form-item :label="t('frp.frpcPath')"><el-input v-model="client.settings.binPath" :placeholder="client.binary || t('frp.autoFind')" class="mono" /></el-form-item>
              </el-form>
            </div>
          </div>

          <div class="g-card">
            <div class="g-card-head">
              <div class="g-card-title"><ArrowRightLeft :size="16" />{{ t('frp.proxies') }} <span class="g-card-desc">{{ t('frp.proxiesDesc') }}</span></div>
              <el-button size="small" :icon="Plus" @click="openEditor()">{{ t('frp.addProxy') }}</el-button>
            </div>
            <div class="g-card-body">
              <el-empty v-if="!proxies.length" :description="t('frp.noProxies')" :image-size="60" />
              <div class="card-grid">
                <EntityCard v-for="p in proxies" :key="p.key" :off="!p.enable">
                  <template #title>{{ p.name }}</template>
                  <template v-if="p.remark" #sub>{{ p.remark }}</template>
                  <template #status><el-tag size="small" type="info">{{ p.type }}</el-tag></template>
                  <Field :label="t('frp.local')" mono>{{ p.localIp }}:{{ p.localPort }}</Field>
                  <Field v-if="p.type === 'tcp' || p.type === 'udp'" :label="t('frp.remotePort')" mono>{{ p.remotePort }}</Field>
                  <Field v-else-if="p.type === 'http' || p.type === 'https'" :label="t('frp.domain')" mono>
                    <div v-for="d in p.customDomains" :key="d">{{ d }}</div>
                    <div v-if="p.subdomain">{{ t('frp.subdomainSuffix', { sub: p.subdomain }) }}</div>
                  </Field>
                  <template #actions>
                    <el-switch :model-value="p.enable" size="small" @change="(v: boolean | string | number) => toggle(p, Boolean(v))" />
                    <el-button text size="small" :icon="Pencil" @click="openEditor(p)">{{ t('common.edit') }}</el-button>
                    <el-popconfirm :title="t('frp.confirmDeleteProxy')" @confirm="remove(p)">
                      <template #reference>
                        <el-button text size="small" type="danger" :icon="Trash2">{{ t('common.delete') }}</el-button>
                      </template>
                    </el-popconfirm>
                  </template>
                </EntityCard>
              </div>
            </div>
          </div>
        </template>
      </el-tab-pane>

      <el-tab-pane :label="t('frp.serverTab')" name="server">
        <template v-if="server">
          <el-alert v-if="server.error && server.settings.enable" type="error" :title="server.error" :closable="false" show-icon class="block-gap" />
          <div class="g-card">
            <div class="g-card-head">
              <div class="g-card-title"><Server :size="16" />{{ t('frp.service') }} <span v-if="server.version" class="g-card-desc">{{ server.version }}</span></div>
              <div class="head-actions">
                <StatusBadge v-bind="roleBadge(server)" />
                <el-button type="primary" size="small" :icon="Save" :loading="savingServer" @click="saveServer">{{ t('common.save') }}</el-button>
              </div>
            </div>
            <div class="g-card-body">
              <el-form label-width="130px" :label-position="labelPosition" class="svc-form">
                <el-form-item :label="t('common.enable')"><el-switch v-model="server.settings.enable" /></el-form-item>
                <el-form-item :label="t('frp.bindPort')"><el-input-number v-model="server.settings.bindPort" :min="1" :max="65535" /></el-form-item>
                <el-form-item :label="t('frp.token')"><el-input v-model="server.settings.token" type="password" show-password :placeholder="t('frp.tokenServerPlaceholder')" /></el-form-item>
                <el-form-item :label="t('frp.httpPort')"><el-input-number v-model="server.settings.vhostHttpPort" :min="0" :max="65535" /><span class="help">{{ t('frp.httpPortHelp') }}</span></el-form-item>
                <el-form-item :label="t('frp.httpsPort')"><el-input-number v-model="server.settings.vhostHttpsPort" :min="0" :max="65535" /><span class="help">{{ t('frp.httpsPortHelp') }}</span></el-form-item>
                <el-form-item :label="t('frp.subdomainHost')"><el-input v-model="server.settings.subdomainHost" :placeholder="t('frp.subdomainHostPlaceholder')" /></el-form-item>
                <el-form-item :label="t('frp.allowPorts')">
                  <el-input v-model="server.settings.allowPorts" :placeholder="t('frp.allowPortsPlaceholder')" class="mono" />
                </el-form-item>
                <el-form-item :label="t('frp.dashboardPort')"><el-input-number v-model="server.settings.dashboardPort" :min="0" :max="65535" /><span class="help">{{ t('frp.dashboardPortHelp') }}</span></el-form-item>
                <template v-if="server.settings.dashboardPort">
                  <el-form-item :label="t('frp.dashboardAddr')"><el-input v-model="server.settings.dashboardAddr" placeholder="127.0.0.1" class="w-200" /></el-form-item>
                  <el-form-item :label="t('frp.dashboardUser')"><el-input v-model="server.settings.dashboardUser" class="w-200" /></el-form-item>
                  <el-form-item :label="t('frp.dashboardPwd')"><el-input v-model="server.settings.dashboardPwd" type="password" show-password class="w-200" /></el-form-item>
                </template>
                <el-form-item :label="t('frp.frpsPath')"><el-input v-model="server.settings.binPath" :placeholder="server.binary || t('frp.autoFind')" class="mono" /></el-form-item>
              </el-form>
              <div class="form-help">{{ t('frp.firewallHelp') }}</div>
            </div>
          </div>
        </template>
      </el-tab-pane>
    </el-tabs>

    <LogCard url="/api/frp/logs" clearable />

    <el-dialog v-model="importVisible" :title="tab === 'client' ? t('frp.importFrpc') : t('frp.importFrps')" width="640px" :fullscreen="isMobile">
      <el-alert type="info" :closable="false" show-icon class="block-gap" :title="t('frp.importTip')" />
      <div class="import-file">
        <el-button size="small" :icon="FileUp" @click="pickFile">{{ t('frp.pickFile') }}</el-button>
        <span class="help">{{ t('frp.pasteHint') }}</span>
        <input ref="fileInput" type="file" class="hidden-input" @change="onFile" />
      </div>
      <el-input v-model="importText" type="textarea" :rows="10" class="mono" :placeholder="t('frp.pastePlaceholder')" @input="importPreview = undefined" />
      <el-radio-group v-if="tab === 'client'" v-model="proxyMode" class="import-mode" @change="importPreview = undefined">
        <el-radio value="append">{{ t('frp.modeAppend') }}</el-radio>
        <el-radio value="replace">{{ t('frp.modeReplace') }}</el-radio>
      </el-radio-group>
      <div v-if="importPreview" class="import-preview">
        <div class="preview-title">{{ t('frp.parseResult') }}</div>
        <div v-if="importPreview.client" class="preview-line mono">
          {{ importPreview.client.serverAddr || t('frp.notFilled') }}:{{ importPreview.client.serverPort }}
          <template v-if="importPreview.client.protocol"> · {{ importPreview.client.protocol }}</template>
          <template v-if="importPreview.client.token"> · {{ t('frp.hasToken') }}</template>
        </div>
        <div v-if="importPreview.server" class="preview-line mono">
          bind {{ importPreview.server.bindPort }}
          <template v-if="importPreview.server.vhostHttpPort"> · http {{ importPreview.server.vhostHttpPort }}</template>
          <template v-if="importPreview.server.vhostHttpsPort"> · https {{ importPreview.server.vhostHttpsPort }}</template>
          <template v-if="importPreview.server.subdomainHost"> · {{ importPreview.server.subdomainHost }}</template>
          <template v-if="importPreview.server.dashboardPort"> · {{ t('frp.dashboard') }} {{ importPreview.server.dashboardAddr }}:{{ importPreview.server.dashboardPort }}</template>
        </div>
        <div v-if="tab === 'client'" class="preview-line">
          {{ t('frp.proxyCount', { count: importPreview.proxies.length }) }}
          <div class="tags">
            <el-tag v-for="p in importPreview.proxies" :key="p.name" size="small" type="info">{{ p.name }} · {{ p.type }}</el-tag>
          </div>
        </div>
        <el-alert v-for="w in importPreview.warnings" :key="w" type="warning" :title="w" :closable="false" show-icon class="preview-warn" />
        <el-alert v-if="importPreview.applied" type="success" :closable="false" show-icon :title="importDoneText" class="preview-warn" />
      </div>
      <template #footer>
        <el-button @click="importVisible = false">{{ t('common.close') }}</el-button>
        <el-button :loading="importing" :disabled="!importText.trim()" @click="runImport(false)">{{ t('frp.parsePreview') }}</el-button>
        <el-button type="primary" :loading="importing" :disabled="!importPreview || importPreview.applied" @click="runImport(true)">{{ t('common.import') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="editorVisible" :title="editing.key ? t('frp.editProxy') : t('frp.addProxy')" width="540px" :fullscreen="isMobile">
      <el-form :model="editing" label-width="100px" :label-position="labelPosition">
        <el-form-item :label="t('common.name')"><el-input v-model="editing.name" /></el-form-item>
        <el-form-item :label="t('frp.type')">
          <el-select v-model="editing.type">
            <el-option label="tcp" value="tcp" />
            <el-option label="udp" value="udp" />
            <el-option label="http" value="http" />
            <el-option label="https" value="https" />
            <el-option :label="t('frp.stcpLabel')" value="stcp" />
            <el-option :label="t('frp.xtcpLabel')" value="xtcp" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('frp.localAddr')"><el-input v-model="editing.localIp" placeholder="127.0.0.1" /></el-form-item>
        <el-form-item :label="t('frp.localPort')"><el-input-number v-model="editing.localPort" :min="1" :max="65535" /></el-form-item>
        <el-form-item v-if="editing.type === 'tcp' || editing.type === 'udp'" :label="t('frp.remotePort')">
          <el-input-number v-model="editing.remotePort" :min="1" :max="65535" />
        </el-form-item>
        <template v-if="editing.type === 'http' || editing.type === 'https'">
          <el-form-item :label="t('frp.customDomains')"><el-input v-model="domainsText" :placeholder="t('frp.domainsPlaceholder')" /></el-form-item>
          <el-form-item :label="t('frp.subdomain')"><el-input v-model="editing.subdomain" :placeholder="t('frp.subdomainPlaceholder')" /></el-form-item>
        </template>
        <el-form-item v-if="editing.type === 'stcp' || editing.type === 'xtcp'" :label="t('frp.secretKey')">
          <el-input v-model="editing.secretKey" type="password" show-password />
          <div class="form-help">{{ t('frp.secretHelp') }}</div>
        </el-form-item>
        <el-form-item :label="t('common.remark')"><el-input v-model="editing.remark" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="saveProxy">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { ArrowRightLeft, FileUp, Pencil, Plug, Plus, Save, Server, Trash2, Waypoints } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import EntityCard from '../components/EntityCard.vue'
import Field from '../components/Field.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import { useFormLayout } from '../composables/useIsMobile'
import { useListText, splitCommasOrSpaces } from '../composables/useListText'
import { usePolling } from '../composables/usePolling'

interface ClientSettings {
  enable: boolean
  binPath: string
  serverAddr: string
  serverPort: number
  token: string
  user: string
  protocol: string
}
interface ServerSettings {
  enable: boolean
  binPath: string
  bindPort: number
  token: string
  vhostHttpPort: number
  vhostHttpsPort: number
  subdomainHost: string
  allowPorts: string
  dashboardPort: number
  dashboardAddr: string
  dashboardUser: string
  dashboardPwd: string
}
interface RoleView<T> {
  settings: T
  state: { running: boolean; error?: string }
  binary: string
  version: string
  error: string
}
interface Proxy {
  key?: string
  name: string
  type: string
  enable: boolean
  localIp: string
  localPort: number
  remotePort: number
  customDomains: string[] | null
  subdomain: string
  secretKey: string
  remark: string
}

const { t } = useI18n()
const { isMobile, labelPosition } = useFormLayout()
const tab = ref('client')
const client = ref<RoleView<ClientSettings>>()
const server = ref<RoleView<ServerSettings>>()
const proxies = ref<Proxy[]>([])
const savingClient = ref(false)
const savingServer = ref(false)
const editorVisible = ref(false)
const editing = ref<Proxy>(empty())
// 定时刷新，标签页在后台时暂停
const polling = usePolling(refresh, 5000)

interface ImportResult {
  applied: boolean
  role: string
  client?: ClientSettings
  server?: ServerSettings
  proxies: Proxy[]
  warnings: string[]
  added: number
}
const importVisible = ref(false)
const importText = ref('')
const proxyMode = ref('append')
const importing = ref(false)
const importPreview = ref<ImportResult>()
const fileInput = ref<HTMLInputElement>()
const importDoneText = computed(() => (tab.value === 'client' ? t('frp.importedClient', { count: importPreview.value?.added ?? 0 }) : t('frp.importedServer')))

// 输入框保留原始文本，逗号、空格可正常输入（见 useListText）
const domainsText = useListText(() => editing.value.customDomains, (v) => (editing.value.customDomains = v), {
  split: splitCommasOrSpaces,
  separator: ', ',
})

function empty(): Proxy {
  return { name: '', type: 'tcp', enable: true, localIp: '127.0.0.1', localPort: 22, remotePort: 6000, customDomains: [], subdomain: '', secretKey: '', remark: '' }
}

function roleBadge(r: RoleView<{ enable: boolean }>): { type: StatusType; text: string; title?: string } {
  if (!r.settings.enable) {
    return { type: 'off', text: t('frp.disabled') }
  }
  if (r.error) {
    return { type: 'error', text: t('frp.startFailed'), title: r.error }
  }
  return r.state.running ? { type: 'running', text: t('common.running') } : { type: 'warn', text: t('frp.notRunning') }
}

// refresh 仅刷新运行状态，不覆盖用户正在编辑的表单
async function refresh() {
  try {
    const [c, s] = await Promise.all([api.get<RoleView<ClientSettings>>('/api/frp/client'), api.get<RoleView<ServerSettings>>('/api/frp/server')])
    if (client.value) {
      Object.assign(client.value, { state: c.state, binary: c.binary, version: c.version, error: c.error })
    }
    if (server.value) {
      Object.assign(server.value, { state: s.state, binary: s.binary, version: s.version, error: s.error })
    }
  } catch {
    // 轮询失败时保留上一次的状态
  }
}

async function load() {
  const [c, s] = await Promise.all([api.get<RoleView<ClientSettings>>('/api/frp/client'), api.get<RoleView<ServerSettings>>('/api/frp/server')])
  client.value = c
  server.value = s
  proxies.value = await api.get<Proxy[]>('/api/frp/proxies')
}

async function saveClient() {
  savingClient.value = true
  try {
    await api.put('/api/frp/client', client.value?.settings)
    ElMessage.success(t('common.saved'))
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    savingClient.value = false
  }
}

async function saveServer() {
  savingServer.value = true
  try {
    await api.put('/api/frp/server', server.value?.settings)
    ElMessage.success(t('common.saved'))
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    savingServer.value = false
  }
}

function openImport() {
  importText.value = ''
  importPreview.value = undefined
  importVisible.value = true
}

function pickFile() {
  fileInput.value?.click()
}

// onFile 读取所选配置文件并填入文本框
async function onFile(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) {
    importText.value = await file.text()
    importPreview.value = undefined
  }
  input.value = ''
}

// runImport 先预览再导入；apply 为 true 时保存并刷新页面数据
async function runImport(apply: boolean) {
  importing.value = true
  try {
    importPreview.value = await api.post<ImportResult>('/api/frp/import', {
      role: tab.value,
      content: importText.value,
      apply,
      proxyMode: proxyMode.value,
    })
    if (apply) {
      await load()
    }
  } catch (e) {
    importPreview.value = undefined
    ElMessage.error((e as Error).message)
  } finally {
    importing.value = false
  }
}

function openEditor(p?: Proxy) {
  editing.value = p ? JSON.parse(JSON.stringify(p)) : empty()
  editorVisible.value = true
}

async function saveProxy() {
  try {
    const { key, ...body } = editing.value
    if (key) {
      await api.put(`/api/frp/proxies/${key}`, body)
    } else {
      await api.post('/api/frp/proxies', body)
    }
    editorVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function toggle(p: Proxy, enable: boolean) {
  try {
    await api.put(`/api/frp/proxies/${p.key}/enable`, { enable })
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function remove(p: Proxy) {
  try {
    await api.del(`/api/frp/proxies/${p.key}`)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(() => {
  load()
  polling.start()
})
</script>

<style scoped>
.block-gap {
  margin-bottom: 16px;
}
.head-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}
.svc-form {
  max-width: 640px;
}
.svc-form .el-form-item:last-child {
  margin-bottom: 0;
}
.import-file {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}
.hidden-input {
  display: none;
}
.import-mode {
  margin-top: 10px;
}
.import-preview {
  margin-top: 14px;
}
.preview-title {
  font-weight: 500;
  margin-bottom: 6px;
}
.preview-line {
  margin-bottom: 8px;
}
.tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 6px;
}
.preview-warn {
  margin-top: 6px;
}
.w-200 {
  width: 200px;
}
</style>
