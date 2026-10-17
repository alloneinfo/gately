<template>
  <el-card shadow="never" class="bk">
    <template #header>
      <div class="head">
        <span class="title"><CalendarClock :size="16" />定时备份</span>
        <el-switch :model-value="settings.enable" :loading="saving" @change="(v: string | number | boolean) => toggle(Boolean(v))" />
      </div>
    </template>

    <el-form :model="settings" label-width="110px" :label-position="labelPosition">
      <el-form-item label="执行计划">
        <el-input v-model="settings.schedule" class="mono" placeholder="0 3 * * *" />
        <div class="form-help">5 段 cron 表达式，如 0 3 * * * 为每天 03:00；也可用 @daily、@weekly</div>
      </el-form-item>
      <el-form-item label="保留数量">
        <el-input-number v-model="settings.keep" :min="1" :max="365" />
        <span class="inline-help">每个目的地各保留最新的 N 份</span>
      </el-form-item>
      <el-form-item label="加密口令">
        <el-input v-model="settings.password" show-password :placeholder="settings.hasPassword ? '已设置，留空保持不变' : '留空不加密'" />
        <div class="form-help">设置后备份文件为 .zip.enc，恢复时需输入同一口令；口令丢失后无法解密</div>
      </el-form-item>
      <el-form-item label="本地目录">
        <el-switch v-model="settings.localEnable" />
        <span class="inline-help">写入配置目录下的 backups 文件夹</span>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="saving" @click="save">保存设置</el-button>
        <el-button :icon="Play" :loading="running" @click="runNow">立即备份</el-button>
      </el-form-item>
    </el-form>

    <div class="sub">
      <span>远程目的地</span>
      <el-button text size="small" :icon="Plus" @click="openTarget()">添加</el-button>
    </div>
    <el-empty v-if="!targets.length" :image-size="40" description="可选：把备份同步到存储管理中的 WebDAV 或其他本地目录" />
    <div v-for="t in targets" :key="t.key" class="target">
      <el-switch :model-value="t.enable" size="small" @change="(v: string | number | boolean) => toggleTarget(t, Boolean(v))" />
      <div class="t-main">
        <div class="t-name">{{ t.name }}</div>
        <div class="t-sub mono">{{ mountName(t.mount) }}:/{{ t.dir }}</div>
      </div>
      <el-button text size="small" :icon="Pencil" @click="openTarget(t)" />
      <el-popconfirm title="删除该目的地？已写入的备份不会被删除" @confirm="removeTarget(t)">
        <template #reference>
          <el-button text size="small" type="danger" :icon="Trash2" />
        </template>
      </el-popconfirm>
    </div>

    <div class="sub"><span>最近一次</span></div>
    <div v-if="status.lastRun" class="last">
      <div class="meta">
        {{ new Date(status.lastRun).toLocaleString() }}（{{ status.manual ? '手动' : '定时' }}）
        <span v-if="status.nextRun">，下次 {{ new Date(status.nextRun).toLocaleString() }}</span>
      </div>
      <div v-for="r in status.results" :key="r.target" class="res">
        <StatusBadge :type="r.ok ? 'running' : 'error'" :text="r.ok ? '成功' : '失败'" />
        <span class="r-name">{{ r.target }}</span>
        <span v-if="r.error" class="r-err">{{ r.error }}</span>
        <span v-else-if="r.pruned" class="meta">清理旧备份 {{ r.pruned }} 份</span>
      </div>
    </div>
    <div v-else class="meta">尚未执行过备份<span v-if="status.nextRun">，下次 {{ new Date(status.nextRun).toLocaleString() }}</span></div>

    <div v-if="localFiles.length" class="sub"><span>本地备份文件</span></div>
    <div v-for="f in localFiles" :key="f.name" class="file">
      <span class="mono f-name">{{ f.name }}</span>
      <span class="meta">{{ formatSize(f.size) }} · {{ new Date(f.modTime).toLocaleString() }}</span>
      <a :href="`/api/backup/local/${encodeURIComponent(f.name)}`" download><el-button text size="small" :icon="Download" /></a>
    </div>

    <el-dialog v-model="targetVisible" :title="editingTarget.key ? '编辑目的地' : '添加目的地'" width="460px" :fullscreen="isMobile" append-to-body>
      <el-form :model="editingTarget" label-width="90px" :label-position="labelPosition">
        <el-form-item label="名称"><el-input v-model="editingTarget.name" /></el-form-item>
        <el-form-item label="存储挂载">
          <el-select v-model="editingTarget.mount" style="width: 100%" placeholder="选择存储管理中的挂载点">
            <el-option v-for="m in mounts" :key="m.key" :label="`${m.name}（${m.type === 'webdav' ? 'WebDAV' : '本地'}）`" :value="m.key" :disabled="m.readOnly || !m.enable" />
          </el-select>
          <div v-if="!mounts.length" class="form-help">还没有挂载点，请先到「存储管理」添加</div>
        </el-form-item>
        <el-form-item label="目录"><el-input v-model="editingTarget.dir" placeholder="如 gately-backups，留空为根目录" /></el-form-item>
        <el-form-item label="启用"><el-switch v-model="editingTarget.enable" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="targetVisible = false">取消</el-button>
        <el-button type="primary" @click="saveTarget">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { CalendarClock, Download, Pencil, Play, Plus, Trash2 } from 'lucide-vue-next'
import { api } from '../api'
import StatusBadge from './StatusBadge.vue'
import { useFormLayout } from '../composables/useIsMobile'

interface Settings {
  enable: boolean
  schedule: string
  keep: number
  password: string
  hasPassword?: boolean
  localEnable: boolean
}
interface Target {
  key?: string
  name: string
  enable: boolean
  mount: string
  dir: string
}
interface Mount {
  key: string
  name: string
  type: string
  enable: boolean
  readOnly: boolean
}
interface Result {
  target: string
  ok: boolean
  file?: string
  error?: string
  pruned?: number
}
interface Status {
  running: boolean
  lastRun?: string
  nextRun?: string
  manual: boolean
  results: Result[]
}
interface LocalFile {
  name: string
  size: number
  modTime: string
}

const { isMobile, labelPosition } = useFormLayout()
const settings = reactive<Settings>({ enable: false, schedule: '0 3 * * *', keep: 7, password: '', localEnable: true })
const targets = ref<Target[]>([])
const mounts = ref<Mount[]>([])
const status = ref<Status>({ running: false, manual: false, results: [] })
const localFiles = ref<LocalFile[]>([])
const saving = ref(false)
const running = ref(false)
const targetVisible = ref(false)
const editingTarget = ref<Target>(emptyTarget())
let timer: number | undefined

function emptyTarget(): Target {
  return { name: '', enable: true, mount: '', dir: 'gately-backups' }
}

function mountName(key: string) {
  return mounts.value.find((m) => m.key === key)?.name ?? '（挂载点已删除）'
}

function formatSize(n: number) {
  if (n < 1024) {
    return `${n} B`
  }
  if (n < 1024 * 1024) {
    return `${(n / 1024).toFixed(1)} KB`
  }
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}

async function loadSettings() {
  Object.assign(settings, await api.get<Settings>('/api/backup/settings'))
}

async function loadLive() {
  const [t, s, f] = await Promise.all([
    api.get<Target[]>('/api/backup/targets'),
    api.get<Status>('/api/backup/status'),
    api.get<LocalFile[]>('/api/backup/local'),
  ])
  targets.value = t
  status.value = s
  localFiles.value = f
}

async function put(body: Settings) {
  saving.value = true
  try {
    Object.assign(settings, await api.put<Settings>('/api/backup/settings', body))
    await loadLive()
    return true
  } catch (e) {
    ElMessage.error((e as Error).message)
    return false
  } finally {
    saving.value = false
  }
}

async function save() {
  if (await put({ ...settings })) {
    ElMessage.success('已保存')
  }
}

async function toggle(enable: boolean) {
  await put({ ...settings, enable })
}

async function runNow() {
  running.value = true
  try {
    status.value = await api.post<Status>('/api/backup/run')
    const failed = status.value.results.filter((r) => !r.ok).length
    if (failed) {
      ElMessage.error(`${failed} 个目的地备份失败`)
    } else {
      ElMessage.success('备份完成')
    }
    await loadLive()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    running.value = false
  }
}

function openTarget(t?: Target) {
  editingTarget.value = t ? { ...t } : emptyTarget()
  targetVisible.value = true
}

async function saveTarget() {
  try {
    const { key, ...body } = editingTarget.value
    if (key) {
      await api.put(`/api/backup/targets/${key}`, body)
    } else {
      await api.post('/api/backup/targets', body)
    }
    targetVisible.value = false
    await loadLive()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function toggleTarget(t: Target, enable: boolean) {
  try {
    await api.put(`/api/backup/targets/${t.key}/enable`, { enable })
    await loadLive()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function removeTarget(t: Target) {
  try {
    await api.del(`/api/backup/targets/${t.key}`)
    await loadLive()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(async () => {
  try {
    mounts.value = await api.get<Mount[]>('/api/backup/mounts')
    await loadSettings()
    await loadLive()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
  timer = window.setInterval(() => loadLive().catch(() => {}), 15000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.bk {
  margin-bottom: 16px;
}
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
}
.inline-help {
  margin-left: 10px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.sub {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 14px 0 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-regular);
}
.target,
.file,
.res {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 0;
}
.t-main {
  flex: 1;
  min-width: 0;
}
.t-name {
  font-size: 13px;
}
.t-sub,
.meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.r-err {
  color: var(--el-color-danger);
  font-size: 12px;
  word-break: break-all;
}
.f-name {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
