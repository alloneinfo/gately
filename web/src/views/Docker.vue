<template>
  <div>
    <PageHeader title="Docker" desc="管理容器、Compose 项目、镜像、网络与卷，检查镜像更新并一键升级" :icon="Container">
      <el-button :icon="RefreshCw" :loading="refreshing" @click="refresh">刷新</el-button>
    </PageHeader>

    <el-alert v-if="dockerError" type="error" title="无法连接 Docker" :closable="false" show-icon class="err">
      <template #default>
        <div class="err-detail mono">{{ dockerError }}</div>
        <div>请在「系统」标签的设置中确认 Docker 地址；Gately 运行在容器中时需挂载 /var/run/docker.sock。</div>
        <el-button v-if="tab !== 'system'" size="small" class="err-btn" :icon="SlidersHorizontal" @click="tab = 'system'">打开设置</el-button>
      </template>
    </el-alert>

    <el-tabs v-model="tab" class="docker-tabs">
      <el-tab-pane label="容器" name="containers" lazy><ContainersTab :ref="(el: unknown) => setRef('containers', el)" /></el-tab-pane>
      <el-tab-pane label="Compose" name="compose" lazy><ComposeTab :ref="(el: unknown) => setRef('compose', el)" /></el-tab-pane>
      <el-tab-pane label="镜像" name="images" lazy><ImagesTab :ref="(el: unknown) => setRef('images', el)" /></el-tab-pane>
      <el-tab-pane label="网络" name="networks" lazy><NetworksTab :ref="(el: unknown) => setRef('networks', el)" /></el-tab-pane>
      <el-tab-pane label="卷" name="volumes" lazy><VolumesTab :ref="(el: unknown) => setRef('volumes', el)" /></el-tab-pane>
      <el-tab-pane label="系统" name="system" lazy><SystemTab :ref="(el: unknown) => setRef('system', el)" @saved="loadInfo" /></el-tab-pane>
    </el-tabs>

    <TaskPanel />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Container, RefreshCw, SlidersHorizontal } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import ComposeTab from './docker/ComposeTab.vue'
import ContainersTab from './docker/ContainersTab.vue'
import ImagesTab from './docker/ImagesTab.vue'
import NetworksTab from './docker/NetworksTab.vue'
import SystemTab from './docker/SystemTab.vue'
import TaskPanel from './docker/TaskPanel.vue'
import VolumesTab from './docker/VolumesTab.vue'
import { dockerError, dockerInfo, reportError } from './docker/state'
import { resumeRunningTasks } from './docker/tasks'
import type { DockerInfo } from './docker/types'

interface Reloadable {
  reload?: () => unknown
}

const tab = ref('containers')
const refreshing = ref(false)
// 各标签页实例，用于顶部“刷新”按钮触发当前标签重新加载
const refs: Record<string, Reloadable | null> = {}

function setRef(name: string, el: unknown) {
  refs[name] = el as Reloadable | null
}

async function loadInfo() {
  try {
    dockerInfo.value = await api.get<DockerInfo>('/api/docker/info')
    reportError('')
  } catch (e) {
    reportError((e as Error).message)
  }
}

async function refresh() {
  refreshing.value = true
  try {
    await loadInfo()
    await refs[tab.value]?.reload?.()
  } finally {
    refreshing.value = false
  }
}

onMounted(() => {
  loadInfo()
  resumeRunningTasks()
})
</script>

<style scoped>
.err {
  margin-bottom: 16px;
}
.err-detail {
  font-size: 12px;
  word-break: break-all;
  margin: 2px 0 4px;
  opacity: 0.85;
}
.err-btn {
  margin-top: 8px;
}
.docker-tabs :deep(.el-tabs__header) {
  margin-bottom: 14px;
}
.docker-tabs :deep(.el-tabs__nav-wrap) {
  overflow-x: auto;
}
.docker-tabs :deep(.el-tabs__nav-wrap::after) {
  background-color: var(--g-border);
}
@media (max-width: 767.98px) {
  .docker-tabs :deep(.el-tabs__item) {
    min-height: 44px;
    line-height: 44px;
    padding: 0 14px;
  }
}
</style>
