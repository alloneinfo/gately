<template>
  <div>
    <div class="g-card">
      <div class="g-card-head">
        <div class="g-card-title"><Server :size="16" />Docker 信息</div>
        <el-button size="small" :icon="RefreshCw" :loading="loading" @click="load">刷新</el-button>
      </div>
      <div v-if="info" class="grid">
        <Field label="版本">{{ info.version }}</Field>
        <Field label="API 版本">{{ info.apiVersion }}</Field>
        <Field label="系统 / 架构">{{ info.os }} / {{ info.arch }}</Field>
        <Field label="内核" mono>{{ info.kernel }}</Field>
        <Field label="主机名">{{ info.name }}</Field>
        <Field label="CPU / 内存">{{ info.cpus }} 核 / {{ formatBytes(info.memTotal) }}</Field>
        <Field label="存储驱动">{{ info.storageDriver }}</Field>
        <Field label="数据目录" mono>{{ info.rootDir }}</Field>
        <Field label="容器">共 {{ info.containers.total }}，运行 {{ info.containers.running }}，暂停 {{ info.containers.paused }}，停止 {{ info.containers.stopped }}</Field>
        <Field label="镜像">{{ info.images }}</Field>
        <Field label="Compose CLI">{{ info.composeCli ? info.composeCliVersion || '可用' : '未安装' }}</Field>
      </div>
      <el-empty v-else-if="!loading" description="无法获取 Docker 信息" :image-size="50" />
    </div>

    <div class="g-card">
      <div class="g-card-head">
        <div class="g-card-title"><HardDrive :size="16" />磁盘占用</div>
      </div>
      <div v-if="disk" class="g-card-body disks">
        <div v-for="d in diskItems" :key="d.key" class="disk">
          <RingGauge :value="pct(d.item)" :size="64" :stroke="7" />
          <div class="d-main">
            <div class="d-name">{{ d.name }} <span class="muted">{{ d.item.count }} 项</span></div>
            <div class="d-size">{{ formatBytes(d.item.size) }}</div>
            <div class="d-rec muted">可回收 {{ formatBytes(d.item.reclaimable) }}</div>
          </div>
        </div>
      </div>
      <div class="g-card-body prune">
        <div class="p-title">清理</div>
        <div class="p-btns">
          <el-button v-for="p in prunes" :key="p.label" :icon="Eraser" :loading="pruning === p.label" @click="prune(p)">{{ p.label }}</el-button>
        </div>
      </div>
    </div>

    <SettingsPanel @saved="onSaved" />
    <TaskHistory ref="history" />
    <LogCard url="/api/docker/logs" title="操作日志" clearable />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
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

const emit = defineEmits<{ saved: [] }>()
const info = dockerInfo
const disk = ref<DiskUsage>()
const loading = ref(false)
const pruning = ref('')
const history = ref<InstanceType<typeof TaskHistory>>()

interface PruneDef {
  label: string
  target: string
  all?: boolean
  tip: string
}

const prunes: PruneDef[] = [
  { label: '清理已停止容器', target: 'containers', tip: '删除所有已停止的容器。' },
  { label: '清理悬空镜像', target: 'images', tip: '删除所有无标签且未被使用的镜像。' },
  { label: '清理全部未使用镜像', target: 'images', all: true, tip: '删除所有未被任何容器使用的镜像，之后使用需重新拉取。' },
  { label: '清理未使用网络', target: 'networks', tip: '删除所有未被容器使用的自定义网络。' },
  { label: '清理未使用卷', target: 'volumes', tip: '删除所有未被容器使用的卷，卷内数据将永久丢失。' },
  { label: '清理构建缓存', target: 'buildcache', tip: '删除镜像构建缓存。' },
]

const diskItems = computed(() => {
  const d = disk.value
  if (!d) {
    return []
  }
  return [
    { key: 'images', name: '镜像', item: d.images },
    { key: 'containers', name: '容器', item: d.containers },
    { key: 'volumes', name: '卷', item: d.volumes },
    { key: 'buildCache', name: '构建缓存', item: d.buildCache },
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
    await ElMessageBox.confirm(p.tip + '确定继续？', p.label, { type: 'warning' })
  } catch {
    return
  }
  pruning.value = p.label
  try {
    const r = await api.post<{ deleted: number; spaceReclaimed: number }>('/api/docker/prune', { target: p.target, all: p.all })
    ElMessage.success(`已清理 ${r.deleted} 项，释放 ${formatBytes(r.spaceReclaimed)}`)
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
