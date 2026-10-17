<template>
  <div>
    <PageHeader title="frp 内网穿透" desc="用 frpc 把内网服务暴露到公网服务器，或在本机运行 frps 作为服务端" :icon="Waypoints">
      <el-button :icon="FileUp" @click="openImport">导入配置</el-button>
    </PageHeader>

    <el-tabs v-model="tab">
      <el-tab-pane label="客户端 frpc" name="client">
        <template v-if="client">
          <el-alert v-if="client.error && client.settings.enable" type="error" :title="client.error" :closable="false" show-icon class="block-gap" />
          <div class="g-card">
            <div class="g-card-head">
              <div class="g-card-title"><Plug :size="16" />连接 <span v-if="client.version" class="g-card-desc">{{ client.version }}</span></div>
              <div class="head-actions">
                <StatusBadge v-bind="roleBadge(client)" />
                <el-button type="primary" size="small" :icon="Save" :loading="savingClient" @click="saveClient">保存</el-button>
              </div>
            </div>
            <div class="g-card-body">
              <el-form label-width="110px" :label-position="labelPosition" class="svc-form">
                <el-form-item label="启用"><el-switch v-model="client.settings.enable" /></el-form-item>
                <el-form-item label="服务端地址"><el-input v-model="client.settings.serverAddr" placeholder="frps 的域名或 IP" /></el-form-item>
                <el-form-item label="服务端端口"><el-input-number v-model="client.settings.serverPort" :min="1" :max="65535" /></el-form-item>
                <el-form-item label="令牌"><el-input v-model="client.settings.token" type="password" show-password placeholder="需与 frps 的 token 一致" /></el-form-item>
                <el-form-item label="用户名"><el-input v-model="client.settings.user" placeholder="可选，用作代理名称前缀" /></el-form-item>
                <el-form-item label="传输协议">
                  <el-select v-model="client.settings.protocol" class="w-200">
                    <el-option label="tcp（默认）" value="" />
                    <el-option label="kcp" value="kcp" />
                    <el-option label="quic" value="quic" />
                    <el-option label="websocket" value="websocket" />
                    <el-option label="wss" value="wss" />
                  </el-select>
                </el-form-item>
                <el-form-item label="frpc 路径"><el-input v-model="client.settings.binPath" :placeholder="client.binary || '留空自动查找'" class="mono" /></el-form-item>
              </el-form>
            </div>
          </div>

          <div class="g-card">
            <div class="g-card-head">
              <div class="g-card-title"><ArrowRightLeft :size="16" />代理 <span class="g-card-desc">变更后自动重启 frpc</span></div>
              <el-button size="small" :icon="Plus" @click="openEditor()">添加代理</el-button>
            </div>
            <div class="g-card-body">
              <el-empty v-if="!proxies.length" description="暂无代理" :image-size="60" />
              <div class="card-grid">
                <EntityCard v-for="p in proxies" :key="p.key" :off="!p.enable">
                  <template #title>{{ p.name }}</template>
                  <template v-if="p.remark" #sub>{{ p.remark }}</template>
                  <template #status><el-tag size="small" type="info">{{ p.type }}</el-tag></template>
                  <Field label="本地" mono>{{ p.localIp }}:{{ p.localPort }}</Field>
                  <Field v-if="p.type === 'tcp' || p.type === 'udp'" label="远程端口" mono>{{ p.remotePort }}</Field>
                  <Field v-else-if="p.type === 'http' || p.type === 'https'" label="域名" mono>
                    <div v-for="d in p.customDomains" :key="d">{{ d }}</div>
                    <div v-if="p.subdomain">{{ p.subdomain }}.（子域名）</div>
                  </Field>
                  <template #actions>
                    <el-switch :model-value="p.enable" size="small" @change="(v: boolean | string | number) => toggle(p, Boolean(v))" />
                    <el-button text size="small" :icon="Pencil" @click="openEditor(p)">编辑</el-button>
                    <el-popconfirm title="删除该代理？" @confirm="remove(p)">
                      <template #reference>
                        <el-button text size="small" type="danger" :icon="Trash2">删除</el-button>
                      </template>
                    </el-popconfirm>
                  </template>
                </EntityCard>
              </div>
            </div>
          </div>
        </template>
      </el-tab-pane>

      <el-tab-pane label="服务端 frps" name="server">
        <template v-if="server">
          <el-alert v-if="server.error && server.settings.enable" type="error" :title="server.error" :closable="false" show-icon class="block-gap" />
          <div class="g-card">
            <div class="g-card-head">
              <div class="g-card-title"><Server :size="16" />服务 <span v-if="server.version" class="g-card-desc">{{ server.version }}</span></div>
              <div class="head-actions">
                <StatusBadge v-bind="roleBadge(server)" />
                <el-button type="primary" size="small" :icon="Save" :loading="savingServer" @click="saveServer">保存</el-button>
              </div>
            </div>
            <div class="g-card-body">
              <el-form label-width="130px" :label-position="labelPosition" class="svc-form">
                <el-form-item label="启用"><el-switch v-model="server.settings.enable" /></el-form-item>
                <el-form-item label="绑定端口"><el-input-number v-model="server.settings.bindPort" :min="1" :max="65535" /></el-form-item>
                <el-form-item label="令牌"><el-input v-model="server.settings.token" type="password" show-password placeholder="强烈建议设置，frpc 需使用相同令牌" /></el-form-item>
                <el-form-item label="HTTP 端口"><el-input-number v-model="server.settings.vhostHttpPort" :min="0" :max="65535" /><span class="help">0 表示不启用 http 代理</span></el-form-item>
                <el-form-item label="HTTPS 端口"><el-input-number v-model="server.settings.vhostHttpsPort" :min="0" :max="65535" /><span class="help">0 表示不启用 https 代理</span></el-form-item>
                <el-form-item label="子域名根域名"><el-input v-model="server.settings.subdomainHost" placeholder="如 frp.example.com" /></el-form-item>
                <el-form-item label="允许的远程端口">
                  <el-input v-model="server.settings.allowPorts" placeholder="如 6000-6100,7500，留空不限制" class="mono" />
                </el-form-item>
                <el-form-item label="控制台端口"><el-input-number v-model="server.settings.dashboardPort" :min="0" :max="65535" /><span class="help">0 表示关闭</span></el-form-item>
                <template v-if="server.settings.dashboardPort">
                  <el-form-item label="控制台监听"><el-input v-model="server.settings.dashboardAddr" placeholder="127.0.0.1" class="w-200" /></el-form-item>
                  <el-form-item label="控制台用户"><el-input v-model="server.settings.dashboardUser" class="w-200" /></el-form-item>
                  <el-form-item label="控制台密码"><el-input v-model="server.settings.dashboardPwd" type="password" show-password class="w-200" /></el-form-item>
                </template>
                <el-form-item label="frps 路径"><el-input v-model="server.settings.binPath" :placeholder="server.binary || '留空自动查找'" class="mono" /></el-form-item>
              </el-form>
              <div class="form-help">frps 监听的端口需要在防火墙与路由器中放行；Docker 部署时请映射对应端口。</div>
            </div>
          </div>
        </template>
      </el-tab-pane>
    </el-tabs>

    <LogCard url="/api/frp/logs" clearable />

    <el-dialog v-model="importVisible" :title="tab === 'client' ? '导入 frpc 配置' : '导入 frps 配置'" width="640px" :fullscreen="isMobile">
      <el-alert type="info" :closable="false" show-icon class="block-gap" title="支持旧版 frpc.ini / frps.ini（含 [common] 段）和 frp 0.52+ 的 toml。导入不会改变启用状态，确认无误后再手动启用。" />
      <div class="import-file">
        <el-button size="small" :icon="FileUp" @click="pickFile">选择文件</el-button>
        <span class="help">或直接把配置内容粘贴到下方</span>
        <input ref="fileInput" type="file" class="hidden-input" @change="onFile" />
      </div>
      <el-input v-model="importText" type="textarea" :rows="10" class="mono" placeholder="粘贴 frpc.ini / frpc.toml 的内容" @input="importPreview = undefined" />
      <el-radio-group v-if="tab === 'client'" v-model="proxyMode" class="import-mode" @change="importPreview = undefined">
        <el-radio value="append">追加（同名代理保持不变）</el-radio>
        <el-radio value="replace">替换全部现有代理</el-radio>
      </el-radio-group>
      <div v-if="importPreview" class="import-preview">
        <div class="preview-title">解析结果</div>
        <div v-if="importPreview.client" class="preview-line mono">
          {{ importPreview.client.serverAddr || '（未填写）' }}:{{ importPreview.client.serverPort }}
          <template v-if="importPreview.client.protocol"> · {{ importPreview.client.protocol }}</template>
          <template v-if="importPreview.client.token"> · 含令牌</template>
        </div>
        <div v-if="importPreview.server" class="preview-line mono">
          bind {{ importPreview.server.bindPort }}
          <template v-if="importPreview.server.vhostHttpPort"> · http {{ importPreview.server.vhostHttpPort }}</template>
          <template v-if="importPreview.server.vhostHttpsPort"> · https {{ importPreview.server.vhostHttpsPort }}</template>
          <template v-if="importPreview.server.subdomainHost"> · {{ importPreview.server.subdomainHost }}</template>
          <template v-if="importPreview.server.dashboardPort"> · 控制台 {{ importPreview.server.dashboardAddr }}:{{ importPreview.server.dashboardPort }}</template>
        </div>
        <div v-if="tab === 'client'" class="preview-line">
          共 {{ importPreview.proxies.length }} 个代理
          <div class="tags">
            <el-tag v-for="p in importPreview.proxies" :key="p.name" size="small" type="info">{{ p.name }} · {{ p.type }}</el-tag>
          </div>
        </div>
        <el-alert v-for="w in importPreview.warnings" :key="w" type="warning" :title="w" :closable="false" show-icon class="preview-warn" />
        <el-alert v-if="importPreview.applied" type="success" :closable="false" show-icon :title="importDoneText" class="preview-warn" />
      </div>
      <template #footer>
        <el-button @click="importVisible = false">关闭</el-button>
        <el-button :loading="importing" :disabled="!importText.trim()" @click="runImport(false)">解析预览</el-button>
        <el-button type="primary" :loading="importing" :disabled="!importPreview || importPreview.applied" @click="runImport(true)">导入</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="editorVisible" :title="editing.key ? '编辑代理' : '添加代理'" width="540px" :fullscreen="isMobile">
      <el-form :model="editing" label-width="100px" :label-position="labelPosition">
        <el-form-item label="名称"><el-input v-model="editing.name" /></el-form-item>
        <el-form-item label="类型">
          <el-select v-model="editing.type">
            <el-option label="tcp" value="tcp" />
            <el-option label="udp" value="udp" />
            <el-option label="http" value="http" />
            <el-option label="https" value="https" />
            <el-option label="stcp（私密 TCP）" value="stcp" />
            <el-option label="xtcp（点对点 TCP）" value="xtcp" />
          </el-select>
        </el-form-item>
        <el-form-item label="本地地址"><el-input v-model="editing.localIp" placeholder="127.0.0.1" /></el-form-item>
        <el-form-item label="本地端口"><el-input-number v-model="editing.localPort" :min="1" :max="65535" /></el-form-item>
        <el-form-item v-if="editing.type === 'tcp' || editing.type === 'udp'" label="远程端口">
          <el-input-number v-model="editing.remotePort" :min="1" :max="65535" />
        </el-form-item>
        <template v-if="editing.type === 'http' || editing.type === 'https'">
          <el-form-item label="自定义域名"><el-input v-model="domainsText" placeholder="多个用逗号分隔" /></el-form-item>
          <el-form-item label="子域名"><el-input v-model="editing.subdomain" placeholder="需 frps 配置了子域名根域名" /></el-form-item>
        </template>
        <el-form-item v-if="editing.type === 'stcp' || editing.type === 'xtcp'" label="密钥">
          <el-input v-model="editing.secretKey" type="password" show-password />
          <div class="form-help">访问端（visitor）需使用相同密钥</div>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="editing.remark" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">取消</el-button>
        <el-button type="primary" @click="saveProxy">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowRightLeft, FileUp, Pencil, Plug, Plus, Save, Server, Trash2, Waypoints } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import EntityCard from '../components/EntityCard.vue'
import Field from '../components/Field.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import { useFormLayout } from '../composables/useIsMobile'

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

const { isMobile, labelPosition } = useFormLayout()
const tab = ref('client')
const client = ref<RoleView<ClientSettings>>()
const server = ref<RoleView<ServerSettings>>()
const proxies = ref<Proxy[]>([])
const savingClient = ref(false)
const savingServer = ref(false)
const editorVisible = ref(false)
const editing = ref<Proxy>(empty())
let timer: number | undefined

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
const importDoneText = computed(() => (tab.value === 'client' ? `已导入，新增 ${importPreview.value?.added ?? 0} 个代理` : '已导入服务端配置'))

const domainsText = computed({
  get: () => (editing.value.customDomains ?? []).join(', '),
  set: (v: string) => (editing.value.customDomains = v.split(/[,\s]+/).filter(Boolean)),
})

function empty(): Proxy {
  return { name: '', type: 'tcp', enable: true, localIp: '127.0.0.1', localPort: 22, remotePort: 6000, customDomains: [], subdomain: '', secretKey: '', remark: '' }
}

function roleBadge(r: RoleView<{ enable: boolean }>): { type: StatusType; text: string; title?: string } {
  if (!r.settings.enable) {
    return { type: 'off', text: '已停用' }
  }
  if (r.error) {
    return { type: 'error', text: '启动失败', title: r.error }
  }
  return r.state.running ? { type: 'running', text: '运行中' } : { type: 'warn', text: '未运行' }
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
    ElMessage.success('已保存')
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
    ElMessage.success('已保存')
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
  timer = window.setInterval(refresh, 5000)
})
onBeforeUnmount(() => window.clearInterval(timer))
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
