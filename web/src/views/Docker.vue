<template>
  <div>
    <PageHeader title="Docker" desc="管理本机容器，检查镜像更新并一键升级" :icon="Container">
      <el-input v-model="keyword" placeholder="搜索容器或镜像" clearable :prefix-icon="Search" class="search" />
      <el-button :icon="SlidersHorizontal" @click="openSettings">设置</el-button>
      <el-button :icon="RefreshCw" :loading="loading" @click="load">刷新</el-button>
    </PageHeader>

    <el-alert v-if="error" type="error" title="无法连接 Docker" :closable="false" show-icon class="err">
      <template #default>
        <div class="err-detail mono">{{ error }}</div>
        <div>请在「设置」中确认 Docker 地址；Gately 运行在容器中时需挂载 /var/run/docker.sock。</div>
      </template>
    </el-alert>

    <div v-loading="loading && !containers.length" class="card-list">
      <el-empty v-if="!loading && !error && !filtered.length" description="没有容器" />
      <EntityCard v-for="c in filtered" :key="c.id" :off="c.state !== 'running'">
        <template #lead>
          <div class="ct-icon" :class="{ on: c.state === 'running' }"><Box :size="18" /></div>
        </template>
        <template #title>
          <span>{{ c.name }}</span>
          <el-tag v-if="c.project" size="small" type="info">{{ c.project }}</el-tag>
        </template>
        <template #sub>{{ c.status }}</template>
        <template #status>
          <StatusBadge v-bind="ctStatus(c)" />
        </template>
        <Field label="镜像" mono>
          {{ c.image }}
          <div v-if="c.update" class="upd">
            <StatusBadge v-if="c.update.error" type="error" text="检查失败" :title="c.update.error" />
            <StatusBadge v-else-if="c.update.hasUpdate" type="warn" text="有新版本" />
            <StatusBadge v-else type="ok" text="已是最新" />
          </div>
        </Field>
        <Field label="端口" mono>
          <div v-for="p in c.ports ?? []" :key="p">{{ p }}</div>
          <span v-if="!(c.ports ?? []).length" class="muted">-</span>
        </Field>
        <template #actions>
          <el-button v-if="c.state !== 'running'" text size="small" type="primary" :icon="Play" @click="act(c, 'start')">启动</el-button>
          <el-button v-else text size="small" :icon="RotateCw" @click="act(c, 'restart')">重启</el-button>
          <el-popconfirm v-if="c.state === 'running'" :title="`停止 ${c.name}？`" @confirm="act(c, 'stop')">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Square">停止</el-button>
            </template>
          </el-popconfirm>
          <el-button text size="small" :icon="ScrollText" @click="showLogs(c)">日志</el-button>
          <el-button text size="small" :icon="SearchCheck" :loading="checking[c.id]" @click="check(c)">检查更新</el-button>
          <el-button v-if="c.update?.hasUpdate" text size="small" type="warning" :icon="ArrowUpCircle" :loading="c.busy || upgrading[c.id]" @click="upgrade(c)">升级</el-button>
        </template>
      </EntityCard>
    </div>

    <LogCard url="/api/docker/logs" title="操作日志" clearable />

    <el-dialog v-model="logsVisible" :title="`${logsName} 日志（最近 300 行）`" width="900px" top="5vh" :fullscreen="isMobile">
      <pre class="container-logs mono">{{ logs }}</pre>
      <template #footer>
        <el-button :icon="RefreshCw" @click="refreshLogs">刷新</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="settingsVisible" title="Docker 设置" width="500px" :fullscreen="isMobile">
      <el-form label-width="90px" :label-position="labelPosition">
        <el-form-item label="地址">
          <el-input v-model="host" placeholder="unix:///var/run/docker.sock" />
          <div class="form-help">Gately 运行在容器中时需挂载 /var/run/docker.sock</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="settingsVisible = false">取消</el-button>
        <el-button type="primary" @click="saveSettings">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowUpCircle, Box, Container, Play, RefreshCw, RotateCw, ScrollText, Search, SearchCheck, SlidersHorizontal, Square } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import EntityCard from '../components/EntityCard.vue'
import Field from '../components/Field.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import { useFormLayout } from '../composables/useIsMobile'

interface ContainerView {
  id: string
  name: string
  image: string
  state: string
  status: string
  ports: string[] | null
  project?: string
  busy: boolean
  update?: { checked: string; hasUpdate: boolean; error?: string }
}

const { isMobile, labelPosition } = useFormLayout()
const containers = ref<ContainerView[]>([])
const loading = ref(false)
const error = ref('')
const keyword = ref('')
const checking = reactive<Record<string, boolean>>({})
const upgrading = reactive<Record<string, boolean>>({})
const logsVisible = ref(false)
const logs = ref('')
const logsName = ref('')
let logsID = ''
const settingsVisible = ref(false)
const host = ref('')

const filtered = computed(() => {
  const k = keyword.value.trim().toLowerCase()
  return k ? containers.value.filter((c) => c.name.toLowerCase().includes(k) || c.image.toLowerCase().includes(k)) : containers.value
})

function ctStatus(c: ContainerView): { type: StatusType; text: string } {
  const map: Record<string, { type: StatusType; text: string }> = {
    running: { type: 'running', text: '运行中' },
    exited: { type: 'off', text: '已停止' },
    created: { type: 'off', text: '已创建' },
    paused: { type: 'warn', text: '已暂停' },
    restarting: { type: 'pending', text: '重启中' },
    dead: { type: 'error', text: '异常' },
  }
  return map[c.state] ?? { type: 'info', text: c.state }
}

async function load() {
  loading.value = true
  try {
    containers.value = await api.get('/api/docker/containers')
    error.value = ''
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

async function act(c: ContainerView, action: string) {
  try {
    await api.post(`/api/docker/containers/${c.id}/${action}`)
    ElMessage.success('已执行')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function showLogs(c: ContainerView) {
  logsID = c.id
  logsName.value = c.name
  logs.value = '加载中…'
  logsVisible.value = true
  await refreshLogs()
}

async function refreshLogs() {
  try {
    const r = await api.get<{ logs: string }>(`/api/docker/containers/${logsID}/logs?tail=300`)
    logs.value = r.logs || '(无日志)'
  } catch (e) {
    logs.value = (e as Error).message
  }
}

async function check(c: ContainerView) {
  checking[c.id] = true
  try {
    await api.post(`/api/docker/containers/${c.id}/check`)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    checking[c.id] = false
  }
}

async function upgrade(c: ContainerView) {
  await ElMessageBox.confirm(`将拉取最新镜像并按原配置重建 ${c.name}，期间服务会短暂中断；新容器启动失败时自动恢复旧容器。`, '升级容器', { type: 'warning' })
  upgrading[c.id] = true
  try {
    await api.post(`/api/docker/containers/${c.id}/upgrade`)
    ElMessage.success('升级完成')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    upgrading[c.id] = false
  }
}

async function openSettings() {
  const s = await api.get<{ host: string }>('/api/docker/settings')
  host.value = s.host
  settingsVisible.value = true
}

async function saveSettings() {
  try {
    await api.put('/api/docker/settings', { host: host.value })
    settingsVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(load)
</script>

<style scoped>
.search {
  width: 220px;
}
.err {
  margin-bottom: 16px;
}
.err-detail {
  font-size: 12px;
  word-break: break-all;
  margin: 2px 0 4px;
  opacity: 0.85;
}
.ct-icon {
  width: 34px;
  height: 34px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}
.ct-icon.on {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}
.upd {
  margin-top: 4px;
}
.container-logs {
  max-height: 65vh;
  overflow: auto;
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--g-border);
  padding: 10px;
  border-radius: 8px;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}
@media (max-width: 767.98px) {
  .search {
    width: 100%;
  }
  .container-logs {
    max-height: none;
  }
}
</style>
