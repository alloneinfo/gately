<template>
  <div>
    <PageHeader title="SSL 证书" desc="通过 ACME 自动申请与续签证书，也可上传或读取本地证书文件" :icon="ShieldCheck">
      <el-button :icon="SlidersHorizontal" @click="openSettings">续签设置</el-button>
      <el-button type="primary" :icon="Plus" @click="openEditor()">添加证书</el-button>
    </PageHeader>

    <ModuleBanner
      v-if="renewSettings && !renewSettings.enable"
      title="自动续签已关闭"
      desc="ACME 证书不会在到期前自动续签，可在卡片中手动续签。"
      action="开启自动续签"
      :loading="enabling"
      @enable="enableRenew"
    />

    <div v-loading="loading" class="card-list">
      <el-empty v-if="!loading && !certs.length" description="暂无证书，可申请、上传或在「设置 → 备份与迁移」中导入 Lucky 配置" />
      <EntityCard v-for="c in certs" :key="c.key" :off="!c.enable">
        <template #lead>
          <el-switch :model-value="c.enable" @change="(v: string | number | boolean) => toggle(c, Boolean(v))" />
        </template>
        <template #title>
          <span>{{ c.remark || '未命名' }}</span>
          <el-tag size="small" effect="plain">{{ sourceText(c.source) }}</el-tag>
        </template>
        <template #status>
          <StatusBadge v-bind="certStatus(c)" />
        </template>
        <Field label="域名" mono wide>{{ domainsOf(c).join(', ') || '-' }}</Field>
        <Field label="有效期至">{{ c.info ? new Date(c.info.notAfter).toLocaleDateString() : '-' }}</Field>
        <Field label="颁发者">{{ c.info?.issuer || '-' }}</Field>
        <Field label="更新时间">{{ c.updatedAt ? new Date(c.updatedAt).toLocaleString() : '-' }}</Field>
        <template v-if="c.enable && (c.state.lastError || c.state.mappingNote || c.state.nextRetry)" #extra>
          <div v-if="c.state.lastError" class="err-line">{{ c.state.lastError }}</div>
          <div v-if="c.state.nextRetry" class="meta">下次重试 {{ new Date(c.state.nextRetry).toLocaleString() }}</div>
          <div v-if="c.state.mappingNote" class="meta">{{ c.state.mappingNote }}</div>
        </template>
        <template #actions>
          <el-button v-if="c.source === 'acme'" text size="small" :icon="RotateCw" :disabled="c.state.obtaining || !c.enable" @click="renew(c)">续签</el-button>
          <el-dropdown v-if="c.info" trigger="click" @command="(cmd: string) => download(c, cmd)">
            <el-button text size="small" :icon="Download">下载</el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="crt">证书 (.crt)</el-dropdown-item>
                <el-dropdown-item command="key">私钥 (.key)</el-dropdown-item>
                <el-dropdown-item command="issuer" :disabled="!c.issuerPem">颁发者证书</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <el-button text size="small" :icon="Pencil" @click="openEditor(c)">编辑</el-button>
          <el-popconfirm title="确定删除该证书？已映射的文件不会删除" width="240" @confirm="remove(c)">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Trash2">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </EntityCard>
    </div>

    <LogCard url="/api/ssl/logs" clearable />

    <el-dialog v-model="editorVisible" :title="editing.key ? '编辑证书' : '添加证书'" width="760px" top="5vh" :fullscreen="isMobile">
      <el-form :model="editing" label-width="130px" :label-position="labelPosition">
        <el-form-item label="备注">
          <el-input v-model="editing.remark" placeholder="启用映射时作为文件名，例如 nas.example.com" />
        </el-form-item>
        <el-form-item label="来源">
          <el-radio-group v-model="editing.source" :disabled="!!editing.key">
            <el-radio-button value="acme">ACME 自动申请</el-radio-button>
            <el-radio-button value="upload">上传证书</el-radio-button>
            <el-radio-button value="file">读取文件</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <template v-if="editing.source === 'acme'">
          <el-form-item label="域名" required>
            <el-input v-model="domainsText" type="textarea" :rows="2" placeholder="每行一个，支持通配符 *.example.com" class="mono" />
          </el-form-item>
          <el-form-item label="账户邮箱"><el-input v-model="editing.acme.email" placeholder="留空使用续签设置中的默认邮箱" /></el-form-item>
          <el-form-item label="证书颁发机构">
            <el-select v-model="editing.acme.ca" class="w-260">
              <el-option label="Let's Encrypt" value="letsencrypt" />
              <el-option label="Let's Encrypt（测试环境）" value="letsencrypt-staging" />
              <el-option label="ZeroSSL" value="zerossl" />
              <el-option label="Google Trust Services" value="google" />
              <el-option label="自定义" value="custom" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="editing.acme.ca === 'custom'" label="目录地址">
            <el-input v-model="editing.acme.directoryUrl" placeholder="https://.../directory" />
          </el-form-item>
          <template v-if="['zerossl', 'google', 'custom'].includes(editing.acme.ca)">
            <el-form-item label="EAB Key ID"><el-input v-model="editing.acme.eabKid" /></el-form-item>
            <el-form-item label="EAB HMAC Key"><el-input v-model="editing.acme.eabHmac" type="password" show-password /></el-form-item>
          </template>
          <el-form-item label="密钥类型">
            <el-select v-model="editing.acme.keyType" class="w-160">
              <el-option label="ECC P-256" value="P256" />
              <el-option label="ECC P-384" value="P384" />
              <el-option label="RSA 2048" value="2048" />
              <el-option label="RSA 4096" value="4096" />
            </el-select>
          </el-form-item>
          <el-divider content-position="left">DNS 验证</el-divider>
          <el-form-item label="DNS 服务商">
            <el-select v-model="editing.acme.provider" class="w-260" @change="extra = {}">
              <el-option v-for="p in providers" :key="p.name" :label="p.title" :value="p.name" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="currentProvider?.idLabel" :label="currentProvider.idLabel">
            <el-input v-model="editing.acme.credentials.id" />
          </el-form-item>
          <el-form-item v-if="currentProvider?.secretLabel" :label="currentProvider.secretLabel">
            <el-input v-model="editing.acme.credentials.secret" type="password" show-password placeholder="已保存的密钥显示为 ********" />
          </el-form-item>
          <el-form-item v-for="f in currentProvider?.extra ?? []" :key="f.key" :label="f.label">
            <el-input v-model="extra[f.key]" :placeholder="f.placeholder" />
          </el-form-item>
          <el-form-item label="主域名"><el-input v-model="editing.acme.zone" placeholder="留空自动推断；多级后缀推断错误时填写，例如 example.com.cn" /></el-form-item>
          <el-form-item label="检查用 DNS"><el-input v-model="resolversText" placeholder="可选，例如 223.5.5.5:53, 1.1.1.1:53" /></el-form-item>
          <el-form-item label="生效等待(秒)"><el-input-number v-model="editing.acme.propagationTimeout" :min="30" :max="3600" /></el-form-item>
          <el-form-item label="生效检查">
            <el-checkbox v-model="editing.acme.disableAuthoritativeCheck">只查询上面的 DNS，不查询权威服务器</el-checkbox>
            <el-checkbox v-model="editing.acme.skipPropagationCheck">跳过检查（等待 20 秒后直接验证）</el-checkbox>
          </el-form-item>
        </template>

        <template v-if="editing.source === 'upload'">
          <el-form-item label="证书 (PEM)" required>
            <el-input v-model="editing.certPem" type="textarea" :rows="5" class="mono" placeholder="-----BEGIN CERTIFICATE-----，包含中间证书" />
            <label class="file-pick"><Upload :size="14" />从文件读取<input type="file" accept=".pem,.crt,.cer" @change="(e) => readFile(e, 'certPem')" /></label>
          </el-form-item>
          <el-form-item label="私钥 (PEM)" required>
            <el-input v-model="editing.keyPem" type="textarea" :rows="4" class="mono" placeholder="已保存的私钥显示为 ********，不修改则保持原值" />
            <label class="file-pick"><Upload :size="14" />从文件读取<input type="file" accept=".pem,.key" @change="(e) => readFile(e, 'keyPem')" /></label>
          </el-form-item>
        </template>

        <template v-if="editing.source === 'file'">
          <el-form-item label="证书路径" required><el-input v-model="editing.certFile" placeholder="/etc/ssl/example/fullchain.pem" /></el-form-item>
          <el-form-item label="私钥路径" required><el-input v-model="editing.keyFile" placeholder="/etc/ssl/example/privkey.pem" /></el-form-item>
          <el-form-item label=" "><div class="form-help">定期检查文件变化并自动加载，适合由其他工具（如 certbot）维护的证书。</div></el-form-item>
        </template>

        <el-divider content-position="left">证书映射</el-divider>
        <el-form-item label="写入目录"><el-switch v-model="editing.mapping.enable" /></el-form-item>
        <template v-if="editing.mapping.enable">
          <el-form-item label="目录"><el-input v-model="editing.mapping.dir" placeholder="绝对路径，将写入 备注.crt / .key / .pem / _issuerCertificate.crt" /></el-form-item>
          <el-form-item label="变化后执行">
            <el-input v-model="editing.mapping.script" type="textarea" :rows="3" class="mono" placeholder="例如 docker exec nginx nginx -s reload" />
            <div class="form-help">环境变量：certName certDir certFile keyFile</div>
          </el-form-item>
        </template>
        <el-form-item label="启用"><el-switch v-model="editing.enable" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="settingsVisible" title="续签设置" width="520px" :fullscreen="isMobile">
      <el-form v-if="settings" :model="settings" label-width="170px" :label-position="labelPosition">
        <el-form-item label="自动续签"><el-switch v-model="settings.enable" /></el-form-item>
        <el-form-item label="到期前多少天续签"><el-input-number v-model="settings.renewBeforeDays" :min="1" :max="60" /></el-form-item>
        <el-form-item label="短期证书提前小时数"><el-input-number v-model="settings.shortLivedRenewHours" :min="1" :max="120" /></el-form-item>
        <el-form-item label="默认账户邮箱"><el-input v-model="settings.defaultEmail" /></el-form-item>
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
import { Download, Pencil, Plus, RotateCw, ShieldCheck, SlidersHorizontal, Trash2, Upload } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import ModuleBanner from '../components/ModuleBanner.vue'
import EntityCard from '../components/EntityCard.vue'
import Field from '../components/Field.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import { useFormLayout } from '../composables/useIsMobile'

interface ProviderMeta {
  name: string
  title: string
  idLabel?: string
  secretLabel?: string
  extra?: { key: string; label: string; placeholder?: string }[]
}

interface Cert {
  key?: string
  remark: string
  enable: boolean
  source: string
  acme: {
    domains: string[] | null
    email: string
    ca: string
    directoryUrl: string
    eabKid: string
    eabHmac: string
    keyType: string
    provider: string
    credentials: { id: string; secret: string; extra?: Record<string, string> }
    zone: string
    resolvers: string[] | null
    propagationTimeout: number
    skipPropagationCheck: boolean
    disableAuthoritativeCheck: boolean
  }
  certFile: string
  keyFile: string
  certPem: string
  keyPem: string
  issuerPem: string
  mapping: { enable: boolean; dir: string; script: string }
  updatedAt?: string
}

interface CertView extends Cert {
  key: string
  info?: { domains: string[]; issuer: string; notBefore: string; notAfter: string }
  state: { obtaining: boolean; lastError?: string; nextRetry?: string; mappingNote?: string }
}

const certs = ref<CertView[]>([])
const providers = ref<ProviderMeta[]>([])
const loading = ref(true)
const editorVisible = ref(false)
const saving = ref(false)
const editing = ref<Cert>(emptyCert())
const extra = ref<Record<string, string>>({})
const settingsVisible = ref(false)
interface RenewSettings {
  enable: boolean
  renewBeforeDays: number
  shortLivedRenewHours: number
  defaultEmail: string
}
const settings = ref<RenewSettings>()
// renewSettings 为当前生效的续签设置，用于显示横幅
const renewSettings = ref<RenewSettings>()
const enabling = ref(false)
const { isMobile, labelPosition } = useFormLayout()
let timer: number | undefined

const currentProvider = computed(() => providers.value.find((p) => p.name === editing.value.acme.provider))
const splitList = (v: string) => v.split(/[\n,]/).map((s) => s.trim()).filter(Boolean)
const domainsText = computed({
  get: () => (editing.value.acme.domains ?? []).join('\n'),
  set: (v: string) => (editing.value.acme.domains = splitList(v)),
})
const resolversText = computed({
  get: () => (editing.value.acme.resolvers ?? []).join(', '),
  set: (v: string) => (editing.value.acme.resolvers = splitList(v)),
})

function emptyCert(): Cert {
  return {
    remark: '',
    enable: true,
    source: 'acme',
    acme: {
      domains: [],
      email: '',
      ca: 'letsencrypt',
      directoryUrl: '',
      eabKid: '',
      eabHmac: '',
      keyType: 'P256',
      provider: 'alidns',
      credentials: { id: '', secret: '' },
      zone: '',
      resolvers: [],
      propagationTimeout: 300,
      skipPropagationCheck: false,
      disableAuthoritativeCheck: false,
    },
    certFile: '',
    keyFile: '',
    certPem: '',
    keyPem: '',
    issuerPem: '',
    mapping: { enable: false, dir: '', script: '' },
  }
}

function sourceText(s: string) {
  return ({ acme: 'ACME', upload: '上传', file: '文件' } as Record<string, string>)[s] ?? s
}

function domainsOf(c: CertView) {
  return c.info?.domains ?? c.acme.domains ?? []
}

// certStatus 计算证书展示状态：停用的证书只显示“已停用”
function certStatus(c: CertView): { type: StatusType; text: string; title?: string } {
  if (!c.enable) {
    return { type: 'off', text: '已停用' }
  }
  if (c.state.obtaining) {
    return { type: 'pending', text: '签发中' }
  }
  if (c.info) {
    const d = Math.floor((new Date(c.info.notAfter).getTime() - Date.now()) / 86400000)
    if (d < 0) {
      return { type: 'error', text: '已过期' }
    }
    if (c.state.lastError) {
      return { type: 'warn', text: `剩余 ${d} 天 · 续签失败`, title: c.state.lastError }
    }
    return { type: d < 15 ? 'warn' : 'ok', text: `剩余 ${d} 天` }
  }
  if (c.state.lastError) {
    return { type: 'error', text: '签发失败', title: c.state.lastError }
  }
  return { type: 'pending', text: c.source === 'acme' ? '待签发' : '未加载' }
}

async function load() {
  try {
    certs.value = await api.get<CertView[]>('/api/ssl/certs')
  } finally {
    loading.value = false
  }
}

function openEditor(row?: CertView) {
  if (row) {
    const { info: _i, state: _s, ...cert } = row
    editing.value = JSON.parse(JSON.stringify(cert))
  } else {
    editing.value = emptyCert()
  }
  extra.value = { ...(editing.value.acme.credentials.extra ?? {}) }
  editorVisible.value = true
}

function readFile(e: Event, field: 'certPem' | 'keyPem') {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (f) {
    f.text().then((t) => (editing.value[field] = t))
  }
}

async function save() {
  editing.value.acme.credentials.extra = { ...extra.value }
  saving.value = true
  try {
    const { key, ...body } = editing.value
    if (key) {
      await api.put(`/api/ssl/certs/${key}`, body)
    } else {
      await api.post('/api/ssl/certs', body)
    }
    editorVisible.value = false
    ElMessage.success(body.source === 'acme' ? '已保存，需要签发时会在后台进行' : '已保存')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function toggle(row: CertView, enable: boolean) {
  try {
    await api.put(`/api/ssl/certs/${row.key}/enable`, { enable })
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function renew(row: CertView) {
  try {
    await api.post(`/api/ssl/certs/${row.key}/renew`)
    ElMessage.success('已开始签发，进度见日志')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

function download(row: CertView, type: string) {
  window.open(`/api/ssl/certs/${row.key}/download?type=${type}`, '_blank')
}

async function remove(row: CertView) {
  try {
    await api.del(`/api/ssl/certs/${row.key}`)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function loadRenew() {
  renewSettings.value = await api.get<RenewSettings>('/api/ssl/settings')
}

async function openSettings() {
  await loadRenew()
  settings.value = JSON.parse(JSON.stringify(renewSettings.value))
  settingsVisible.value = true
}

async function enableRenew() {
  if (!renewSettings.value) {
    return
  }
  enabling.value = true
  try {
    await api.put('/api/ssl/settings', { ...renewSettings.value, enable: true })
    ElMessage.success('已开启自动续签')
    await loadRenew()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    enabling.value = false
  }
}

async function saveSettings() {
  try {
    await api.put('/api/ssl/settings', settings.value)
    settingsVisible.value = false
    ElMessage.success('已保存')
    await loadRenew()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(async () => {
  loadRenew()
  providers.value = await api.get<ProviderMeta[]>('/api/ssl/providers')
  await load()
  timer = window.setInterval(load, 5000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.err-line {
  font-size: 12px;
  color: var(--el-color-danger);
  word-break: break-all;
}
.w-260 {
  width: 260px;
}
.w-160 {
  width: 160px;
}
.file-pick {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 6px;
  font-size: 12px;
  color: var(--el-color-primary);
  cursor: pointer;
}
.file-pick input {
  display: none;
}
.el-checkbox {
  white-space: normal;
  height: auto;
  min-height: 32px;
}
@media (max-width: 767.98px) {
  .w-260,
  .w-160 {
    width: 100%;
  }
}
</style>
