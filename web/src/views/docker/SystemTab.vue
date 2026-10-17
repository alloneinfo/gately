<template>
  <div>
    <div class="g-card">
      <div class="g-card-head">
        <div class="g-card-title"><Server :size="16" />{{ t('docker.system.infoTitle') }}</div>
        <el-button size="small" :icon="RefreshCw" :loading="loading" @click="load">{{ t('common.refresh') }}</el-button>
      </div>
      <div v-if="info" class="grid">
        <Field :label="t('docker.system.version')">{{ info.version }}</Field>
        <Field :label="t('docker.system.apiVersion')">{{ info.apiVersion }}</Field>
        <Field :label="t('docker.system.osArch')">{{ info.os }} / {{ info.arch }}</Field>
        <Field :label="t('docker.system.kernel')" mono>{{ info.kernel }}</Field>
        <Field :label="t('docker.system.hostname')">{{ info.name }}</Field>
        <Field :label="t('docker.system.cpuMem')">{{ t('docker.system.cpuMemValue', { cpus: info.cpus, mem: formatBytes(info.memTotal) }) }}</Field>
        <Field :label="t('docker.system.storageDriver')">{{ info.storageDriver }}</Field>
        <Field :label="t('docker.system.dataDir')" mono>{{ info.rootDir }}</Field>
        <Field :label="t('docker.system.containers')">{{ t('docker.system.containersSummary', { total: info.containers.total, running: info.containers.running, paused: info.containers.paused, stopped: info.containers.stopped }) }}</Field>
        <Field :label="t('docker.system.images')">{{ info.images }}</Field>
        <Field label="Compose CLI">{{ info.composeCli ? info.composeCliVersion || t('docker.system.composeAvailable') : t('docker.system.composeMissing') }}</Field>
      </div>
      <el-empty v-else-if="!loading" :description="t('docker.system.infoFailed')" :image-size="50" />
    </div>

    <div class="g-card">
      <div class="g-card-head">
        <div class="g-card-title"><HardDrive :size="16" />{{ t('docker.system.diskTitle') }}</div>
      </div>
      <div v-if="disk" class="g-card-body disks">
        <div v-for="d in diskItems" :key="d.key" class="disk">
          <RingGauge :value="pct(d.item)" :size="64" :stroke="7" />
          <div class="d-main">
            <div class="d-name">{{ d.name }} <span class="muted">{{ t('docker.system.items', { count: d.item.count }) }}</span></div>
            <div class="d-size">{{ formatBytes(d.item.size) }}</div>
            <div class="d-rec muted">{{ t('docker.system.reclaimable', { size: formatBytes(d.item.reclaimable) }) }}</div>
          </div>
        </div>
      </div>
      <div class="g-card-body prune">
        <div class="p-title">{{ t('docker.system.prune') }}</div>
        <div class="p-btns">
          <el-button v-for="p in prunes" :key="p.key" :icon="Eraser" :loading="pruning === p.key" @click="prune(p)">{{ p.label }}</el-button>
        </div>
      </div>
    </div>

    <SettingsPanel @saved="onSaved" />
    <TaskHistory ref="history" />
    <LogCard url="/api/docker/logs" :title="t('docker.system.logTitle')" clearable />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Eraser, HardDrive, RefreshCw, Server } from 'lucide-vue-next'
import { api } from '../../api'
import Field from '../../components/Field.vue'
import LogCard from '../../components/LogCard.vue'
import RingGauge from '../../components/RingGauge.vue'
import SettingsPanel from './SettingsPanel.vue'
import TaskHistory from './TaskHistory.vue'
import { formatBytes } from './format'
import { dockerInfo, reportError } from './state'
import type { DiskItem, DiskUsage } from './types'

const { t } = useI18n()

const emit = defineEmits<{ saved: [] }>()
const info = dockerInfo
const disk = ref<DiskUsage>()
const loading = ref(false)
const pruning = ref('')
const history = ref<InstanceType<typeof TaskHistory>>()

interface PruneDef {
  key: string
  label: string
  target: string
  all?: boolean
  tip: string
}

// 文案随界面语言变化，所以用 computed 在使用处翻译
const prunes = computed<PruneDef[]>(() => [
  { key: 'containers', label: t('docker.system.pruneContainers'), target: 'containers', tip: t('docker.system.pruneContainersTip') },
  { key: 'dangling', label: t('docker.system.pruneDangling'), target: 'images', tip: t('docker.system.pruneDanglingTip') },
  { key: 'allImages', label: t('docker.system.pruneAllImages'), target: 'images', all: true, tip: t('docker.system.pruneAllImagesTip') },
  { key: 'networks', label: t('docker.system.pruneNetworks'), target: 'networks', tip: t('docker.system.pruneNetworksTip') },
  { key: 'volumes', label: t('docker.system.pruneVolumes'), target: 'volumes', tip: t('docker.system.pruneVolumesTip') },
  { key: 'buildCache', label: t('docker.system.pruneBuildCache'), target: 'buildcache', tip: t('docker.system.pruneBuildCacheTip') },
])

const diskItems = computed(() => {
  const d = disk.value
  if (!d) {
    return []
  }
  return [
    { key: 'images', name: t('docker.system.images'), item: d.images },
    { key: 'containers', name: t('docker.system.containers'), item: d.containers },
    { key: 'volumes', name: t('docker.system.volumes'), item: d.volumes },
    { key: 'buildCache', name: t('docker.system.buildCache'), item: d.buildCache },
  ]
})

// 可回收占比，总量为 0 时按 0 处理
function pct(i: DiskItem): number {
  return i.size > 0 ? (i.reclaimable / i.size) * 100 : 0
}

async function load() {
  loading.value = true
  try {
    dockerInfo.value = await api.get('/api/docker/info')
    reportError('')
    disk.value = await api.get<DiskUsage>('/api/docker/disk-usage')
  } catch (e) {
    reportError((e as Error).message)
  } finally {
    loading.value = false
  }
}

async function prune(p: PruneDef) {
  try {
    await ElMessageBox.confirm(t('docker.system.confirmContinue', { tip: p.tip }), p.label, { type: 'warning' })
  } catch {
    return
  }
  pruning.value = p.key
  try {
    const r = await api.post<{ deleted: number; spaceReclaimed: number }>('/api/docker/prune', { target: p.target, all: p.all })
    ElMessage.success(t('docker.system.pruned', { count: r.deleted, size: formatBytes(r.spaceReclaimed) }))
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    pruning.value = ''
  }
}

function onSaved() {
  load()
  emit('saved')
}

defineExpose({ reload: () => Promise.all([load(), history.value?.reload()]) })
onMounted(load)
</script>

<style scoped>
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 12px 20px;
  padding: 14px 16px;
}
.disks {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(230px, 1fr));
  gap: 16px;
}
.disk {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
}
.d-name {
  font-weight: 600;
  font-size: 13px;
}
.d-size {
  font-size: 18px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}
.d-rec {
  font-size: 12px;
}
.prune {
  border-top: 1px solid var(--g-border);
}
.p-title {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 8px;
}
.p-btns {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.p-btns .el-button + .el-button {
  margin-left: 0;
}
@media (max-width: 767.98px) {
  .grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    padding: 12px;
    gap: 10px 12px;
  }
  .disks {
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }
  .disk {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
  .p-btns .el-button {
    flex: 1 1 100%;
    min-height: 40px;
  }
}
</style>
