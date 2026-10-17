<template>
  <div>
    <PageHeader :title="t('ssl.title')" :desc="t('ssl.desc')" :icon="ShieldCheck">
      <CompactToggle v-model="compact" />
      <el-button :icon="SlidersHorizontal" @click="openSettings">{{ t('ssl.settingsTitle') }}</el-button>
      <el-button type="primary" :icon="Plus" @click="openEditor()">{{ t('ssl.addCert') }}</el-button>
    </PageHeader>

    <ModuleBanner
      v-if="renewSettings && !renewSettings.enable"
      :title="t('ssl.renewOffTitle')"
      :desc="t('ssl.renewOffDesc')"
      :action="t('ssl.renewOffAction')"
      :loading="enabling"
      @enable="enableRenew"
    />

    <div v-loading="loading" class="card-list" :class="{ 'is-compact': compact }">
      <el-empty v-if="!loading && !certs.length" :description="t('ssl.empty')" />
      <EntityCard v-for="c in certs" :key="c.key" :off="!c.enable" :compact="compact">
        <template #lead>
          <el-switch :model-value="c.enable" @change="(v: string | number | boolean) => toggle(c, Boolean(v))" />
        </template>
        <template #title>
          <span>{{ c.remark || t('ssl.unnamed') }}</span>
          <el-tag size="small" effect="plain">{{ sourceText(c.source) }}</el-tag>
        </template>
        <template #status>
          <StatusBadge v-bind="certStatus(c)" />
        </template>
        <Field :label="t('ssl.domain')" mono wide>{{ domainsOf(c).join(', ') || '-' }}</Field>
        <Field :label="t('ssl.validUntil')">{{ c.info ? new Date(c.info.notAfter).toLocaleDateString() : '-' }}</Field>
        <Field :label="t('ssl.issuer')">{{ c.info?.issuer || '-' }}</Field>
        <Field :label="t('ssl.updatedAt')">{{ c.updatedAt ? new Date(c.updatedAt).toLocaleString() : '-' }}</Field>
        <template v-if="c.enable && (c.state.lastError || c.state.mappingNote || c.state.nextRetry)" #extra>
          <div v-if="c.state.lastError" class="err-line">{{ c.state.lastError }}</div>
          <div v-if="c.state.nextRetry" class="meta">{{ t('ssl.nextRetry', { time: new Date(c.state.nextRetry).toLocaleString() }) }}</div>
          <div v-if="c.state.mappingNote" class="meta">{{ c.state.mappingNote }}</div>
        </template>
        <template #actions>
          <el-button v-if="c.source === 'acme'" text size="small" :icon="RotateCw" :disabled="c.state.obtaining || !c.enable" @click="renew(c)">{{ t('ssl.renew') }}</el-button>
          <el-dropdown v-if="c.info" trigger="click" @command="(cmd: string) => download(c, cmd)">
            <el-button text size="small" :icon="Download">{{ t('common.download') }}</el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="crt">{{ t('ssl.dlCrt') }}</el-dropdown-item>
                <el-dropdown-item command="key">{{ t('ssl.dlKey') }}</el-dropdown-item>
                <el-dropdown-item command="issuer" :disabled="!c.issuerPem">{{ t('ssl.dlIssuer') }}</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <el-button text size="small" :icon="Pencil" @click="openEditor(c)">{{ t('common.edit') }}</el-button>
          <el-popconfirm :title="t('ssl.confirmDelete')" width="240" @confirm="remove(c)">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Trash2">{{ t('common.delete') }}</el-button>
            </template>
          </el-popconfirm>
        </template>
      </EntityCard>
    </div>

    <LogCard url="/api/ssl/logs" clearable />

    <el-dialog v-model="editorVisible" :title="editing.key ? t('ssl.editCert') : t('ssl.addCert')" width="760px" top="5vh" :fullscreen="isMobile">
      <el-form :model="editing" label-width="130px" :label-position="labelPosition">
        <el-form-item :label="t('common.remark')">
          <el-input v-model="editing.remark" :placeholder="t('ssl.remarkPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('ssl.source')">
          <el-radio-group v-model="editing.source" :disabled="!!editing.key">
            <el-radio-button value="acme">{{ t('ssl.sourceAcme') }}</el-radio-button>
            <el-radio-button value="upload">{{ t('ssl.sourceUpload') }}</el-radio-button>
            <el-radio-button value="file">{{ t('ssl.sourceFile') }}</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <template v-if="editing.source === 'acme'">
          <el-form-item :label="t('ssl.domain')" required>
            <el-input v-model="domainsText" type="textarea" :rows="2" :placeholder="t('ssl.domainsPlaceholder')" class="mono" />
          </el-form-item>
          <el-form-item :label="t('ssl.email')"><el-input v-model="editing.acme.email" :placeholder="t('ssl.emailPlaceholder')" /></el-form-item>
          <el-form-item :label="t('ssl.ca')">
            <el-select v-model="editing.acme.ca" class="w-260">
              <el-option label="Let's Encrypt" value="letsencrypt" />
              <el-option :label="t('ssl.caStaging')" value="letsencrypt-staging" />
              <el-option label="ZeroSSL" value="zerossl" />
              <el-option label="Google Trust Services" value="google" />
              <el-option :label="t('ssl.caCustom')" value="custom" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="editing.acme.ca === 'custom'" :label="t('ssl.directoryUrl')">
            <el-input v-model="editing.acme.directoryUrl" placeholder="https://.../directory" />
          </el-form-item>
          <template v-if="['zerossl', 'google', 'custom'].includes(editing.acme.ca)">
            <el-form-item label="EAB Key ID"><el-input v-model="editing.acme.eabKid" /></el-form-item>
            <el-form-item label="EAB HMAC Key"><el-input v-model="editing.acme.eabHmac" type="password" show-password /></el-form-item>
          </template>
          <el-form-item :label="t('ssl.keyType')">
            <el-select v-model="editing.acme.keyType" class="w-160">
              <el-option label="ECC P-256" value="P256" />
              <el-option label="ECC P-384" value="P384" />
              <el-option label="RSA 2048" value="2048" />
              <el-option label="RSA 4096" value="4096" />
            </el-select>
          </el-form-item>
          <el-divider content-position="left">{{ t('ssl.dnsVerify') }}</el-divider>
          <el-form-item :label="t('ssl.dnsProvider')">
            <el-select v-model="editing.acme.provider" class="w-260" @change="extra = {}">
              <el-option v-for="p in providers" :key="p.name" :label="p.title" :value="p.name" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="currentProvider?.idLabel" :label="currentProvider.idLabel">
            <el-input v-model="editing.acme.credentials.id" />
          </el-form-item>
          <el-form-item v-if="currentProvider?.secretLabel" :label="currentProvider.secretLabel">
            <el-input v-model="editing.acme.credentials.secret" type="password" show-password :placeholder="t('ssl.secretPlaceholder')" />
          </el-form-item>
          <el-form-item v-for="f in currentProvider?.extra ?? []" :key="f.key" :label="f.label">
            <el-input v-model="extra[f.key]" :placeholder="f.placeholder" />
          </el-form-item>
          <el-form-item :label="t('ssl.zone')"><el-input v-model="editing.acme.zone" :placeholder="t('ssl.zonePlaceholder')" /></el-form-item>
          <el-form-item :label="t('ssl.resolvers')"><el-input v-model="resolversText" :placeholder="t('ssl.resolversPlaceholder')" /></el-form-item>
          <el-form-item :label="t('ssl.propagationTimeout')"><el-input-number v-model="editing.acme.propagationTimeout" :min="30" :max="3600" /></el-form-item>
          <el-form-item :label="t('ssl.propagationCheck')">
            <el-checkbox v-model="editing.acme.disableAuthoritativeCheck">{{ t('ssl.disableAuthCheck') }}</el-checkbox>
            <el-checkbox v-model="editing.acme.skipPropagationCheck">{{ t('ssl.skipCheck') }}</el-checkbox>
          </el-form-item>
        </template>

        <template v-if="editing.source === 'upload'">
          <el-form-item :label="t('ssl.certPem')" required>
            <el-input v-model="editing.certPem" type="textarea" :rows="5" class="mono" :placeholder="t('ssl.certPemPlaceholder')" />
            <label class="file-pick"><Upload :size="14" />{{ t('ssl.readFromFile') }}<input type="file" accept=".pem,.crt,.cer" @change="(e) => readFile(e, 'certPem')" /></label>
          </el-form-item>
          <el-form-item :label="t('ssl.keyPem')" required>
            <el-input v-model="editing.keyPem" type="textarea" :rows="4" class="mono" :placeholder="t('ssl.keyPemPlaceholder')" />
            <label class="file-pick"><Upload :size="14" />{{ t('ssl.readFromFile') }}<input type="file" accept=".pem,.key" @change="(e) => readFile(e, 'keyPem')" /></label>
          </el-form-item>
        </template>

        <template v-if="editing.source === 'file'">
          <el-form-item :label="t('ssl.certPath')" required><el-input v-model="editing.certFile" placeholder="/etc/ssl/example/fullchain.pem" /></el-form-item>
          <el-form-item :label="t('ssl.keyPath')" required><el-input v-model="editing.keyFile" placeholder="/etc/ssl/example/privkey.pem" /></el-form-item>
          <el-form-item label=" "><div class="form-help">{{ t('ssl.fileHelp') }}</div></el-form-item>
        </template>

        <el-divider content-position="left">{{ t('ssl.mapping') }}</el-divider>
        <el-form-item :label="t('ssl.writeDir')"><el-switch v-model="editing.mapping.enable" /></el-form-item>
        <template v-if="editing.mapping.enable">
          <el-form-item :label="t('ssl.dir')"><el-input v-model="editing.mapping.dir" :placeholder="t('ssl.dirPlaceholder')" /></el-form-item>
          <el-form-item :label="t('ssl.postScript')">
            <el-input v-model="editing.mapping.script" type="textarea" :rows="3" class="mono" :placeholder="t('ssl.postScriptPlaceholder')" />
            <div class="form-help">{{ t('ssl.envHelp') }}</div>
          </el-form-item>
        </template>
        <el-form-item :label="t('common.enable')"><el-switch v-model="editing.enable" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="save">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="settingsVisible" :title="t('ssl.settingsTitle')" width="520px" :fullscreen="isMobile">
      <el-form v-if="settings" :model="settings" label-width="170px" :label-position="labelPosition">
        <el-form-item :label="t('ssl.autoRenew')"><el-switch v-model="settings.enable" /></el-form-item>
        <el-form-item :label="t('ssl.renewBeforeDays')"><el-input-number v-model="settings.renewBeforeDays" :min="1" :max="60" /></el-form-item>
        <el-form-item :label="t('ssl.shortLivedHours')"><el-input-number v-model="settings.shortLivedRenewHours" :min="1" :max="120" /></el-form-item>
        <el-form-item :label="t('ssl.defaultEmail')"><el-input v-model="settings.defaultEmail" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="settingsVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="saveSettings">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Download, Pencil, Plus, RotateCw, ShieldCheck, SlidersHorizontal, Trash2, Upload } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import ModuleBanner from '../components/ModuleBanner.vue'
import CompactToggle from '../components/CompactToggle.vue'
import EntityCard from '../components/EntityCard.vue'
import { useCompactView } from '../composables/useCompactView'
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
const compact = useCompactView('ssl')
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
const { t } = useI18n()
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
  return ({ acme: 'ACME', upload: t('ssl.sourceUploadShort'), file: t('ssl.sourceFileShort') } as Record<string, string>)[s] ?? s
}

function domainsOf(c: CertView) {
  return c.info?.domains ?? c.acme.domains ?? []
}

// certStatus 计算证书展示状态：停用的证书只显示“已停用”
function certStatus(c: CertView): { type: StatusType; text: string; title?: string } {
  if (!c.enable) {
    return { type: 'off', text: t('ssl.deactivated') }
  }
  if (c.state.obtaining) {
    return { type: 'pending', text: t('ssl.issuing') }
  }
  if (c.info) {
    const d = Math.floor((new Date(c.info.notAfter).getTime() - Date.now()) / 86400000)
    if (d < 0) {
      return { type: 'error', text: t('ssl.expired') }
    }
    if (c.state.lastError) {
      return { type: 'warn', text: t('ssl.daysLeftRenewFailed', { days: d }), title: c.state.lastError }
    }
    return { type: d < 15 ? 'warn' : 'ok', text: t('ssl.daysLeft', { days: d }) }
  }
  if (c.state.lastError) {
    return { type: 'error', text: t('ssl.issueFailed'), title: c.state.lastError }
  }
  return { type: 'pending', text: c.source === 'acme' ? t('ssl.pendingIssue') : t('ssl.notLoaded') }
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
    f.text().then((text) => (editing.value[field] = text))
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
    ElMessage.success(body.source === 'acme' ? t('ssl.savedAcme') : t('common.saved'))
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
    ElMessage.success(t('ssl.renewStarted'))
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
    ElMessage.success(t('ssl.renewEnabled'))
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
    ElMessage.success(t('common.saved'))
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
