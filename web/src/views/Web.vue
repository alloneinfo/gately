<template>
  <div>
    <PageHeader title="Web 服务" desc="按域名把请求反向代理、重定向或提供静态文件，可叠加认证与 WAF" :icon="Server">
      <el-button :icon="SlidersHorizontal" @click="openSettings">全局设置</el-button>
      <el-button type="primary" :icon="Plus" @click="openListener()">添加监听</el-button>
    </PageHeader>

    <ModuleBanner v-if="moduleOff" title="Web 服务模块已停用" desc="模块停用期间所有监听端口都会关闭；启用后各监听与站点按自身开关生效。" :loading="enabling" @enable="enableModule" />

    <div v-loading="loading" class="card-list">
      <el-empty v-if="!loading && !listeners.length" description="暂无监听，可添加或在「设置 → 备份与迁移」中导入 Lucky 配置" />
      <EntityCard v-for="l in listeners" :key="l.key" :off="!l.enable || moduleOff">
        <template #lead>
          <el-switch :model-value="l.enable" @change="(v: string | number | boolean) => toggleListener(l, Boolean(v))" />
        </template>
        <template #title>
          <span>{{ l.name || '未命名监听' }}</span>
          <span class="tags">
            <el-tag size="small" effect="plain" class="mono">{{ l.tls ? 'https' : 'http' }}://{{ l.listenIp || '*' }}:{{ l.port }}</el-tag>
            <el-tag size="small" type="info">{{ networkText(l.network) }}</el-tag>
            <el-tag v-if="l.acceptProxyProtocol" size="small" type="info">接收 PROXY</el-tag>
          </span>
        </template>
        <template v-if="l.remark" #sub>{{ l.remark }}</template>
        <template #status>
          <StatusBadge v-bind="listenerStatus(l)" />
        </template>
        <template #extra>
          <div class="sites">
            <div v-for="row in siteRows(l)" :key="row.isDefault ? '__default' : row.index" class="site" :class="{ 'is-off': !row.site.enable }">
              <div class="site-switch">
                <el-switch :model-value="row.site.enable" size="small" @change="(v: string | number | boolean) => toggleSite(l, row, Boolean(v))" />
              </div>
              <div class="site-main">
                <div v-if="row.isDefault" class="site-domains"><el-tag size="small" type="warning">默认站点</el-tag> <span class="meta">未匹配的域名</span></div>
                <div v-else class="site-domains mono">{{ row.site.domains.join(', ') }}</div>
                <div v-if="row.site.name" class="meta">{{ row.site.name }}</div>
              </div>
              <div class="site-target">
                <el-tag size="small" effect="plain">{{ typeText(row.site.type) }}</el-tag>
                <span class="mono target">{{ summary(row.site) }}</span>
              </div>
              <div class="site-opts">
                <StatusBadge v-if="!row.site.enable" type="off" text="已停用" />
                <el-tag v-if="row.site.webAuth" size="small" type="warning">网页认证</el-tag>
                <el-tag v-else-if="row.site.basicAuth" size="small" type="warning">基本认证</el-tag>
                <el-tag v-if="row.site.ipMode" size="small" type="info">{{ row.site.ipMode === 'whitelist' ? '白名单' : '黑名单' }}</el-tag>
                <el-tag v-if="row.site.sendProxyProtocol" size="small" type="info">PROXY v2</el-tag>
                <el-tag v-if="downCount(l, row) > 0" size="small" type="danger">{{ downCount(l, row) }} 个后端不可用</el-tag>
                <el-tag v-if="row.site.insecureSkipVerify" size="small" type="info">跳过证书校验</el-tag>
                <el-tag v-if="row.site.waf" size="small" :type="wafInstances.some((w) => w.key === row.site.waf) ? 'danger' : 'info'">
                  WAF{{ wafInstances.some((w) => w.key === row.site.waf) ? '' : '（实例不存在）' }}
                </el-tag>
              </div>
              <div class="site-stats">
                <template v-if="row.site.key && l.stats?.[row.site.key]">
                  <span>{{ l.stats[row.site.key].requests }} 请求</span>
                  <span v-if="l.stats[row.site.key].errors" class="text-danger"> · 错误 {{ l.stats[row.site.key].errors }}</span>
                </template>
                <span v-else class="muted">-</span>
              </div>
              <div class="site-actions">
                <el-button v-if="l.running" text size="small" :icon="BarChart3" @click="openStats(l, row)">统计</el-button>
                <el-button text size="small" :icon="Pencil" @click="openSite(l, row)">编辑</el-button>
                <el-popconfirm v-if="!row.isDefault" title="删除该站点？" @confirm="removeSite(l, row)">
                  <template #reference>
                    <el-button text size="small" type="danger" :icon="Trash2" aria-label="删除站点" />
                  </template>
                </el-popconfirm>
              </div>
            </div>
            <div v-if="!siteRows(l).length" class="meta empty-sites">还没有站点</div>
          </div>
        </template>
        <template #actions>
          <el-button text size="small" type="primary" :icon="Plus" @click="openSite(l)">添加站点</el-button>
          <el-button text size="small" :icon="Pencil" @click="openListener(l)">编辑监听</el-button>
          <el-popconfirm title="确定删除该监听及其全部站点？" width="240" @confirm="removeListener(l)">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Trash2">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </EntityCard>
    </div>

    <LogCard url="/api/webservice/logs" clearable />
    <SiteStats ref="statsDialog" />

    <el-dialog v-model="listenerVisible" :title="editingListener.key ? '编辑监听' : '添加监听'" width="560px" :fullscreen="isMobile">
      <el-form :model="editingListener" label-width="130px" :label-position="labelPosition">
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
          <el-select v-model="editingListener.minTls" class="w-120">
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

    <el-dialog v-model="siteVisible" :title="siteIsDefault ? '默认站点' : site.key ? '编辑站点' : '添加站点'" width="760px" top="5vh" :fullscreen="isMobile">
      <el-form :model="site" label-width="130px" :label-position="labelPosition">
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
                <el-select v-model="site.redirectCode" class="w-120">
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
              <el-input v-model="oauthUsersText" type="textarea" :rows="2" class="mono" placeholder="每行一个允许的身份：提供方标识:账号，如 github:octocat、google:me@gmail.com、qq:openid；行尾可用 # 加备注；github:* 表示任意 GitHub 账号" />
              <div class="form-help">提供方需先在「全局设置」中配置</div>
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
            <SiteAccess v-model:site="site" :tls="siteTls" :port="sitePort" />
            <el-form-item label="WAF 实例">
              <el-select v-model="site.waf" clearable placeholder="不启用" class="w-260">
                <el-option v-for="w in wafInstances" :key="w.key" :label="w.name" :value="w.key" />
                <el-option v-if="site.waf && !wafInstances.some((w) => w.key === site.waf)" :label="`${site.waf}（实例不存在）`" :value="site.waf" />
              </el-select>
            </el-form-item>
            <el-form-item label="跨域 (CORS)"><el-switch v-model="site.cors" /></el-form-item>
          </el-tab-pane>
          <el-tab-pane label="流量" name="traffic">
            <SiteTraffic v-model:site="site" />
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

    <el-dialog v-model="settingsVisible" title="Web 服务全局设置" width="640px" :fullscreen="isMobile">
      <el-form v-if="settings" :model="settings" label-width="150px" :label-position="labelPosition">
        <el-form-item label="启用模块"><el-switch v-model="settings.enable" /></el-form-item>
        <el-form-item label="可信 PROXY 来源">
          <el-input v-model="trustedText" type="textarea" :rows="4" class="mono" placeholder="每行一个 IP 或网段；留空表示不信任任何来源" />
        </el-form-item>
        <el-divider content-position="left">第三方登录（网页认证使用）</el-divider>
        <el-form-item label="回调地址">
          <el-input v-if="settings" v-model="settings.oauthCallbackBase" placeholder="https://auth.example.com（需由 Gately 的某个监听提供）" />
          <div class="form-help">在 GitHub / Google 中登记的回调地址为：{{ (settings?.oauthCallbackBase || 'https://...').replace(/\/$/, '') }}/__gately_auth/oauth/callback（GitHub、Google、QQ 均填此地址）</div>
        </el-form-item>
        <OAuthProviders v-if="settings" v-model="settings.oauthProviders" />
        <AccessSettings v-if="settings" v-model:auto-ban-seconds="settings.autoBanSeconds" />
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
import { BarChart3, Pencil, Plus, Server, SlidersHorizontal, Trash2 } from 'lucide-vue-next'
import { api, type OAuthProvider } from '../api'
import PageHeader from '../components/PageHeader.vue'
import ModuleBanner from '../components/ModuleBanner.vue'
import EntityCard from '../components/EntityCard.vue'
import OAuthProviders from '../components/OAuthProviders.vue'
import AccessSettings from '../components/AccessSettings.vue'
import SiteTraffic, { type Rewrite } from '../components/SiteTraffic.vue'
import SiteStats from '../components/SiteStats.vue'
import SiteAccess, { type PathRule, type ShareLink } from '../components/SiteAccess.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import { useFormLayout } from '../composables/useIsMobile'

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
  authOptional: boolean
  countryMode: string
  countries: string[] | null
  rules: PathRule[]
  shareLinks: ShareLink[]
  lbPolicy: string
  health: { enable: boolean; mode: string; path: string; interval: number; timeout: number; fails: number; passes: number }
  compress: boolean
  rateLimit: { enable: boolean; rate: number; burst: number; maxConns: number }
  cache: { enable: boolean; ttl: number; maxMb: number }
  rewrites: Rewrite[]
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
  upstreams?: Record<string, { url: string; healthy: boolean }[]>
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
// 分享链接地址需要所属监听器的协议与端口
const siteTls = ref(false)
const sitePort = ref(0)
let siteIndex = -1
const settingsVisible = ref(false)
const { isMobile, labelPosition } = useFormLayout()
// moduleEnabled 为模块全局开关，undefined 表示尚未加载
const moduleEnabled = ref<boolean>()
const moduleOff = computed(() => moduleEnabled.value === false)
const enabling = ref(false)
const settings = ref<{ enable: boolean; trustedProxies: string[] | null; oauthCallbackBase: string; oauthProviders: OAuthProvider[]; autoBanSeconds: number }>()
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
    authOptional: false,
    countryMode: '',
    countries: [],
    rules: [],
    shareLinks: [],
    lbPolicy: '',
    health: { enable: false, mode: 'http', path: '/', interval: 10, timeout: 3, fails: 3, passes: 2 },
    compress: false,
    rateLimit: { enable: false, rate: 10, burst: 20, maxConns: 0 },
    cache: { enable: false, ttl: 0, maxMb: 64 },
    rewrites: [],
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

function networkText(n: string) {
  return ({ tcp: '双栈', tcp4: '仅 IPv4', tcp6: '仅 IPv6' } as Record<string, string>)[n] ?? n
}

// listenerStatus 计算监听展示状态：停用的监听只显示“已停用”
function listenerStatus(l: ListenerView): { type: StatusType; text: string; title?: string } {
  if (moduleOff.value) {
    return { type: 'off', text: '模块已停用' }
  }
  if (!l.enable) {
    return { type: 'off', text: '已停用' }
  }
  if (l.error) {
    return { type: 'error', text: l.error, title: l.error }
  }
  if (l.running) {
    return { type: 'running', text: '运行中' }
  }
  return { type: 'warn', text: '未运行' }
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
  try {
    await api.del(`/api/webservice/listeners/${l.key}`)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

const statsDialog = ref<InstanceType<typeof SiteStats>>()

function openStats(l: ListenerView, row: SiteRow) {
  statsDialog.value?.open(l.key ?? '', row.isDefault ? 'default' : (row.site.key ?? ''), row.site.name || row.site.domains?.[0] || '默认站点')
}

// downCount 统计站点中不可用的后端数量
function downCount(l: ListenerView, row: SiteRow) {
  const k = row.isDefault ? 'default' : (row.site.key ?? '')
  return (l.upstreams?.[k] ?? []).filter((u) => !u.healthy).length
}

function openSite(l: ListenerView, row?: SiteRow) {
  siteOwner = l
  siteTls.value = l.tls
  sitePort.value = l.port
  siteIndex = row ? row.index : -1
  siteIsDefault.value = !!row?.isDefault
  // 旧配置中新增字段为 null，合并默认值后保证数组可直接编辑
  const draft: Site = row ? JSON.parse(JSON.stringify(row.site)) : emptySite()
  draft.rules = draft.rules ?? []
  draft.shareLinks = draft.shareLinks ?? []
  draft.countries = draft.countries ?? []
  draft.rewrites = draft.rewrites ?? []
  // 旧配置没有这些对象，补全默认值后可直接编辑
  const defaults = emptySite()
  draft.health = { ...defaults.health, ...(draft.health ?? {}) }
  draft.rateLimit = { ...defaults.rateLimit, ...(draft.rateLimit ?? {}) }
  draft.cache = { ...defaults.cache, ...(draft.cache ?? {}) }
  site.value = draft
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

async function fetchSettings() {
  const s = await api.get<{ enable: boolean; trustedProxies: string[] | null; oauthCallbackBase: string; oauthProviders: OAuthProvider[] | null; autoBanSeconds: number }>('/api/webservice/settings')
  moduleEnabled.value = s.enable
  return { ...s, oauthProviders: s.oauthProviders ?? [] }
}

async function openSettings() {
  settings.value = await fetchSettings()
  settingsVisible.value = true
}

// enableModule 在横幅中一键启用模块
async function enableModule() {
  enabling.value = true
  try {
    const s = await fetchSettings()
    await api.put('/api/webservice/settings', { ...s, enable: true })
    ElMessage.success('Web 服务模块已启用')
    await Promise.all([fetchSettings(), load()])
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    enabling.value = false
  }
}

async function saveSettings() {
  try {
    await api.put('/api/webservice/settings', settings.value)
    settingsVisible.value = false
    ElMessage.success('已保存')
    await Promise.all([fetchSettings(), load()])
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(async () => {
  fetchSettings()
  wafInstances.value = await api.get('/api/waf/instances')
  await load()
  timer = window.setInterval(load, 5000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.sites {
  border: 1px solid var(--g-border);
  border-radius: 8px;
  overflow: hidden;
}
.site {
  display: grid;
  grid-template-columns: 40px minmax(0, 1.5fr) minmax(0, 1.8fr) minmax(0, 1.3fr) 110px auto;
  gap: 8px 14px;
  align-items: center;
  padding: 9px 8px 9px 12px;
  font-size: 13px;
}
.site + .site {
  border-top: 1px solid var(--g-border);
}
.site.is-off .site-main,
.site.is-off .site-target,
.site.is-off .site-stats {
  opacity: 0.55;
}
.site-main,
.site-target {
  min-width: 0;
}
.site-domains {
  word-break: break-all;
}
.target {
  margin-left: 6px;
  font-size: 12px;
  word-break: break-all;
}
.site-opts {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.site-stats {
  font-size: 12px;
  white-space: nowrap;
}
.site-actions {
  display: flex;
  justify-content: flex-end;
}
.site-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}
.empty-sites {
  padding: 10px 12px;
}
.inline {
  margin-left: 16px;
}
.w-120 {
  width: 120px;
}
.w-260 {
  width: 260px;
}
.el-checkbox {
  white-space: normal;
  height: auto;
  min-height: 32px;
}
@media (max-width: 767.98px) {
  .site {
    grid-template-columns: auto minmax(0, 1fr) auto;
    padding: 10px;
  }
  .site-switch {
    grid-row: 1;
    grid-column: 1;
  }
  .site-main {
    grid-row: 1;
    grid-column: 2;
  }
  .site-actions {
    grid-row: 1;
    grid-column: 3;
  }
  .site-target,
  .site-opts,
  .site-stats {
    grid-column: 2 / 4;
  }
  .site-opts:empty {
    display: none;
  }
  .w-260 {
    width: 100%;
  }
  .inline {
    margin-left: 0;
  }
}
</style>
