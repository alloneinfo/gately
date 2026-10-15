<template>
  <div>
    <div class="page-header">
      <h2>SSL 证书</h2>
      <div>
        <el-button @click="openSettings">续签设置</el-button>
        <el-button type="primary" @click="openEditor()">添加证书</el-button>
      </div>
    </div>

    <el-card shadow="never">
      <el-table :data="certs" v-loading="loading" empty-text="暂无证书，可申请、上传或在设置中导入 Lucky 配置">
        <el-table-column label="启用" width="70">
          <template #default="{ row }">
            <el-switch :model-value="row.enable" size="small" @change="(v: string | number | boolean) => toggle(row, Boolean(v))" />
          </template>
        </el-table-column>
        <el-table-column label="证书" min-width="220">
          <template #default="{ row }">
            <div class="remark-title">{{ row.remark || '未命名' }}</div>
            <div class="mono domains">{{ domainsOf(row).join(', ') }}</div>
          </template>
        </el-table-column>
        <el-table-column label="来源" width="100">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">{{ sourceText(row.source) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="有效期" min-width="170">
          <template #default="{ row }">
            <template v-if="row.info">
              <el-tag :type="expiryType(row.info.notAfter)" size="small">{{ daysLeft(row.info.notAfter) }}</el-tag>
              <div class="meta">至 {{ new Date(row.info.notAfter).toLocaleDateString() }} · {{ row.info.issuer }}</div>
            </template>
            <span v-else class="meta">尚无证书</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="200">
          <template #default="{ row }">
            <el-tag v-if="row.state.obtaining" type="warning" size="small">签发中</el-tag>
            <div v-if="row.state.lastError" class="error" :title="row.state.lastError">{{ row.state.lastError }}</div>
            <div v-if="row.state.mappingNote" class="meta">{{ row.state.mappingNote }}</div>
            <div v-if="row.state.nextRetry" class="meta">下次重试 {{ new Date(row.state.nextRetry).toLocaleString() }}</div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.source === 'acme'" link type="primary" :disabled="row.state.obtaining" @click="renew(row)">续签</el-button>
            <el-dropdown v-if="row.info" trigger="click" @command="(c: string) => download(row, c)">
              <el-button link type="primary">下载</el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="crt">证书 (.crt)</el-dropdown-item>
                  <el-dropdown-item command="key">私钥 (.key)</el-dropdown-item>
                  <el-dropdown-item command="issuer" :disabled="!row.issuerPem">颁发者证书</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <el-button link type="primary" @click="openEditor(row)">编辑</el-button>
            <el-popconfirm title="确定删除该证书？已映射的文件不会删除" @confirm="remove(row)">
              <template #reference>
                <el-button link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never" header="日志" class="logs">
      <LogViewer url="/api/ssl/logs" clearable />
    </el-card>

    <el-dialog v-model="editorVisible" :title="editing.key ? '编辑证书' : '添加证书'" width="760px" top="5vh">
      <el-form :model="editing" label-width="130px">
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
            <el-select v-model="editing.acme.ca" style="width: 260px">
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
            <el-select v-model="editing.acme.keyType" style="width: 160px">
              <el-option label="ECC P-256" value="P256" />
              <el-option label="ECC P-384" value="P384" />
              <el-option label="RSA 2048" value="2048" />
              <el-option label="RSA 4096" value="4096" />
            </el-select>
          </el-form-item>
          <el-divider content-position="left">DNS 验证</el-divider>
          <el-form-item label="DNS 服务商">
            <el-select v-model="editing.acme.provider" style="width: 260px" @change="extra = {}">
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
            <input type="file" accept=".pem,.crt,.cer" @change="(e) => readFile(e, 'certPem')" />
          </el-form-item>
          <el-form-item label="私钥 (PEM)" required>
            <el-input v-model="editing.keyPem" type="textarea" :rows="4" class="mono" placeholder="已保存的私钥显示为 ********，不修改则保持原值" />
            <input type="file" accept=".pem,.key" @change="(e) => readFile(e, 'keyPem')" />
          </el-form-item>
        </template>

        <template v-if="editing.source === 'file'">
          <el-form-item label="证书路径" required><el-input v-model="editing.certFile" placeholder="/etc/ssl/example/fullchain.pem" /></el-form-item>
          <el-form-item label="私钥路径" required><el-input v-model="editing.keyFile" placeholder="/etc/ssl/example/privkey.pem" /></el-form-item>
          <div class="hint">定期检查文件变化并自动加载，适合由其他工具（如 certbot）维护的证书。</div>
        </template>

        <el-divider content-position="left">证书映射</el-divider>
        <el-form-item label="写入目录"><el-switch v-model="editing.mapping.enable" /></el-form-item>
        <template v-if="editing.mapping.enable">
          <el-form-item label="目录"><el-input v-model="editing.mapping.dir" placeholder="绝对路径，将写入 备注.crt / .key / .pem / _issuerCertificate.crt" /></el-form-item>
          <el-form-item label="变化后执行">
            <el-input v-model="editing.mapping.script" type="textarea" :rows="3" class="mono" placeholder="例如 docker exec nginx nginx -s reload" />
            <div class="hint">环境变量：certName certDir certFile keyFile</div>
          </el-form-item>
        </template>
        <el-form-item label="启用"><el-switch v-model="editing.enable" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="settingsVisible" title="续签设置" width="520px">
      <el-form v-if="settings" :model="settings" label-width="170px">
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
import { api } from '../api'
import LogViewer from '../components/LogViewer.vue'

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
const settings = ref<{ enable: boolean; renewBeforeDays: number; shortLivedRenewHours: number; defaultEmail: string }>()
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

function daysLeft(t: string) {
  const d = Math.floor((new Date(t).getTime() - Date.now()) / 86400000)
  return d < 0 ? '已过期' : `剩余 ${d} 天`
}

function expiryType(t: string) {
  const d = (new Date(t).getTime() - Date.now()) / 86400000
  return d < 0 ? 'danger' : d < 15 ? 'warning' : 'success'
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

async function openSettings() {
  settings.value = await api.get('/api/ssl/settings')
  settingsVisible.value = true
}

async function saveSettings() {
  try {
    await api.put('/api/ssl/settings', settings.value)
    settingsVisible.value = false
    ElMessage.success('已保存')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(async () => {
  providers.value = await api.get<ProviderMeta[]>('/api/ssl/providers')
  await load()
  timer = window.setInterval(load, 5000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.remark-title {
  font-weight: 500;
}
.domains {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.error {
  font-size: 12px;
  color: var(--el-color-danger);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 280px;
}
.hint {
  margin: -6px 0 12px 130px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.el-form-item .hint {
  margin: 4px 0 0;
}
.logs {
  margin-top: 16px;
}
</style>
