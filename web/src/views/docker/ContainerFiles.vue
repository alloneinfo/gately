<template>
  <div class="files">
    <div class="bar">
      <div class="crumbs">
        <button type="button" class="crumb" @click="go('/')">/</button>
        <template v-for="(c, i) in crumbs" :key="c.path">
          <button type="button" class="crumb" @click="go(c.path)">{{ c.name }}</button>
          <span v-if="i < crumbs.length - 1" class="sep">/</span>
        </template>
      </div>
      <div class="ops">
        <el-button :icon="ArrowUp" :disabled="path === '/'" @click="go(parent)">上级</el-button>
        <el-button :icon="RefreshCw" :loading="loading" @click="load">刷新</el-button>
        <el-button :icon="FolderPlus" @click="mkdir">新建文件夹</el-button>
        <el-button type="primary" :icon="Upload" :loading="uploading" @click="picker?.click()">上传</el-button>
        <input ref="picker" type="file" multiple class="hidden" @change="onPick" />
      </div>
    </div>

    <div v-loading="loading" class="list">
      <div v-for="f in entries" :key="f.path" class="item" @click="open(f)">
        <component :is="f.dir ? Folder : File" :size="20" class="ico" :class="{ dir: f.dir }" />
        <div class="main">
          <div class="name">{{ f.name }}<span v-if="f.link" class="link mono"> → {{ f.link }}</span></div>
          <div class="sub">{{ f.dir ? '' : formatBytes(f.size) + ' · ' }}{{ formatTime(f.modified) }}<span v-if="f.mode" class="mono"> · {{ f.mode }}</span></div>
        </div>
        <div class="acts" @click.stop>
          <template v-if="!isMobile">
            <el-button text :icon="Download" aria-label="下载" @click="downloadFile(f)" />
            <el-button text :icon="Pencil" aria-label="重命名" @click="rename(f)" />
            <el-button text type="danger" :icon="Trash2" aria-label="删除" @click="remove(f)" />
          </template>
          <el-dropdown v-else trigger="click" @command="(c: string) => onCmd(c, f)">
            <el-button text :icon="Ellipsis" class="touch" aria-label="更多" />
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="download" :icon="Download">下载</el-dropdown-item>
                <el-dropdown-item command="rename" :icon="Pencil">重命名</el-dropdown-item>
                <el-dropdown-item command="delete" :icon="Trash2" divided>删除</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
      <el-empty v-if="!loading && !entries.length" description="目录为空" :image-size="60" />
    </div>

    <el-dialog v-model="editVisible" :title="editPath" :width="'860px'" :fullscreen="isMobile" top="4vh" append-to-body>
      <el-alert v-if="truncated" type="warning" :closable="false" show-icon class="gap" title="文件超过 1MiB，仅显示前一部分，已禁止保存以免截断原文件" />
      <el-input v-model="content" type="textarea" :rows="isMobile ? 18 : 24" class="editor mono" spellcheck="false" :readonly="truncated" />
      <template #footer>
        <el-button @click="editVisible = false">关闭</el-button>
        <el-button type="primary" :icon="Save" :disabled="truncated" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowUp, Download, Ellipsis, File, Folder, FolderPlus, Pencil, RefreshCw, Save, Trash2, Upload } from 'lucide-vue-next'
import { api } from '../../api'
import { useIsMobile } from '../../composables/useIsMobile'
import { download, formatBytes, formatTime } from './format'
import type { FileEntry } from './types'

const props = defineProps<{ containerId: string }>()
const isMobile = useIsMobile()
const base = `/api/docker/containers/${props.containerId}/files`

const path = ref('/')
const entries = ref<FileEntry[]>([])
const loading = ref(false)
const uploading = ref(false)
const picker = ref<HTMLInputElement>()
const editVisible = ref(false)
const editPath = ref('')
const content = ref('')
const truncated = ref(false)
const saving = ref(false)

const crumbs = computed(() => {
  const parts = path.value.split('/').filter(Boolean)
  return parts.map((name, i) => ({ name, path: '/' + parts.slice(0, i + 1).join('/') }))
})
const parent = computed(() => {
  const parts = path.value.split('/').filter(Boolean)
  parts.pop()
  return '/' + parts.join('/')
})

function join(dir: string, name: string): string {
  return dir === '/' ? `/${name}` : `${dir}/${name}`
}

async function load() {
  loading.value = true
  try {
    entries.value = (await api.get<FileEntry[]>(`${base}/list?path=${encodeURIComponent(path.value)}`)) ?? []
  } catch (e) {
    ElMessage.error((e as Error).message)
    entries.value = []
  } finally {
    loading.value = false
  }
}

function go(p: string) {
  path.value = p || '/'
  load()
}

async function open(f: FileEntry) {
  if (f.dir) {
    go(f.path)
    return
  }
  try {
    const r = await api.get<{ content: string; size: number; truncated: boolean }>(`${base}/read?path=${encodeURIComponent(f.path)}`)
    content.value = r.content
    truncated.value = r.truncated
    editPath.value = f.path
    editVisible.value = true
  } catch (e) {
    ElMessage.error(`${(e as Error).message}（二进制文件请使用下载）`)
  }
}

async function save() {
  saving.value = true
  try {
    await api.post(`${base}/write`, { path: editPath.value, content: content.value })
    ElMessage.success('已保存')
    editVisible.value = false
    load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

function downloadFile(f: FileEntry) {
  download(`${base}/download?path=${encodeURIComponent(f.path)}`)
}

async function mkdir() {
  try {
    const { value } = await ElMessageBox.prompt('文件夹名称', '新建文件夹', { inputPattern: /^[^/]+$/, inputErrorMessage: '名称不能为空且不能包含 /' })
    await api.post(`${base}/mkdir`, { path: join(path.value, value) })
    load()
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error((e as Error).message)
    }
  }
}

async function rename(f: FileEntry) {
  try {
    const { value } = await ElMessageBox.prompt('新名称', '重命名', { inputValue: f.name, inputPattern: /^[^/]+$/, inputErrorMessage: '名称不能为空且不能包含 /' })
    await api.post(`${base}/rename`, { from: f.path, to: join(path.value, value) })
    load()
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error((e as Error).message)
    }
  }
}

async function remove(f: FileEntry) {
  try {
    await ElMessageBox.confirm(`确定删除 ${f.path}${f.dir ? '（含全部内容）' : ''}？该操作无法恢复。`, '删除', { type: 'warning' })
    await api.del(`${base}?path=${encodeURIComponent(f.path)}`)
    load()
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error((e as Error).message)
    }
  }
}

function onCmd(cmd: string, f: FileEntry) {
  if (cmd === 'download') {
    downloadFile(f)
  } else if (cmd === 'rename') {
    rename(f)
  } else {
    remove(f)
  }
}

async function onPick(ev: Event) {
  const input = ev.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  input.value = ''
  if (!files.length) {
    return
  }
  const fd = new FormData()
  fd.append('path', path.value)
  for (const f of files) {
    fd.append('file', f)
  }
  uploading.value = true
  try {
    await api.post(`${base}/upload`, fd)
    ElMessage.success(`已上传 ${files.length} 个文件`)
    load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    uploading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  margin-bottom: 10px;
}
.crumbs {
  flex: 1 1 240px;
  min-width: 0;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 2px;
  font-family: var(--g-mono);
  font-size: 13px;
}
.crumb {
  border: none;
  background: transparent;
  color: var(--el-color-primary);
  padding: 4px 4px;
  font: inherit;
  cursor: pointer;
  border-radius: 4px;
}
.crumb:hover {
  background: var(--el-fill-color-light);
}
.sep {
  color: var(--el-text-color-placeholder);
}
.ops {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.ops .el-button + .el-button {
  margin-left: 0;
}
.hidden {
  display: none;
}
.list {
  border: 1px solid var(--g-border);
  border-radius: var(--g-radius);
  min-height: 120px;
  max-height: 58vh;
  overflow-y: auto;
}
.item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 8px 8px 14px;
  border-bottom: 1px solid var(--g-border);
  cursor: pointer;
  min-height: 48px;
}
.item:last-child {
  border-bottom: none;
}
.item:hover {
  background: var(--el-fill-color-lighter);
}
.ico {
  flex-shrink: 0;
  color: var(--el-text-color-secondary);
}
.ico.dir {
  color: var(--el-color-warning);
}
.main {
  flex: 1;
  min-width: 0;
}
.name {
  font-size: 14px;
  word-break: break-all;
}
.link {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 1px;
}
.acts {
  flex-shrink: 0;
  display: flex;
}
.gap {
  margin-bottom: 8px;
}
.editor :deep(textarea) {
  font-family: var(--g-mono);
  font-size: 13px;
  line-height: 1.5;
  white-space: pre;
  overflow-x: auto;
}
@media (max-width: 767.98px) {
  .ops {
    width: 100%;
  }
  .ops .el-button {
    flex: 1 1 40%;
    min-height: 40px;
  }
  .list {
    max-height: calc(100dvh - 300px);
  }
  .item {
    min-height: 56px;
  }
  .touch {
    min-width: 40px;
    min-height: 40px;
  }
  .editor :deep(textarea) {
    font-size: 16px;
  }
}
</style>
