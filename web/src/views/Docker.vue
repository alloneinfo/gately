<template>
  <div>
    <div class="page-header">
      <h2>Docker 管理</h2>
      <div>
        <el-input v-model="keyword" placeholder="搜索容器或镜像" clearable style="width: 220px; margin-right: 10px" />
        <el-button @click="openSettings">设置</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>
    <el-alert v-if="error" type="error" :title="error" :closable="false" class="err" />
    <el-card shadow="never">
      <el-table :data="filtered" v-loading="loading" empty-text="没有容器">
        <el-table-column label="容器" min-width="180">
          <template #default="{ row }">
            <div>{{ row.name }}</div>
            <div class="meta" v-if="row.project">项目 {{ row.project }}</div>
          </template>
        </el-table-column>
        <el-table-column label="镜像" min-width="220">
          <template #default="{ row }">
            <span class="mono small">{{ row.image }}</span>
            <div v-if="row.update">
              <el-tag v-if="row.update.error" type="danger" size="small" :title="row.update.error">检查失败</el-tag>
              <el-tag v-else-if="row.update.hasUpdate" type="warning" size="small">有新版本</el-tag>
              <el-tag v-else type="success" size="small">已是最新</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="160">
          <template #default="{ row }">
            <el-tag :type="row.state === 'running' ? 'success' : 'info'" size="small">{{ row.state }}</el-tag>
            <div class="meta">{{ row.status }}</div>
          </template>
        </el-table-column>
        <el-table-column label="端口" min-width="150">
          <template #default="{ row }">
            <div v-for="p in row.ports" :key="p" class="mono small">{{ p }}</div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.state !== 'running'" link type="primary" @click="act(row, 'start')">启动</el-button>
            <el-button v-else link type="primary" @click="act(row, 'restart')">重启</el-button>
            <el-popconfirm v-if="row.state === 'running'" :title="`停止 ${row.name}？`" @confirm="act(row, 'stop')">
              <template #reference>
                <el-button link type="danger">停止</el-button>
              </template>
            </el-popconfirm>
            <el-button link type="primary" @click="showLogs(row)">日志</el-button>
            <el-button link type="primary" :loading="checking[row.id]" @click="check(row)">检查更新</el-button>
            <el-button v-if="row.update?.hasUpdate" link type="warning" :loading="row.busy || upgrading[row.id]" @click="upgrade(row)">升级</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    <el-card shadow="never" header="操作日志" class="logs">
      <LogViewer url="/api/docker/logs" clearable />
    </el-card>

    <el-dialog v-model="logsVisible" :title="`${logsName} 日志（最近 300 行）`" width="900px" top="5vh">
      <pre class="container-logs mono">{{ logs }}</pre>
      <template #footer>
        <el-button @click="refreshLogs">刷新</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="settingsVisible" title="Docker 设置" width="480px">
      <el-form label-width="90px">
        <el-form-item label="地址">
          <el-input v-model="host" placeholder="unix:///var/run/docker.sock" />
          <div class="meta">Gately 运行在容器中时需挂载 /var/run/docker.sock</div>
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
import { api } from '../api'
import LogViewer from '../components/LogViewer.vue'

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
.meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.small {
  font-size: 12px;
}
.err {
  margin-bottom: 14px;
}
.container-logs {
  max-height: 65vh;
  overflow: auto;
  background: var(--el-fill-color-darker);
  padding: 10px;
  border-radius: 6px;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}
.logs {
  margin-top: 16px;
}
</style>
