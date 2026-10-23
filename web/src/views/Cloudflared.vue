<template>
  <div>
    <PageHeader :title="t('cloudflared.title')" :desc="t('cloudflared.desc')" :icon="Cloud">
      <el-button type="primary" :icon="Plus" @click="openEditor()">{{ t('cloudflared.addTunnel') }}</el-button>
    </PageHeader>

    <el-alert v-if="binError" type="warning" :title="binError" :closable="false" show-icon class="block-gap" />

    <div class="card-grid">
      <el-empty v-if="!tunnels.length" :description="t('cloudflared.empty')" class="empty" />
      <EntityCard v-for="tn in tunnels" :key="tn.key" :off="!tn.enable">
        <template #title>{{ tn.name }}</template>
        <template #sub>{{ tn.mode === 'token' ? t('cloudflared.modeToken') : t('cloudflared.modeQuick') }}<template v-if="tn.remark"> · {{ tn.remark }}</template></template>
        <template #status><StatusBadge v-bind="tunnelBadge(tn)" /></template>
        <Field v-if="tn.mode === 'quick'" :label="t('cloudflared.localService')" mono>{{ tn.service }}</Field>
        <Field v-if="tn.mode === 'quick'" :label="t('cloudflared.publicAddr')" mono wide>
          <a v-if="tn.url" :href="tn.url" target="_blank" rel="noopener">{{ tn.url }}</a>
          <span v-else class="muted">{{ tn.state.running ? t('cloudflared.requesting') : '—' }}</span>
        </Field>
        <Field v-if="tn.mode === 'token'" :label="t('cloudflared.edgeConnections')" mono>{{ tn.connections }}</Field>
        <Field :label="t('cloudflared.protocol')" mono>{{ tn.protocol || 'auto' }}</Field>
        <template #actions>
          <el-switch :model-value="tn.enable" size="small" @change="(v: boolean | string | number) => toggle(tn, Boolean(v))" />
          <el-button text size="small" :icon="RefreshCw" :disabled="!tn.enable" @click="restart(tn)">{{ t('common.restart') }}</el-button>
          <el-button text size="small" :icon="Pencil" @click="openEditor(tn)">{{ t('common.edit') }}</el-button>
          <el-popconfirm :title="t('cloudflared.confirmDelete')" @confirm="remove(tn)">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Trash2">{{ t('common.delete') }}</el-button>
            </template>
          </el-popconfirm>
        </template>
      </EntityCard>
    </div>

    <div class="g-card">
      <div class="g-card-head">
        <div class="g-card-title"><Terminal :size="16" />{{ t('cloudflared.program') }} <span v-if="version" class="g-card-desc">{{ version }}</span></div>
        <el-button size="small" :icon="Save" @click="saveSettings">{{ t('common.save') }}</el-button>
      </div>
      <div class="g-card-body">
        <el-form label-width="110px" :label-position="labelPosition" class="svc-form">
          <el-form-item :label="t('cloudflared.binary')">
            <el-input v-model="binPath" :placeholder="binary || t('cloudflared.binaryPlaceholder')" class="mono" />
          </el-form-item>
        </el-form>
      </div>
    </div>

    <LogCard url="/api/cloudflared/logs" clearable />

    <el-dialog v-model="editorVisible" :title="editing.key ? t('cloudflared.editTunnel') : t('cloudflared.addTunnel')" width="540px" :fullscreen="isMobile">
      <el-form :model="editing" label-width="100px" :label-position="labelPosition">
        <el-form-item :label="t('common.name')"><el-input v-model="editing.name" /></el-form-item>
        <el-form-item :label="t('cloudflared.mode')">
          <el-radio-group v-model="editing.mode">
            <el-radio-button value="token">{{ t('cloudflared.modeToken') }}</el-radio-button>
            <el-radio-button value="quick">{{ t('cloudflared.modeQuick') }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="editing.mode === 'token'" :label="t('cloudflared.token')">
          <el-input v-model="editing.token" type="password" show-password placeholder="eyJh…" />
          <div class="form-help">{{ t('cloudflared.tokenHelp') }}</div>
        </el-form-item>
        <el-form-item v-else :label="t('cloudflared.localService')">
          <el-input v-model="editing.service" placeholder="http://localhost:8080" class="mono" />
          <div class="form-help">{{ t('cloudflared.serviceHelp') }}</div>
        </el-form-item>
        <el-form-item :label="t('cloudflared.protocol')">
          <el-select v-model="editing.protocol">
            <el-option label="auto" value="auto" />
            <el-option label="quic" value="quic" />
            <el-option label="http2" value="http2" />
          </el-select>
          <div class="form-help">{{ t('cloudflared.protocolHelp') }}</div>
        </el-form-item>
        <el-form-item :label="t('cloudflared.edgeIp')">
          <el-select v-model="editing.edgeIpVersion">
            <el-option label="auto" value="auto" />
            <el-option label="IPv4" value="4" />
            <el-option label="IPv6" value="6" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('common.remark')"><el-input v-model="editing.remark" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="save">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Cloud, Pencil, Plus, RefreshCw, Save, Terminal, Trash2 } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import EntityCard from '../components/EntityCard.vue'
import Field from '../components/Field.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import { useFormLayout } from '../composables/useIsMobile'
import { usePolling } from '../composables/usePolling'

interface Tunnel {
  key?: string
  name: string
  enable: boolean
  mode: string
  token: string
  service: string
  protocol: string
  edgeIpVersion: string
  remark: string
}
// TunnelView 是后端返回的隧道，附带运行状态
interface TunnelView extends Tunnel {
  state: { running: boolean }
  error: string
  url?: string
  connections: number
}

const { t } = useI18n()
const { isMobile, labelPosition } = useFormLayout()
const tunnels = ref<TunnelView[]>([])
const binPath = ref('')
const binary = ref('')
const version = ref('')
const binError = ref('')
const editorVisible = ref(false)
const editing = ref<Tunnel>(empty())
// 定时刷新，标签页在后台时暂停
const polling = usePolling(loadTunnels, 4000)

function empty(): Tunnel {
  return { name: '', enable: true, mode: 'token', token: '', service: 'http://localhost:8080', protocol: 'auto', edgeIpVersion: 'auto', remark: '' }
}

function tunnelBadge(tn: TunnelView): { type: StatusType; text: string; title?: string } {
  if (!tn.enable) {
    return { type: 'off', text: t('cloudflared.deactivated') }
  }
  if (tn.error) {
    return { type: 'error', text: t('cloudflared.startFailed'), title: tn.error }
  }
  if (!tn.state.running) {
    return { type: 'warn', text: t('cloudflared.notRunning') }
  }
  if (tn.mode === 'token' && tn.connections === 0) {
    return { type: 'pending', text: t('cloudflared.connecting') }
  }
  return { type: 'running', text: t('common.running') }
}

async function loadTunnels() {
  try {
    tunnels.value = await api.get<TunnelView[]>('/api/cloudflared/tunnels')
  } catch {
    // 轮询失败时保留上一次的状态
  }
}

async function loadSettings() {
  const r = await api.get<{ settings: { binPath: string }; binary: string; version: string; error: string }>('/api/cloudflared/settings')
  binPath.value = r.settings.binPath
  binary.value = r.binary
  version.value = r.version
  binError.value = r.error
}

async function saveSettings() {
  try {
    await api.put('/api/cloudflared/settings', { binPath: binPath.value })
    ElMessage.success(t('common.saved'))
    await Promise.all([loadSettings(), loadTunnels()])
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

function openEditor(tn?: Tunnel) {
  editing.value = tn ? JSON.parse(JSON.stringify(tn)) : empty()
  editorVisible.value = true
}

async function save() {
  try {
    const { key, ...body } = editing.value
    if (key) {
      await api.put(`/api/cloudflared/tunnels/${key}`, body)
    } else {
      await api.post('/api/cloudflared/tunnels', body)
    }
    editorVisible.value = false
    await loadTunnels()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function toggle(tn: Tunnel, enable: boolean) {
  try {
    await api.put(`/api/cloudflared/tunnels/${tn.key}/enable`, { enable })
    await loadTunnels()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function restart(tn: Tunnel) {
  try {
    await api.post(`/api/cloudflared/tunnels/${tn.key}/restart`)
    await loadTunnels()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function remove(tn: Tunnel) {
  try {
    await api.del(`/api/cloudflared/tunnels/${tn.key}`)
    await loadTunnels()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(() => {
  loadSettings()
  loadTunnels()
  polling.start()
})
</script>

<style scoped>
.block-gap {
  margin-bottom: 16px;
}
.svc-form {
  max-width: 640px;
}
.svc-form .el-form-item:last-child {
  margin-bottom: 0;
}
.muted {
  color: var(--el-text-color-secondary);
}
.empty {
  grid-column: 1 / -1;
}
.card-grid + .g-card {
  margin-top: 16px;
}
</style>
