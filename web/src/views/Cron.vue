<template>
  <div>
    <PageHeader title="计划任务" desc="按 cron 表达式定时执行命令或发送 HTTP 请求" :icon="CalendarClock">
      <el-button type="primary" :icon="Plus" @click="openEditor()">添加任务</el-button>
    </PageHeader>

    <div class="card-list">
      <el-empty v-if="!tasks.length" description="暂无任务" />
      <EntityCard v-for="t in tasks" :key="t.key" :off="!t.enable">
        <template #lead>
          <el-switch :model-value="t.enable" @change="(v: string | number | boolean) => toggle(t, Boolean(v))" />
        </template>
        <template #title>
          <span>{{ t.name }}</span>
          <el-tag size="small" effect="plain">{{ t.type === 'http' ? 'HTTP 请求' : '命令' }}</el-tag>
        </template>
        <template v-if="t.remark" #sub>{{ t.remark }}</template>
        <template #status>
          <StatusBadge v-bind="taskStatus(t)" />
        </template>
        <Field label="计划" mono>{{ t.spec }}</Field>
        <Field label="下次执行">{{ t.enable && t.nextRun ? new Date(t.nextRun).toLocaleString() : '-' }}</Field>
        <Field label="最近执行">
          <template v-if="last(t)">
            <span :class="last(t)!.ok ? 'text-ok' : 'text-danger'">{{ last(t)!.ok ? '成功' : '失败' }}</span>
            <span class="meta"> · {{ new Date(last(t)!.start).toLocaleString() }} · {{ last(t)!.durationMs }}ms</span>
          </template>
          <span v-else class="muted">尚未执行</span>
        </Field>
        <Field :label="t.type === 'http' ? '请求' : '命令'" mono>
          <span class="ellipsis-2">{{ t.type === 'http' ? `${t.http.method} ${t.http.url}` : t.command }}</span>
        </Field>
        <template #actions>
          <el-button text size="small" :icon="Play" :disabled="t.running" @click="runNow(t)">立即执行</el-button>
          <el-button text size="small" :icon="History" @click="showHistory(t)">记录</el-button>
          <el-button text size="small" :icon="Pencil" @click="openEditor(t)">编辑</el-button>
          <el-popconfirm title="删除该任务？" @confirm="remove(t)">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Trash2">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </EntityCard>
    </div>

    <LogCard url="/api/cron/logs" clearable />

    <el-dialog v-model="editorVisible" :title="editing.key ? '编辑任务' : '添加任务'" width="680px" :fullscreen="isMobile">
      <el-form :model="editing" label-width="110px" :label-position="labelPosition">
        <el-form-item label="名称"><el-input v-model="editing.name" /></el-form-item>
        <el-form-item label="计划">
          <el-input v-model="editing.spec" class="mono" placeholder="分 时 日 月 周，例如 0 3 * * *；可加秒字段，或 @every 10m" @blur="preview" />
          <div v-if="nextRuns.length" class="form-help">接下来：{{ nextRuns.map((t) => new Date(t).toLocaleString()).join('；') }}</div>
          <div v-if="specError" class="form-help text-danger">{{ specError }}</div>
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

    <el-dialog v-model="historyVisible" title="执行记录" width="760px" :fullscreen="isMobile">
      <el-timeline v-if="history.length">
        <el-timeline-item
          v-for="(r, i) in history"
          :key="i"
          :type="r.ok ? 'success' : 'danger'"
          :timestamp="new Date(r.start).toLocaleString() + ` · ${r.durationMs}ms` + (r.manual ? ' · 手动' : '')"
        >
          <pre class="output mono">{{ r.output || '(无输出)' }}</pre>
        </el-timeline-item>
      </el-timeline>
      <el-empty v-else description="暂无记录" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { CalendarClock, History, Pencil, Play, Plus, Trash2 } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import EntityCard from '../components/EntityCard.vue'
import Field from '../components/Field.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import WebhookEditor, { type WebhookConfig } from '../components/WebhookEditor.vue'
import { useFormLayout } from '../composables/useIsMobile'

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
  history: Run[] | null
}

const { isMobile, labelPosition } = useFormLayout()
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
  const h = t.history ?? []
  return h.length ? h[h.length - 1] : undefined
}

// taskStatus 计算任务展示状态：停用的任务只显示“已停用”
function taskStatus(t: TaskView): { type: StatusType; text: string } {
  if (t.running) {
    return { type: 'pending', text: '执行中' }
  }
  if (!t.enable) {
    return { type: 'off', text: '已停用' }
  }
  const l = last(t)
  if (l && !l.ok) {
    return { type: 'error', text: '上次失败' }
  }
  return { type: 'running', text: '已排期' }
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
  try {
    await api.put(`/api/cron/tasks/${t.key}/enable`, { enable })
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
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
  history.value = [...(t.history ?? [])].reverse()
  historyVisible.value = true
}

async function remove(t: TaskView) {
  try {
    await api.del(`/api/cron/tasks/${t.key}`)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(() => {
  load()
  timer = window.setInterval(load, 5000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.ellipsis-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.output {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-size: 12px;
  max-height: 200px;
  overflow: auto;
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--g-border);
  border-radius: 6px;
  padding: 8px 10px;
}
</style>
