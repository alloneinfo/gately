<template>
  <div>
    <div class="toolbar">
      <el-input v-model="keyword" placeholder="搜索项目或服务" clearable :prefix-icon="Search" class="t-search" />
      <div class="t-ops">
        <el-button type="primary" :icon="Plus" @click="newProject">新建项目</el-button>
        <el-button :icon="RefreshCw" :loading="loading" @click="load">刷新</el-button>
      </div>
    </div>
    <el-alert v-if="info && !info.composeCli" type="warning" :closable="false" show-icon class="gap" title="未检测到 docker compose 命令">
      启动、停止、重启、down 可正常使用；up、拉取镜像与编辑 Compose 文件依赖 docker compose CLI，请在 Gately 所在环境安装 Docker Compose 插件。
    </el-alert>

    <div v-loading="loading && !projects.length" class="card-list">
      <el-empty v-if="!loading && !filtered.length" description="没有 Compose 项目" />
      <EntityCard v-for="p in filtered" :key="p.name" :off="p.status === 'stopped'">
        <template #lead>
          <div class="ico" :class="{ on: p.status !== 'stopped' }"><Layers :size="18" /></div>
        </template>
        <template #title>
          <span>{{ p.name }}</span>
          <el-tag v-if="p.managed" size="small" type="success">Gately 管理</el-tag>
        </template>
        <template #sub>{{ p.workingDir || '-' }}</template>
        <template #status>
          <StatusBadge v-bind="projBadge(p)" />
        </template>
        <Field label="容器" wide>{{ p.running }} / {{ p.total }} 运行中</Field>
        <Field label="服务" wide>
          <div v-for="s in p.services" :key="s.name" class="svc">
            <span class="svc-dot" :class="s.state === 'running' ? 'on' : ''" />
            <b>{{ s.name }}</b>
            <span class="muted mono svc-img">{{ s.image }}</span>
            <span v-for="port in s.ports ?? []" :key="port" class="mono svc-port">{{ port }}</span>
          </div>
          <span v-if="!p.services?.length" class="muted">-</span>
        </Field>
        <template #actions>
          <el-button v-if="p.status !== 'running'" text type="primary" :icon="Play" class="btn" :loading="busy[p.name]" @click="simple(p, 'start')">启动</el-button>
          <el-button v-if="p.status !== 'stopped'" text type="danger" :icon="Square" class="btn" :loading="busy[p.name]" @click="stop(p)">停止</el-button>
          <el-button text :icon="RotateCw" class="btn" :loading="busy[p.name]" @click="simple(p, 'restart')">重启</el-button>
          <el-button text :icon="ScrollText" class="btn" @click="showLogs(p)">日志</el-button>
          <el-dropdown trigger="click" @command="(c: string) => onCommand(p, c)">
            <el-button text :icon="Ellipsis" class="btn">更多</el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="up" :icon="Rocket">up（创建并启动）</el-dropdown-item>
                <el-dropdown-item command="pull" :icon="CloudDownload">拉取镜像</el-dropdown-item>
                <el-dropdown-item command="edit" :icon="FilePen">编辑 Compose 文件</el-dropdown-item>
                <el-dropdown-item command="down" :icon="ArrowDownToLine" divided>down（删除容器）</el-dropdown-item>
                <el-dropdown-item v-if="p.managed" command="delete" :icon="Trash2">删除项目</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
      </EntityCard>
    </div>

    <ComposeEditor v-model="editorVisible" :project="editorProject" @done="load" />

    <el-dialog v-model="logsVisible" :title="`${logsName} 汇总日志`" width="900px" :fullscreen="isMobile" top="5vh" append-to-body>
      <pre class="logs mono">{{ logs }}</pre>
      <template #footer>
        <el-button :icon="RefreshCw" @click="refreshLogs">刷新</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="downVisible" title="down 项目" width="440px" append-to-body>
      <p class="dlg-text">将停止并删除项目 <b>{{ downTarget?.name }}</b> 的全部容器与网络，镜像和命名卷默认保留。</p>
      <el-checkbox v-model="downVolumes">同时删除命名卷（数据将丢失）</el-checkbox>
      <template #footer>
        <el-button @click="downVisible = false">取消</el-button>
        <el-button type="danger" :loading="downBusy" @click="doDown">down</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDownToLine, CloudDownload, Ellipsis, FilePen, Layers, Play, Plus, RefreshCw, Rocket, RotateCw, ScrollText, Search, Square, Trash2 } from 'lucide-vue-next'
import { api, ApiError } from '../../api'
import EntityCard from '../../components/EntityCard.vue'
import Field from '../../components/Field.vue'
import StatusBadge, { type StatusType } from '../../components/StatusBadge.vue'
import { useIsMobile } from '../../composables/useIsMobile'
import ComposeEditor from './ComposeEditor.vue'
import { dockerInfo, reportError } from './state'
import { trackTask } from './tasks'
import type { ComposeProject } from './types'

const isMobile = useIsMobile()
const info = dockerInfo
const projects = ref<ComposeProject[]>([])
const loading = ref(false)
const keyword = ref('')
const busy = reactive<Record<string, boolean>>({})
const editorVisible = ref(false)
const editorProject = ref<string>()
const logsVisible = ref(false)
const logsName = ref('')
const logs = ref('')
const downVisible = ref(false)
const downTarget = ref<ComposeProject>()
const downVolumes = ref(false)
const downBusy = ref(false)

const filtered = computed(() => {
  const k = keyword.value.trim().toLowerCase()
  return k ? projects.value.filter((p) => p.name.toLowerCase().includes(k) || (p.services ?? []).some((s) => s.name.toLowerCase().includes(k))) : projects.value
})

function projBadge(p: ComposeProject): { type: StatusType; text: string } {
  if (p.status === 'running') {
    return { type: 'running', text: '运行中' }
  }
  if (p.status === 'partial') {
    return { type: 'warn', text: '部分运行' }
  }
  return { type: 'off', text: '已停止' }
}

// 失败时对 501（缺少 compose CLI）给出友好提示
function fail(e: unknown) {
  if (e instanceof ApiError && e.status === 501) {
    ElMessageBox.alert(`${e.message}\n\n此功能依赖 docker compose CLI，请在 Gately 所在环境安装 Docker Compose 插件，并确认能访问 Compose 文件所在目录。`, '当前环境不支持', { type: 'warning' })
    return
  }
  ElMessage.error((e as Error).message)
}

function needCli(): boolean {
  if (info.value && !info.value.composeCli) {
    ElMessageBox.alert('此功能依赖 docker compose CLI，当前环境未检测到。请安装 Docker Compose 插件后重试。', '当前环境不支持', { type: 'warning' })
    return false
  }
  return true
}

async function load() {
  loading.value = true
  try {
    projects.value = (await api.get<ComposeProject[]>('/api/docker/compose/projects')) ?? []
    reportError('')
  } catch (e) {
    reportError((e as Error).message)
  } finally {
    loading.value = false
  }
}

const url = (p: ComposeProject, tail = '') => `/api/docker/compose/${encodeURIComponent(p.name)}${tail}`

async function simple(p: ComposeProject, action: string) {
  busy[p.name] = true
  try {
    await api.post(url(p, `/${action}`))
    ElMessage.success('已执行')
    await load()
  } catch (e) {
    fail(e)
  } finally {
    busy[p.name] = false
  }
}

async function stop(p: ComposeProject) {
  try {
    await ElMessageBox.confirm(`停止项目 ${p.name} 的全部容器？`, '停止项目', { type: 'warning' })
  } catch {
    return
  }
  await simple(p, 'stop')
}

async function runTask(p: ComposeProject, action: string, label: string) {
  if (!needCli()) {
    return
  }
  try {
    const r = await api.post<{ task: string }>(url(p, `/${action}`))
    trackTask(r.task, `${label} ${p.name}`, () => load())
  } catch (e) {
    fail(e)
  }
}

async function onCommand(p: ComposeProject, cmd: string) {
  if (cmd === 'up') {
    await runTask(p, 'up', 'up')
  } else if (cmd === 'pull') {
    await runTask(p, 'pull', '拉取镜像')
  } else if (cmd === 'edit') {
    if (needCli()) {
      editorProject.value = p.name
      editorVisible.value = true
    }
  } else if (cmd === 'down') {
    downTarget.value = p
    downVolumes.value = false
    downVisible.value = true
  } else if (cmd === 'delete') {
    try {
      await ElMessageBox.confirm(`将 down 项目 ${p.name} 并删除其 Compose 目录，无法恢复。`, '删除项目', { type: 'warning' })
      await api.del(url(p))
      ElMessage.success('已删除')
      await load()
    } catch (e) {
      if (e !== 'cancel' && e !== 'close') {
        fail(e)
      }
    }
  }
}

async function doDown() {
  if (!downTarget.value) {
    return
  }
  downBusy.value = true
  try {
    await api.post(url(downTarget.value, `/down${downVolumes.value ? '?volumes=1' : ''}`))
    downVisible.value = false
    ElMessage.success('已 down')
    await load()
  } catch (e) {
    fail(e)
  } finally {
    downBusy.value = false
  }
}

function newProject() {
  if (!needCli()) {
    return
  }
  editorProject.value = undefined
  editorVisible.value = true
}

let logsProject: ComposeProject | undefined
async function showLogs(p: ComposeProject) {
  logsProject = p
  logsName.value = p.name
  logs.value = '加载中…'
  logsVisible.value = true
  await refreshLogs()
}

async function refreshLogs() {
  if (!logsProject) {
    return
  }
  try {
    const r = await api.get<{ logs: string }>(url(logsProject, '/logs?tail=300'))
    logs.value = r.logs || '(无日志)'
  } catch (e) {
    logs.value = (e as Error).message
  }
}

defineExpose({ reload: load })
onMounted(load)
</script>

<style scoped>
.toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  margin-bottom: 12px;
}
.t-search {
  width: 240px;
}
.t-ops {
  display: flex;
  gap: 8px;
  margin-left: auto;
}
.gap {
  margin-bottom: 12px;
}
.ico {
  width: 34px;
  height: 34px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}
.ico.on {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}
.svc {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 2px 8px;
  padding: 2px 0;
}
.svc-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--el-text-color-placeholder);
  flex-shrink: 0;
}
.svc-dot.on {
  background: var(--el-color-success);
}
.svc-img,
.svc-port {
  font-size: 12px;
}
.svc-port {
  color: var(--el-text-color-secondary);
}
.logs {
  margin: 0;
  max-height: 65vh;
  overflow: auto;
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--g-border);
  padding: 10px;
  border-radius: 8px;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
}
.dlg-text {
  margin: 0 0 12px;
  line-height: 1.6;
  word-break: break-all;
}
@media (max-width: 767.98px) {
  .t-search {
    width: 100%;
  }
  .t-ops {
    width: 100%;
    margin-left: 0;
  }
  .t-ops .el-button {
    flex: 1;
    min-height: 40px;
  }
  .btn {
    min-height: 40px;
    padding: 0 8px;
  }
  .logs {
    max-height: none;
  }
  .toolbar :deep(input) {
    font-size: 16px;
  }
}
</style>
