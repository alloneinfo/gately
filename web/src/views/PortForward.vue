<template>
  <div>
    <PageHeader :title="t('portForward.title')" :desc="t('portForward.desc')" :icon="ArrowLeftRight">
      <el-button :icon="SlidersHorizontal" @click="openSettings">{{ t('portForward.globalSettings') }}</el-button>
      <el-button type="primary" :icon="Plus" @click="openEditor()">{{ t('portForward.addRule') }}</el-button>
    </PageHeader>

    <ModuleBanner v-if="settings && !settings.enable" :title="t('portForward.moduleOffTitle')" :loading="enabling" @enable="enableModule" />

    <div v-loading="loading" class="card-list">
      <el-empty v-if="!loading && !rules.length" :description="t('portForward.empty')" />
      <EntityCard v-for="r in rules" :key="r.key" :off="!r.enable || moduleOff">
        <template #lead>
          <el-switch :model-value="r.enable" @change="(v: string | number | boolean) => toggle(r, Boolean(v))" />
        </template>
        <template #title>
          <span>{{ r.name || t('portForward.unnamed') }}</span>
          <span class="tags">
            <el-tag v-for="p in r.protocols" :key="p" size="small" effect="plain" class="mono">{{ p.toUpperCase() }}</el-tag>
          </span>
        </template>
        <template v-if="r.remark" #sub>{{ r.remark }}</template>
        <template #status>
          <el-tooltip v-if="r.details?.length && r.enable" placement="top">
            <template #content>
              <div v-for="d in r.details" :key="d">{{ d }}</div>
            </template>
            <TriangleAlert :size="16" class="detail" />
          </el-tooltip>
          <StatusBadge v-bind="ruleStatus(r)" />
        </template>
        <Field :label="t('portForward.listen')" mono>{{ listenText(r) }}</Field>
        <Field :label="t('portForward.target')" mono>
          <div v-for="addr in r.targetAddresses" :key="addr">{{ addr }}:{{ r.targetPorts || r.listenPorts }}</div>
        </Field>
        <Field :label="t('portForward.connections')">{{ t('portForward.connectionsValue', { active: r.stats.activeConns, total: r.stats.totalConns }) }}</Field>
        <Field :label="t('portForward.traffic')">↑ {{ formatBytes(r.stats.bytesIn) }} · ↓ {{ formatBytes(r.stats.bytesOut) }}</Field>
        <template #actions>
          <el-button text size="small" :icon="Pencil" @click="openEditor(r)">{{ t('common.edit') }}</el-button>
          <el-popconfirm :title="t('portForward.deleteConfirm')" @confirm="remove(r)">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Trash2">{{ t('common.delete') }}</el-button>
            </template>
          </el-popconfirm>
        </template>
      </EntityCard>
    </div>

    <LogCard url="/api/portforward/logs" clearable />

    <el-dialog v-model="editorVisible" :title="editing.key ? t('portForward.editRule') : t('portForward.addRule')" width="600px" :fullscreen="isMobile">
      <el-form :model="editing" label-width="110px" :label-position="labelPosition">
        <el-form-item :label="t('common.name')">
          <el-input v-model="editing.name" :placeholder="t('portForward.namePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('portForward.protocol')" required>
          <el-checkbox-group v-model="editing.protocols">
            <el-checkbox v-for="p in protocols" :key="p.value" :value="p.value">{{ p.label }}</el-checkbox>
          </el-checkbox-group>
          <div class="form-help">{{ t('portForward.protocolHelp') }}</div>
        </el-form-item>
        <el-form-item :label="t('portForward.listenAddress')">
          <el-input v-model="editing.listenAddress" :placeholder="t('portForward.listenAddressPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('portForward.listenPorts')" required>
          <el-input v-model="editing.listenPorts" :placeholder="t('portForward.portsPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('portForward.targetAddress')" required>
          <el-input v-model="targetsText" type="textarea" :rows="2" :placeholder="t('portForward.targetAddressPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('portForward.targetPorts')">
          <el-input v-model="editing.targetPorts" :placeholder="t('portForward.targetPortsPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('portForward.maxConnections')">
          <el-input-number v-model="editing.maxConnections" :min="0" :max="100000" />
          <span class="help">{{ t('portForward.useGlobalDefault') }}</span>
        </el-form-item>
        <el-form-item :label="t('portForward.udpTimeout')">
          <el-input-number v-model="editing.udpSessionTimeout" :min="0" :max="3600" />
          <span class="help">{{ t('portForward.useGlobalDefault') }}</span>
        </el-form-item>
        <el-form-item :label="t('common.remark')">
          <el-input v-model="editing.remark" />
        </el-form-item>
        <el-form-item :label="t('common.enable')">
          <el-switch v-model="editing.enable" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="save">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="settingsVisible" :title="t('portForward.settingsTitle')" width="520px" :fullscreen="isMobile">
      <el-form v-if="settingsForm" :model="settingsForm" label-width="150px" :label-position="labelPosition">
        <el-form-item :label="t('portForward.enableModule')"><el-switch v-model="settingsForm.enable" /></el-form-item>
        <el-form-item :label="t('portForward.defaultMaxConnections')"><el-input-number v-model="settingsForm.defaultMaxConnections" :min="1" /></el-form-item>
        <el-form-item :label="t('portForward.defaultUdpTimeout')"><el-input-number v-model="settingsForm.defaultUDPSessionTimeout" :min="1" /></el-form-item>
        <el-form-item :label="t('portForward.maxUdpSessions')"><el-input-number v-model="settingsForm.maxUDPSessions" :min="1" /></el-form-item>
        <el-form-item :label="t('portForward.dialTimeout')"><el-input-number v-model="settingsForm.dialTimeout" :min="1" :max="60" /></el-form-item>
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
import { ArrowLeftRight, Pencil, Plus, SlidersHorizontal, Trash2, TriangleAlert } from 'lucide-vue-next'
import { api, formatBytes } from '../api'
import PageHeader from '../components/PageHeader.vue'
import ModuleBanner from '../components/ModuleBanner.vue'
import EntityCard from '../components/EntityCard.vue'
import Field from '../components/Field.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import { useFormLayout } from '../composables/useIsMobile'

interface Rule {
  key?: string
  name: string
  enable: boolean
  protocols: string[]
  listenAddress: string
  listenPorts: string
  targetAddresses: string[]
  targetPorts: string
  maxConnections: number
  udpSessionTimeout: number
  remark: string
}

interface RuleView extends Rule {
  running: boolean
  error?: string
  details?: string[]
  stats: { activeConns: number; totalConns: number; bytesIn: number; bytesOut: number }
}

interface Settings {
  enable: boolean
  defaultMaxConnections: number
  defaultUDPSessionTimeout: number
  maxUDPSessions: number
  dialTimeout: number
}

const protocols = [
  { value: 'tcp', label: 'TCP' },
  { value: 'udp', label: 'UDP' },
  { value: 'tcp4', label: 'TCP4' },
  { value: 'udp4', label: 'UDP4' },
  { value: 'tcp6', label: 'TCP6' },
  { value: 'udp6', label: 'UDP6' },
]

const { t } = useI18n()
const { isMobile, labelPosition } = useFormLayout()
const rules = ref<RuleView[]>([])
const loading = ref(true)
const editorVisible = ref(false)
const saving = ref(false)
const editing = ref<Rule>(emptyRule())
const targetsText = ref('')
const settings = ref<Settings>()
const settingsForm = ref<Settings>()
const settingsVisible = ref(false)
const enabling = ref(false)
let timer: number | undefined

const moduleOff = computed(() => settings.value?.enable === false)

function emptyRule(): Rule {
  return {
    name: '',
    enable: true,
    protocols: ['tcp'],
    listenAddress: '',
    listenPorts: '',
    targetAddresses: [],
    targetPorts: '',
    maxConnections: 0,
    udpSessionTimeout: 0,
    remark: '',
  }
}

function listenText(r: Rule) {
  const host = r.listenAddress ? (r.listenAddress.includes(':') ? `[${r.listenAddress}]` : r.listenAddress) : '*'
  return `${host}:${r.listenPorts}`
}

// ruleStatus 计算规则的展示状态：停用的规则只显示“已停用”，不显示为异常
function ruleStatus(r: RuleView): { type: StatusType; text: string; title?: string } {
  if (moduleOff.value) {
    return { type: 'off', text: t('portForward.moduleOff') }
  }
  if (!r.enable) {
    return { type: 'off', text: t('portForward.statusOff') }
  }
  if (r.error) {
    return { type: 'error', text: r.error, title: r.error }
  }
  if (r.running) {
    return { type: 'running', text: t('common.running') }
  }
  return { type: 'warn', text: t('portForward.statusNotRunning') }
}

async function load() {
  try {
    rules.value = await api.get<RuleView[]>('/api/portforward/rules')
  } finally {
    loading.value = false
  }
}

async function loadSettings() {
  settings.value = await api.get<Settings>('/api/portforward/settings')
}

function openEditor(row?: RuleView) {
  if (row) {
    const { running: _r, error: _e, details: _d, stats: _s, ...rule } = row
    editing.value = JSON.parse(JSON.stringify(rule))
  } else {
    editing.value = emptyRule()
  }
  targetsText.value = editing.value.targetAddresses.join('\n')
  editorVisible.value = true
}

async function save() {
  editing.value.targetAddresses = targetsText.value
    .split(/[\n,]/)
    .map((s) => s.trim())
    .filter(Boolean)
  saving.value = true
  try {
    const { key, ...body } = editing.value
    if (key) {
      await api.put(`/api/portforward/rules/${key}`, body)
    } else {
      await api.post('/api/portforward/rules', body)
    }
    editorVisible.value = false
    ElMessage.success(t('common.saved'))
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function toggle(row: RuleView, enable: boolean) {
  try {
    await api.put(`/api/portforward/rules/${row.key}/enable`, { enable })
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function remove(row: RuleView) {
  try {
    await api.del(`/api/portforward/rules/${row.key}`)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function openSettings() {
  await loadSettings()
  settingsForm.value = JSON.parse(JSON.stringify(settings.value))
  settingsVisible.value = true
}

async function saveSettings() {
  try {
    await api.put('/api/portforward/settings', settingsForm.value)
    settingsVisible.value = false
    ElMessage.success(t('portForward.savedRestarted'))
    await Promise.all([loadSettings(), load()])
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

// enableModule 在横幅中一键启用模块
async function enableModule() {
  if (!settings.value) {
    return
  }
  enabling.value = true
  try {
    await api.put('/api/portforward/settings', { ...settings.value, enable: true })
    ElMessage.success(t('portForward.moduleEnabled'))
    await Promise.all([loadSettings(), load()])
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    enabling.value = false
  }
}

onMounted(() => {
  load()
  loadSettings()
  // 定时刷新连接数与流量
  timer = window.setInterval(load, 3000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.detail {
  color: var(--el-color-warning);
}
</style>
