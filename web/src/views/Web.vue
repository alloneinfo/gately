<template>
  <div>
    <div class="page-header">
      <h2>Web 服务</h2>
      <div>
        <el-button @click="openSettings">全局设置</el-button>
        <el-button type="primary" @click="openListener()">添加监听</el-button>
      </div>
    </div>

    <el-empty v-if="!loading && !listeners.length" description="暂无监听，可添加或在设置中导入 Lucky 配置" />

    <el-card v-for="l in listeners" :key="l.key" shadow="never" class="listener">
      <template #header>
        <div class="head">
          <div class="title">
            <el-switch :model-value="l.enable" size="small" @change="(v: string | number | boolean) => toggleListener(l, Boolean(v))" />
            <strong>{{ l.name || '未命名监听' }}</strong>
            <el-tag size="small" effect="plain" class="mono">{{ l.tls ? 'https' : 'http' }}://{{ l.listenIp || '*' }}:{{ l.port }}</el-tag>
            <el-tag size="small" type="info">{{ l.network }}</el-tag>
            <el-tag v-if="l.acceptProxyProtocol" size="small" type="info">接收 PROXY</el-tag>
            <el-tag v-if="l.error" type="danger" size="small" :title="l.error">{{ l.error }}</el-tag>
            <el-tag v-else-if="l.running" type="success" size="small">运行中</el-tag>
          </div>
          <div>
            <el-button size="small" type="primary" plain @click="openSite(l)">添加站点</el-button>
            <el-button size="small" @click="openListener(l)">编辑监听</el-button>
            <el-popconfirm title="确定删除该监听及其全部站点？" @confirm="removeListener(l)">
              <template #reference>
                <el-button size="small" type="danger" plain>删除</el-button>
              </template>
            </el-popconfirm>
          </div>
        </div>
      </template>
      <el-table :data="siteRows(l)" size="small">
        <el-table-column label="启用" width="60">
          <template #default="{ row }">
            <el-switch :model-value="row.site.enable" size="small" @change="(v: string | number | boolean) => toggleSite(l, row, Boolean(v))" />
          </template>
        </el-table-column>
        <el-table-column label="站点" min-width="200">
          <template #default="{ row }">
            <div v-if="row.isDefault"><el-tag size="small" type="warning">默认站点</el-tag> <span class="meta">未匹配的域名</span></div>
            <template v-else>
              <div class="mono">{{ row.site.domains.join(', ') }}</div>
              <div class="meta" v-if="row.site.name">{{ row.site.name }}</div>
            </template>
          </template>
        </el-table-column>
        <el-table-column label="类型 / 目标" min-width="240">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">{{ typeText(row.site.type) }}</el-tag>
            <span class="mono target">{{ summary(row.site) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="选项" min-width="180">
          <template #default="{ row }">
            <el-tag v-if="row.site.webAuth" size="small" type="warning" class="opt">网页认证</el-tag>
            <el-tag v-else-if="row.site.basicAuth" size="small" type="warning" class="opt">基本认证</el-tag>
            <el-tag v-if="row.site.ipMode" size="small" type="info" class="opt">{{ row.site.ipMode === 'whitelist' ? '白名单' : '黑名单' }}</el-tag>
            <el-tag v-if="row.site.sendProxyProtocol" size="small" type="info" class="opt">PROXY v2</el-tag>
            <el-tag v-if="row.site.insecureSkipVerify" size="small" type="info" class="opt">跳过证书校验</el-tag>
            <el-tag v-if="row.site.waf" size="small" :type="wafInstances.some((w) => w.key === row.site.waf) ? 'danger' : 'info'" class="opt">
              WAF{{ wafInstances.some((w) => w.key === row.site.waf) ? '' : '（实例不存在）' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="请求" width="110">
          <template #default="{ row }">
            <template v-if="l.stats[row.site.key]">
              <div>{{ l.stats[row.site.key].requests }}</div>
              <div class="meta" v-if="l.stats[row.site.key].errors">错误 {{ l.stats[row.site.key].errors }}</div>
            </template>
            <span v-else class="meta">-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="110">
          <template #default="{ row }">
            <el-button link type="primary" @click="openSite(l, row)">编辑</el-button>
            <el-popconfirm v-if="!row.isDefault" title="删除该站点？" @confirm="removeSite(l, row)">
              <template #reference>
                <el-button link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never" header="日志" class="logs">
      <LogViewer url="/api/webservice/logs" clearable />
    </el-card>

    <el-dialog v-model="listenerVisible" :title="editingListener.key ? '编辑监听' : '添加监听'" width="560px">
      <el-form :model="editingListener" label-width="130px">
        <el-form-item label="名称"><el-input v-model="editingListener.name" /></el-form-item>
        <el-form-item label="网络">
          <el-radio-group v-model="editingListener.network">
            <el-radio-button value="tcp">双栈</el-radio-button>
            <el-radio-button value="tcp4">仅 IPv4</el-radio-button>
            <el-radio-button value="tcp6">仅 IPv6</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="监听地址"><el-input v-model="editingListener.listenIp" placeholder="留空表示所有地址" /></el-form-item>
        <el-form-item label="端口" required><el-input-number v-model="editingListener.port" :min="1" :max="65535" /></el-form-item>
        <el-form-item label="HTTPS">
          <el-switch v-model="editingListener.tls" />
          <span class="help">证书按域名自动从「SSL 证书」中选择</span>
        </el-form-item>
        <el-form-item v-if="editingListener.tls" label="最低 TLS 版本">
          <el-select v-model="editingListener.minTls" style="width: 120px">
            <el-option v-for="v in ['1.0', '1.1', '1.2', '1.3']" :key="v" :label="'TLS ' + v" :value="v" />
          </el-select>
        </el-form-item>
        <el-form-item label="接收 PROXY 协议">
          <el-switch v-model="editingListener.acceptProxyProtocol" />
          <span class="help">仅信任全局设置中的来源</span>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="editingListener.remark" /></el-form-item>
        <el-form-item label="启用"><el-switch v-model="editingListener.enable" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="listenerVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveListener">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="siteVisible" :title="siteIsDefault ? '默认站点' : site.key ? '编辑站点' : '添加站点'" width="760px" top="5vh">
      <el-form :model="site" label-width="130px">
        <el-tabs v-model="siteTab">
          <el-tab-pane label="基本" name="basic">
            <el-form-item v-if="!siteIsDefault" label="域名" required>
              <el-input v-model="domainsText" type="textarea" :rows="2" class="mono" placeholder="每行一个；支持 *.example.com 与 *" />
            </el-form-item>
            <el-form-item label="名称"><el-input v-model="site.name" /></el-form-item>
            <el-form-item label="类型">
              <el-radio-group v-model="site.type">
                <el-radio-button value="proxy">反向代理</el-radio-button>
                <el-radio-button value="redirect">重定向</el-radio-button>
                <el-radio-button value="static">静态文件</el-radio-button>
                <el-radio-button value="text">固定文本</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <template v-if="site.type === 'proxy'">
              <el-form-item label="后端地址" required>
                <el-input v-model="targetsText" type="textarea" :rows="2" class="mono" placeholder="http://192.168.1.10:8080，多个时轮询" />
              </el-form-item>
              <el-form-item label="选项">
                <el-checkbox v-model="site.insecureSkipVerify">跳过后端证书校验</el-checkbox>
                <el-checkbox v-model="site.useTargetHost">Host 使用后端地址</el-checkbox>
                <el-checkbox v-model="site.disableKeepAlives">禁用连接复用</el-checkbox>
                <el-checkbox v-model="site.forwardedHeaders">添加 X-Forwarded-*</el-checkbox>
                <el-checkbox v-model="site.sendProxyProtocol">向后端发送 PROXY v2（传递真实 IP，需后端支持）</el-checkbox>
              </el-form-item>
            </template>
            <template v-if="site.type === 'redirect'">
              <el-form-item label="跳转到" required><el-input v-model="site.redirectTo" placeholder="https://example.com" /></el-form-item>
              <el-form-item label="状态码">
                <el-select v-model="site.redirectCode" style="width: 120px">
                  <el-option v-for="c in [301, 302, 307, 308]" :key="c" :label="c" :value="c" />
                </el-select>
                <el-checkbox v-model="site.keepPath" class="inline">保留路径与参数</el-checkbox>
              </el-form-item>
            </template>
            <template v-if="site.type === 'static'">
              <el-form-item label="目录" required><el-input v-model="site.root" placeholder="/data/www" /></el-form-item>
              <el-form-item label="目录列表"><el-switch v-model="site.listDir" /></el-form-item>
            </template>
            <template v-if="site.type === 'text'">
              <el-form-item label="状态码"><el-input-number v-model="site.statusCode" :min="100" :max="599" /></el-form-item>
              <el-form-item label="Content-Type"><el-input v-model="site.contentType" placeholder="text/plain; charset=utf-8" /></el-form-item>
              <el-form-item label="内容"><el-input v-model="site.text" type="textarea" :rows="4" /></el-form-item>
            </template>
            <el-form-item label="访问日志"><el-switch v-model="site.accessLog" /></el-form-item>
            <el-form-item label="启用"><el-switch v-model="site.enable" /></el-form-item>
          </el-tab-pane>
          <el-tab-pane label="安全" name="security">
            <el-form-item label="认证">
              <el-checkbox v-model="site.webAuth">网页认证（浏览器显示登录页）</el-checkbox>
              <el-checkbox v-model="site.basicAuth">基本认证</el-checkbox>
            </el-form-item>
            <el-form-item v-if="site.webAuth || site.basicAuth" label="用户">
              <el-input v-model="usersText" type="textarea" :rows="3" class="mono" placeholder="每行一个 用户名:密码（密码可为 bcrypt 哈希）" />
            </el-form-item>
            <el-form-item v-if="site.webAuth" label="第三方登录">
              <el-input v-model="oauthUsersText" type="textarea" :rows="2" class="mono" placeholder="每行一个允许的身份，如 github:octocat、google:me@gmail.com；github:* 表示任意 GitHub 账号" />
              <div class="help block">提供方需先在「全局设置」中配置</div>
            </el-form-item>
            <el-form-item label="IP 访问控制">
              <el-radio-group v-model="site.ipMode">
                <el-radio-button value="">关闭</el-radio-button>
                <el-radio-button value="whitelist">白名单</el-radio-button>
                <el-radio-button value="blacklist">黑名单</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item v-if="site.ipMode" label="IP / 网段">
              <el-input v-model="ipText" type="textarea" :rows="3" class="mono" placeholder="每行一个，如 192.168.1.0/24" />
            </el-form-item>
            <el-form-item label="WAF 实例">
              <el-select v-model="site.waf" clearable placeholder="不启用" style="width: 260px">
                <el-option v-for="w in wafInstances" :key="w.key" :label="w.name" :value="w.key" />
                <el-option v-if="site.waf && !wafInstances.some((w) => w.key === site.waf)" :label="`${site.waf}（实例不存在）`" :value="site.waf" />
              </el-select>
            </el-form-item>
            <el-form-item label="跨域 (CORS)"><el-switch v-model="site.cors" /></el-form-item>
          </el-tab-pane>
          <el-tab-pane label="请求头" name="headers">
            <el-form-item label="请求头">
              <el-input v-model="reqHeadersText" type="textarea" :rows="3" class="mono" placeholder="发往后端，每行 Key: Value；Value 留空表示删除" />
            </el-form-item>
            <el-form-item label="响应头">
              <el-input v-model="respHeadersText" type="textarea" :rows="3" class="mono" placeholder="返回给客户端，每行 Key: Value；Value 留空表示删除" />
            </el-form-item>
          </el-tab-pane>
        </el-tabs>
      </el-form>
      <template #footer>
        <el-button @click="siteVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveSite">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="settingsVisible" title="Web 服务全局设置" width="640px">
      <el-form v-if="settings" :model="settings" label-width="150px">
        <el-form-item label="启用模块"><el-switch v-model="settings.enable" /></el-form-item>
        <el-form-item label="可信 PROXY 来源">
          <el-input v-model="trustedText" type="textarea" :rows="4" class="mono" placeholder="每行一个 IP 或网段；留空表示不信任任何来源" />
        </el-form-item>
        <el-divider content-position="left">第三方登录（网页认证使用）</el-divider>
        <el-form-item label="回调地址">
          <el-input v-if="settings" v-model="settings.oauthCallbackBase" placeholder="https://auth.example.com（需由 Gately 的某个监听提供）" />
          <div class="help block">在 GitHub / Google 中登记的回调地址为：{{ (settings?.oauthCallbackBase || 'https://...').replace(/\/$/, '') }}/__gately_auth/oauth/callback</div>
        </el-form-item>
        <div v-for="(p, i) in settings?.oauthProviders ?? []" :key="i" class="provider">
          <div class="provider-row">
            <el-input v-model="p.key" placeholder="标识，如 github" style="width: 120px" />
            <el-input v-model="p.name" placeholder="显示名" style="width: 120px" />
            <el-select v-model="p.type" style="width: 120px">
              <el-option label="GitHub" value="github" />
              <el-option label="OIDC" value="oidc" />
            </el-select>
            <el-button link type="danger" @click="settings?.oauthProviders.splice(i, 1)">删除</el-button>
          </div>
          <div class="provider-row">
            <el-input v-model="p.clientId" placeholder="Client ID" />
            <el-input v-model="p.clientSecret" type="password" show-password placeholder="Client Secret" />
          </div>
          <el-input v-if="p.type === 'oidc'" v-model="p.issuer" placeholder="Issuer，如 https://accounts.google.com" />
        </div>
        <el-button size="small" @click="settings?.oauthProviders.push({ key: '', name: '', type: 'github', clientId: '', clientSecret: '', issuer: '' })">添加第三方登录</el-button>
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
import { api } from '../api'
import LogViewer from '../components/LogViewer.vue'

interface Site {
  key?: string
  name: string
  enable: boolean
  domains: string[]
  type: string
  targets: string[] | null
  insecureSkipVerify: boolean
  useTargetHost: boolean
  disableKeepAlives: boolean
  sendProxyProtocol: boolean
  forwardedHeaders: boolean
  redirectTo: string
  redirectCode: number
  keepPath: boolean
  root: string
  listDir: boolean
  text: string
  contentType: string
  statusCode: number
  basicAuth: boolean
  webAuth: boolean
  users: string[] | null
  oauthUsers: string[] | null
  ipMode: string
  ipList: string[] | null
  requestHeaders: string[] | null
  responseHeaders: string[] | null
  cors: boolean
  accessLog: boolean
  waf: string
  remark: string
}

interface Listener {
  key?: string
  name: string
  enable: boolean
  network: string
  listenIp: string
  port: number
  tls: boolean
  minTls: string
  acceptProxyProtocol: boolean
  sites: Site[] | null
  default: Site
  remark: string
}

interface ListenerView extends Listener {
  key: string
  running: boolean
  error?: string
  stats: Record<string, { requests: number; active: number; errors: number }>
}

interface SiteRow {
  site: Site
  index: number
  isDefault: boolean
}

const listeners = ref<ListenerView[]>([])
const wafInstances = ref<{ key: string; name: string }[]>([])
const loading = ref(true)
const saving = ref(false)
const listenerVisible = ref(false)
const editingListener = ref<Listener>(emptyListener())
const siteVisible = ref(false)
const siteTab = ref('basic')
const site = ref<Site>(emptySite())
const siteIsDefault = ref(false)
let siteOwner: ListenerView | null = null
let siteIndex = -1
const settingsVisible = ref(false)
interface OAuthProvider {
  key: string
  name: string
  type: string
  clientId: string
  clientSecret: string
  issuer: string
}
const settings = ref<{ enable: boolean; trustedProxies: string[] | null; oauthCallbackBase: string; oauthProviders: OAuthProvider[] }>()
let timer: number | undefined

const lines = (v: string) => v.split('\n').map((s) => s.trim()).filter(Boolean)
const bind = (get: () => string[] | null | undefined, set: (v: string[]) => void) =>
  computed({ get: () => (get() ?? []).join('\n'), set: (v: string) => set(lines(v)) })
const domainsText = bind(() => site.value.domains, (v) => (site.value.domains = v))
const targetsText = bind(() => site.value.targets, (v) => (site.value.targets = v))
const usersText = bind(() => site.value.users, (v) => (site.value.users = v))
const oauthUsersText = bind(() => site.value.oauthUsers, (v) => (site.value.oauthUsers = v))
const ipText = bind(() => site.value.ipList, (v) => (site.value.ipList = v))
const reqHeadersText = bind(() => site.value.requestHeaders, (v) => (site.value.requestHeaders = v))
const respHeadersText = bind(() => site.value.responseHeaders, (v) => (site.value.responseHeaders = v))
const trustedText = bind(() => settings.value?.trustedProxies, (v) => settings.value && (settings.value.trustedProxies = v))

function emptySite(): Site {
  return {
    name: '',
    enable: true,
    domains: [],
    type: 'proxy',
    targets: [],
    insecureSkipVerify: false,
    useTargetHost: false,
    disableKeepAlives: false,
    sendProxyProtocol: false,
    forwardedHeaders: true,
    redirectTo: '',
    redirectCode: 302,
    keepPath: true,
    root: '',
    listDir: false,
    text: '',
    contentType: '',
    statusCode: 200,
    basicAuth: false,
    webAuth: false,
    users: [],
    oauthUsers: [],
    ipMode: '',
    ipList: [],
    requestHeaders: [],
    responseHeaders: [],
    cors: false,
    accessLog: false,
    waf: '',
    remark: '',
  }
}

function emptyListener(): Listener {
  return {
    name: '',
    enable: true,
    network: 'tcp',
    listenIp: '',
    port: 443,
    tls: true,
    minTls: '1.2',
    acceptProxyProtocol: false,
    sites: [],
    default: { ...emptySite(), enable: false },
    remark: '',
  }
}

function typeText(t: string) {
  return ({ proxy: '反向代理', redirect: '重定向', static: '静态文件', text: '固定文本' } as Record<string, string>)[t] ?? t
}

function summary(s: Site) {
  switch (s.type) {
    case 'proxy':
      return (s.targets ?? []).join(' | ')
    case 'redirect':
      return `${s.redirectCode} → ${s.redirectTo}`
    case 'static':
      return s.root
    default:
      return `${s.statusCode} ${s.text.slice(0, 30)}`
  }
}

function siteRows(l: ListenerView): SiteRow[] {
  const rows = (l.sites ?? []).map((s, i) => ({ site: s, index: i, isDefault: false }))
  if (l.default.enable || (l.default.targets ?? []).length || l.default.type !== 'proxy') {
    rows.push({ site: l.default, index: -1, isDefault: true })
  }
  return rows
}

// 去掉运行状态字段，得到可提交的监听配置
function plain(l: ListenerView): Listener {
  const { running: _r, error: _e, stats: _s, ...rest } = l
  return JSON.parse(JSON.stringify(rest))
}

async function putListener(l: Listener) {
  const { key, ...body } = l
  await api.put(`/api/webservice/listeners/${key}`, body)
}

async function load() {
  try {
    listeners.value = await api.get<ListenerView[]>('/api/webservice/listeners')
  } finally {
    loading.value = false
  }
}

function openListener(l?: ListenerView) {
  editingListener.value = l ? plain(l) : emptyListener()
  listenerVisible.value = true
}

async function saveListener() {
  saving.value = true
  try {
    const { key, ...body } = editingListener.value
    if (key) {
      await api.put(`/api/webservice/listeners/${key}`, body)
    } else {
      await api.post('/api/webservice/listeners', body)
    }
    listenerVisible.value = false
    ElMessage.success('已保存')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function toggleListener(l: ListenerView, enable: boolean) {
  try {
    await api.put(`/api/webservice/listeners/${l.key}/enable`, { enable })
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function removeListener(l: ListenerView) {
  await api.del(`/api/webservice/listeners/${l.key}`)
  await load()
}

function openSite(l: ListenerView, row?: SiteRow) {
  siteOwner = l
  siteIndex = row ? row.index : -1
  siteIsDefault.value = !!row?.isDefault
  site.value = row ? JSON.parse(JSON.stringify(row.site)) : emptySite()
  siteTab.value = 'basic'
  siteVisible.value = true
}

async function saveSite() {
  if (!siteOwner) {
    return
  }
  const l = plain(siteOwner)
  l.sites = l.sites ?? []
  if (siteIsDefault.value) {
    l.default = site.value
  } else if (siteIndex >= 0) {
    l.sites[siteIndex] = site.value
  } else {
    l.sites.push(site.value)
  }
  saving.value = true
  try {
    await putListener(l)
    siteVisible.value = false
    ElMessage.success('已保存，监听已重新加载')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function toggleSite(lv: ListenerView, row: SiteRow, enable: boolean) {
  const l = plain(lv)
  if (row.isDefault) {
    l.default.enable = enable
  } else if (l.sites) {
    l.sites[row.index].enable = enable
  }
  try {
    await putListener(l)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function removeSite(lv: ListenerView, row: SiteRow) {
  const l = plain(lv)
  l.sites?.splice(row.index, 1)
  try {
    await putListener(l)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function openSettings() {
  const s = await api.get<{ enable: boolean; trustedProxies: string[] | null; oauthCallbackBase: string; oauthProviders: OAuthProvider[] | null }>('/api/webservice/settings')
  settings.value = { ...s, oauthProviders: s.oauthProviders ?? [] }
  settingsVisible.value = true
}

async function saveSettings() {
  try {
    await api.put('/api/webservice/settings', settings.value)
    settingsVisible.value = false
    ElMessage.success('已保存')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(async () => {
  wafInstances.value = await api.get('/api/waf/instances')
  await load()
  timer = window.setInterval(load, 5000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.listener {
  margin-bottom: 14px;
}
.head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.title {
  display: flex;
  align-items: center;
  gap: 8px;
}
.meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.target {
  margin-left: 6px;
  font-size: 12px;
}
.opt {
  margin: 2px 4px 2px 0;
}
.help {
  margin-left: 10px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.inline {
  margin-left: 16px;
}
.help.block {
  display: block;
  margin-left: 0;
}
.provider {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 8px;
  margin-bottom: 8px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.provider-row {
  display: flex;
  gap: 6px;
  align-items: center;
}
.logs {
  margin-top: 16px;
}
</style>
