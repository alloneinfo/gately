<template>
  <div>
    <div class="page-header">
      <h2>计划任务</h2>
      <el-button type="primary" @click="openEditor()">添加任务</el-button>
    </div>
    <el-card shadow="never">
      <el-table :data="tasks" empty-text="暂无任务">
        <el-table-column label="启用" width="70">
          <template #default="{ row }">
            <el-switch :model-value="row.enable" size="small" @change="(v: string | number | boolean) => toggle(row, Boolean(v))" />
          </template>
        </el-table-column>
        <el-table-column label="任务" min-width="160">
          <template #default="{ row }">
            <div>{{ row.name }}</div>
            <el-tag size="small" effect="plain">{{ row.type === 'http' ? 'HTTP 请求' : '命令' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="计划" min-width="150">
          <template #default="{ row }">
            <div class="mono">{{ row.spec }}</div>
            <div class="meta" v-if="row.nextRun">下次 {{ new Date(row.nextRun).toLocaleString() }}</div>
          </template>
        </el-table-column>
        <el-table-column label="最近执行" min-width="220">
          <template #default="{ row }">
            <el-tag v-if="row.running" type="warning" size="small">执行中</el-tag>
            <template v-else-if="last(row)">
              <el-tag :type="last(row)!.ok ? 'success' : 'danger'" size="small">{{ last(row)!.ok ? '成功' : '失败' }}</el-tag>
              <span class="meta"> {{ new Date(last(row)!.start).toLocaleString() }} · {{ last(row)!.durationMs }}ms</span>
            </template>
            <span v-else class="meta">尚未执行</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220">
          <template #default="{ row }">
            <el-button link type="primary" :disabled="row.running" @click="runNow(row)">立即执行</el-button>
            <el-button link type="primary" @click="showHistory(row)">记录</el-button>
            <el-button link type="primary" @click="openEditor(row)">编辑</el-button>
            <el-popconfirm title="删除该任务？" @confirm="remove(row)">
              <template #reference>
                <el-button link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    <el-card shadow="never" header="日志" class="logs">
      <LogViewer url="/api/cron/logs" clearable />
    </el-card>

    <el-dialog v-model="editorVisible" :title="editing.key ? '编辑任务' : '添加任务'" width="680px">
      <el-form :model="editing" label-width="110px">
        <el-form-item label="名称"><el-input v-model="editing.name" /></el-form-item>
        <el-form-item label="计划">
          <el-input v-model="editing.spec" class="mono" placeholder="分 时 日 月 周，例如 0 3 * * *；可加秒字段，或 @every 10m" @blur="preview" />
          <div class="meta" v-if="nextRuns.length">接下来：{{ nextRuns.map((t) => new Date(t).toLocaleString()).join('；') }}</div>
          <div class="err" v-if="specError">{{ specError }}</div>
        </el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model="editing.type">
            <el-radio-button value="command">命令</el-radio-button>
            <el-radio-button value="http">HTTP 请求</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="editing.type === 'command'" label="命令">
          <el-input v-model="editing.command" type="textarea" :rows="4" class="mono" placeholder="以 sh -c 执行" />
        </el-form-item>
        <WebhookEditor v-else v-model="editing.http" :variables="['time', 'taskName']" />
        <el-form-item label="备注"><el-input v-model="editing.remark" /></el-form-item>
        <el-form-item label="启用"><el-switch v-model="editing.enable" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="historyVisible" title="执行记录" width="760px">
      <el-timeline>
        <el-timeline-item v-for="(r, i) in history" :key="i" :type="r.ok ? 'success' : 'danger'" :timestamp="new Date(r.start).toLocaleString() + ` · ${r.durationMs}ms` + (r.manual ? ' · 手动' : '')">
          <pre class="output">{{ r.output || '(无输出)' }}</pre>
        </el-timeline-item>
      </el-timeline>
      <el-empty v-if="!history.length" description="暂无记录" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import LogViewer from '../components/LogViewer.vue'
import WebhookEditor, { type WebhookConfig } from '../components/WebhookEditor.vue'

interface Run {
  start: string
  durationMs: number
  ok: boolean
  output: string
  manual: boolean
}

interface Task {
  key?: string
  name: string
  enable: boolean
  spec: string
  type: string
  command: string
  http: WebhookConfig
  remark: string
}

interface TaskView extends Task {
  key: string
  running: boolean
  nextRun?: string
  history: Run[]
}

const tasks = ref<TaskView[]>([])
const editorVisible = ref(false)
const editing = ref<Task>(empty())
const nextRuns = ref<string[]>([])
const specError = ref('')
const historyVisible = ref(false)
const history = ref<Run[]>([])
let timer: number | undefined

function empty(): Task {
  return {
    name: '',
    enable: true,
    spec: '0 3 * * *',
    type: 'command',
    command: '',
    http: { enable: true, url: '', method: 'GET', headers: [], body: '', successContent: [], timeout: 0, insecureSkipVerify: false },
    remark: '',
  }
}

function last(t: TaskView) {
  return t.history.length ? t.history[t.history.length - 1] : undefined
}

async function load() {
  tasks.value = await api.get('/api/cron/tasks')
}

async function preview() {
  try {
    const r = await api.post<{ next: string[] }>('/api/cron/parse', { spec: editing.value.spec })
    nextRuns.value = r.next
    specError.value = ''
  } catch (e) {
    nextRuns.value = []
    specError.value = (e as Error).message
  }
}

function openEditor(t?: TaskView) {
  if (t) {
    const { running: _r, nextRun: _n, history: _h, ...task } = t
    editing.value = JSON.parse(JSON.stringify(task))
  } else {
    editing.value = empty()
  }
  editorVisible.value = true
  preview()
}

async function save() {
  try {
    const { key, ...body } = editing.value
    if (key) {
      await api.put(`/api/cron/tasks/${key}`, body)
    } else {
      await api.post('/api/cron/tasks', body)
    }
    editorVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function toggle(t: TaskView, enable: boolean) {
  await api.put(`/api/cron/tasks/${t.key}/enable`, { enable })
  await load()
}

async function runNow(t: TaskView) {
  try {
    await api.post(`/api/cron/tasks/${t.key}/run`)
    ElMessage.success('已开始执行')
    window.setTimeout(load, 1000)
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

function showHistory(t: TaskView) {
  history.value = [...t.history].reverse()
  historyVisible.value = true
}

async function remove(t: TaskView) {
  await api.del(`/api/cron/tasks/${t.key}`)
  await load()
}

onMounted(() => {
  load()
  timer = window.setInterval(load, 5000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.err {
  font-size: 12px;
  color: var(--el-color-danger);
}
.output {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-size: 12px;
  max-height: 200px;
  overflow: auto;
}
.logs {
  margin-top: 16px;
}
</style>
