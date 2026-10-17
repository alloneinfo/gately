<template>
  <div>
    <PageHeader :title="t('wol.title')" :desc="t('wol.desc')" :icon="Power">
      <el-button type="primary" :icon="Plus" @click="openEditor()">{{ t('wol.addDevice') }}</el-button>
    </PageHeader>

    <div class="card-grid">
      <el-empty v-if="!devices.length" :description="t('wol.empty')" class="empty" />
      <EntityCard v-for="d in devices" :key="d.key">
        <template #lead>
          <div class="dev-icon" :class="{ online: status[d.key ?? ''] === 'online' }"><Monitor :size="18" /></div>
        </template>
        <template #title>{{ d.name }}</template>
        <template v-if="d.remark" #sub>{{ d.remark }}</template>
        <template #status>
          <StatusBadge v-bind="devStatus(d)" />
        </template>
        <Field :label="t('wol.macAddress')" mono>
          <div v-for="m in d.macs" :key="m">{{ m }}</div>
        </Field>
        <Field :label="t('wol.broadcast')" mono>{{ (d.broadcastIps ?? []).length ? (d.broadcastIps ?? []).join(', ') : t('wol.auto') }}:{{ d.port }} ×{{ d.repeat }}</Field>
        <Field v-if="d.checkAddr" :label="t('wol.onlineCheck')" mono>{{ d.checkAddr }}</Field>
        <template #actions>
          <el-button text size="small" :icon="Pencil" @click="openEditor(d)">{{ t('common.edit') }}</el-button>
          <el-popconfirm :title="t('wol.confirmDelete')" @confirm="remove(d)">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Trash2">{{ t('common.delete') }}</el-button>
            </template>
          </el-popconfirm>
          <el-button type="primary" size="small" :icon="Zap" :loading="waking[d.key ?? '']" class="wake" @click="wake(d)">{{ t('wol.wake') }}</el-button>
        </template>
      </EntityCard>
    </div>

    <LogCard url="/api/wol/logs" clearable />

    <el-dialog v-model="editorVisible" :title="editing.key ? t('wol.editDevice') : t('wol.addDevice')" width="540px" :fullscreen="isMobile">
      <el-form :model="editing" label-width="110px" :label-position="labelPosition">
        <el-form-item :label="t('common.name')"><el-input v-model="editing.name" /></el-form-item>
        <el-form-item :label="t('wol.macAddress')"><el-input v-model="macsText" type="textarea" :rows="2" class="mono" :placeholder="t('wol.macPlaceholder')" /></el-form-item>
        <el-form-item :label="t('wol.broadcastAddress')"><el-input v-model="bcText" :placeholder="t('wol.broadcastPlaceholder')" /></el-form-item>
        <el-form-item :label="t('wol.port')"><el-input-number v-model="editing.port" :min="1" :max="65535" /></el-form-item>
        <el-form-item :label="t('wol.repeat')"><el-input-number v-model="editing.repeat" :min="1" :max="20" /></el-form-item>
        <el-form-item :label="t('wol.onlineCheck')">
          <el-input v-model="editing.checkAddr" :placeholder="t('wol.checkPlaceholder')" />
          <div class="form-help">{{ t('wol.checkHelp') }}</div>
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
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { Monitor, Pencil, Plus, Power, Trash2, Zap } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import EntityCard from '../components/EntityCard.vue'
import Field from '../components/Field.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import { useFormLayout } from '../composables/useIsMobile'

interface Device {
  key?: string
  name: string
  macs: string[]
  broadcastIps: string[] | null
  port: number
  repeat: number
  checkAddr: string
  remark: string
  enable: boolean
}

const { t } = useI18n()
const { isMobile, labelPosition } = useFormLayout()
const devices = ref<Device[]>([])
const status = ref<Record<string, string>>({})
const waking = reactive<Record<string, boolean>>({})
const editorVisible = ref(false)
const editing = ref<Device>(empty())
let timer: number | undefined

const macsText = computed({
  get: () => editing.value.macs.join('\n'),
  set: (v: string) => (editing.value.macs = v.split(/[\n,]/).map((s) => s.trim()).filter(Boolean)),
})
const bcText = computed({
  get: () => (editing.value.broadcastIps ?? []).join(', '),
  set: (v: string) => (editing.value.broadcastIps = v.split(',').map((s) => s.trim()).filter(Boolean)),
})

function empty(): Device {
  return { name: '', macs: [], broadcastIps: [], port: 9, repeat: 3, checkAddr: '', remark: '', enable: true }
}

// devStatus 计算设备在线状态；未配置探测地址时明确提示，而不是显示为等待
function devStatus(d: Device): { type: StatusType; text: string } {
  if (!d.checkAddr) {
    return { type: 'info', text: t('wol.noCheck') }
  }
  const s = status.value[d.key ?? '']
  if (s === 'online') {
    return { type: 'running', text: t('wol.online') }
  }
  if (s === 'offline') {
    return { type: 'off', text: t('wol.offline') }
  }
  return { type: 'pending', text: t('wol.probing') }
}

async function load() {
  devices.value = await api.get('/api/wol/devices')
  status.value = await api.get('/api/wol/status')
}

function openEditor(d?: Device) {
  editing.value = d ? JSON.parse(JSON.stringify(d)) : empty()
  editorVisible.value = true
}

async function save() {
  try {
    const { key, ...body } = editing.value
    if (key) {
      await api.put(`/api/wol/devices/${key}`, body)
    } else {
      await api.post('/api/wol/devices', body)
    }
    editorVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function wake(d: Device) {
  const k = d.key ?? ''
  waking[k] = true
  try {
    const r = await api.post<{ sent: number }>(`/api/wol/devices/${k}/wake`)
    ElMessage.success(t('wol.sent', { count: r.sent }))
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    waking[k] = false
  }
}

async function remove(d: Device) {
  try {
    await api.del(`/api/wol/devices/${d.key}`)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(() => {
  load()
  timer = window.setInterval(load, 10000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.dev-icon {
  width: 34px;
  height: 34px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}
.dev-icon.online {
  background: var(--el-color-success-light-9);
  color: var(--el-color-success);
}
.wake {
  margin-left: 6px !important;
}
.empty {
  grid-column: 1 / -1;
}
</style>
