<template>
  <div>
    <PageHeader title="存储管理" desc="登记本地目录或远程 WebDAV 为挂载点，在浏览器中管理文件" :icon="HardDrive">
      <el-button type="primary" :icon="Plus" @click="openEditor()">添加挂载点</el-button>
    </PageHeader>

    <div class="card-grid">
      <el-empty v-if="!mounts.length" description="暂无挂载点，添加后即可浏览与管理文件" class="empty" />
      <EntityCard v-for="m in mounts" :key="m.key" :class="{ active: m.key === current?.key }">
        <template #lead>
          <div class="mt-icon"><component :is="m.type === 'local' ? Folder : Cloud" :size="18" /></div>
        </template>
        <template #title>{{ m.name }}</template>
        <template v-if="m.remark" #sub>{{ m.remark }}</template>
        <template #status>
          <StatusBadge v-if="!m.enable" type="off" text="已停用" />
          <StatusBadge v-else-if="m.readOnly" type="info" text="只读" />
          <StatusBadge v-else type="running" text="可读写" />
        </template>
        <Field :label="m.type === 'local' ? '本地目录' : 'WebDAV'" mono>{{ m.type === 'local' ? m.path : m.url }}</Field>
        <template #actions>
          <el-switch :model-value="m.enable" size="small" @change="(v: boolean) => toggle(m, v)" />
          <el-button text size="small" :icon="Pencil" @click="openEditor(m)">编辑</el-button>
          <el-popconfirm title="仅移除挂载点，不会删除其中的文件。继续？" @confirm="remove(m)">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Trash2">移除</el-button>
            </template>
          </el-popconfirm>
          <el-button type="primary" size="small" :icon="FolderOpen" :disabled="!m.enable" @click="browse(m)">浏览</el-button>
        </template>
      </EntityCard>
    </div>

    <div v-if="current" class="g-card">
      <div class="g-card-head">
        <div class="g-card-title"><FolderOpen :size="16" />{{ current.name }}</div>
        <div class="toolbar">
          <el-button size="small" :icon="RefreshCw" @click="refresh">刷新</el-button>
          <template v-if="!current.readOnly">
            <el-button size="small" :icon="FolderPlus" @click="newFolder">新建目录</el-button>
            <el-button size="small" type="primary" :icon="Upload" :loading="uploading" @click="pickFiles">上传</el-button>
          </template>
          <input ref="fileInput" type="file" multiple hidden @change="onPicked" />
        </div>
      </div>
      <div class="g-card-body">
        <el-breadcrumb separator="/" class="crumbs">
          <el-breadcrumb-item v-for="(c, i) in crumbs" :key="i">
            <a class="crumb" @click="go(c.path)">{{ c.name }}</a>
          </el-breadcrumb-item>
        </el-breadcrumb>
        <el-alert v-if="listError" type="error" :title="listError" :closable="false" show-icon />
        <el-table v-else v-loading="loading" :data="entries" empty-text="空目录" size="small">
          <el-table-column label="名称" min-width="220">
            <template #default="{ row }">
              <a class="name" @click="open(row)">
                <component :is="row.isDir ? Folder : File" :size="15" class="ico" />{{ row.name }}
              </a>
            </template>
          </el-table-column>
          <el-table-column label="大小" width="110" align="right">
            <template #default="{ row }">{{ row.isDir ? '—' : fmtSize(row.size) }}</template>
          </el-table-column>
          <el-table-column label="修改时间" width="170" class-name="hide-sm">
            <template #default="{ row }">{{ fmtTime(row.modTime) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="150" align="right">
            <template #default="{ row }">
              <el-button v-if="!row.isDir" text size="small" :icon="Download" aria-label="下载" @click="download(row)" />
              <template v-if="!current.readOnly">
                <el-button text size="small" :icon="Pencil" aria-label="重命名" @click="rename(row)" />
                <el-popconfirm :title="`删除 ${row.name}？${row.isDir ? '目录内容会一并删除。' : ''}`" @confirm="del(row)">
                  <template #reference>
                    <el-button text size="small" type="danger" :icon="Trash2" aria-label="删除" />
                  </template>
                </el-popconfirm>
              </template>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <el-dialog v-model="editorVisible" :title="editing.key ? '编辑挂载点' : '添加挂载点'" width="540px" :fullscreen="isMobile">
      <el-form :model="editing" label-width="110px" :label-position="labelPosition">
        <el-form-item label="名称"><el-input v-model="editing.name" /></el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model="editing.type">
            <el-radio-button value="local">本地目录</el-radio-button>
            <el-radio-button value="webdav">远程 WebDAV</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="editing.type === 'local'" label="目录">
          <el-input v-model="editing.path" class="mono" placeholder="/data/share" />
        </el-form-item>
        <template v-else>
          <el-form-item label="地址"><el-input v-model="editing.url" class="mono" placeholder="https://nas.example.com:5006/dav" /></el-form-item>
          <el-form-item label="用户名"><el-input v-model="editing.username" /></el-form-item>
          <el-form-item label="密码"><el-input v-model="editing.password" type="password" show-password /></el-form-item>
          <el-form-item label="跳过证书校验">
            <el-switch v-model="editing.skipVerify" />
            <span class="help">仅用于自签名证书的内网服务</span>
          </el-form-item>
        </template>
        <el-form-item label="只读"><el-switch v-model="editing.readOnly" /></el-form-item>
        <el-form-item label="启用"><el-switch v-model="editing.enable" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="editing.remark" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Cloud, Download, File, Folder, FolderOpen, FolderPlus, HardDrive, Pencil, Plus, RefreshCw, Trash2, Upload } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import EntityCard from '../components/EntityCard.vue'
import Field from '../components/Field.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { useFormLayout } from '../composables/useIsMobile'

interface Mount {
  key?: string
  name: string
  type: 'local' | 'webdav'
  remark: string
  enable: boolean
  readOnly: boolean
  path: string
  url: string
  username: string
  password: string
  skipVerify: boolean
}
interface Entry {
  name: string
  isDir: boolean
  size: number
  modTime: string
}

const { isMobile, labelPosition } = useFormLayout()
const mounts = ref<Mount[]>([])
const editorVisible = ref(false)
const editing = ref<Mount>(empty())

const current = ref<Mount>()
const cwd = ref('/')
const entries = ref<Entry[]>([])
const loading = ref(false)
const listError = ref('')
const uploading = ref(false)
const fileInput = ref<HTMLInputElement>()

function empty(): Mount {
  return { name: '', type: 'local', remark: '', enable: true, readOnly: false, path: '', url: '', username: '', password: '', skipVerify: false }
}

const crumbs = computed(() => {
  const out = [{ name: '根目录', path: '/' }]
  let acc = ''
  for (const seg of cwd.value.split('/').filter(Boolean)) {
    acc += '/' + seg
    out.push({ name: seg, path: acc })
  }
  return out
})

function join(dir: string, name: string): string {
  return (dir === '/' ? '' : dir) + '/' + name
}

function fmtSize(n: number): string {
  if (n < 1024) {
    return `${n} B`
  }
  const units = ['KB', 'MB', 'GB', 'TB']
  let v = n / 1024
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(v >= 100 ? 0 : 1)} ${units[i]}`
}

function fmtTime(s: string): string {
  const d = new Date(s)
  return isNaN(d.getTime()) || d.getFullYear() < 1980 ? '—' : d.toLocaleString()
}

async function load() {
  mounts.value = await api.get('/api/storage/mounts')
  if (current.value) {
    current.value = mounts.value.find((m) => m.key === current.value?.key)
  }
}

function openEditor(m?: Mount) {
  editing.value = m ? JSON.parse(JSON.stringify(m)) : empty()
  editorVisible.value = true
}

async function save() {
  try {
    const { key, ...body } = editing.value
    if (key) {
      await api.put(`/api/storage/mounts/${key}`, body)
    } else {
      await api.post('/api/storage/mounts', body)
    }
    editorVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function toggle(m: Mount, enable: boolean) {
  try {
    await api.put(`/api/storage/mounts/${m.key}/enable`, { enable })
    if (!enable && current.value?.key === m.key) {
      current.value = undefined
    }
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function remove(m: Mount) {
  try {
    await api.del(`/api/storage/mounts/${m.key}`)
    if (current.value?.key === m.key) {
      current.value = undefined
    }
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

function browse(m: Mount) {
  current.value = m
  go('/')
}

async function go(p: string) {
  if (!current.value) {
    return
  }
  loading.value = true
  listError.value = ''
  try {
    const r = await api.get<{ path: string; entries: Entry[] | null }>(`/api/storage/files/${current.value.key}/list?path=${encodeURIComponent(p)}`)
    cwd.value = r.path
    entries.value = r.entries ?? []
  } catch (e) {
    listError.value = (e as Error).message
    entries.value = []
  } finally {
    loading.value = false
  }
}

const refresh = () => go(cwd.value)

function open(e: Entry) {
  if (e.isDir) {
    go(join(cwd.value, e.name))
  } else {
    download(e)
  }
}

function download(e: Entry) {
  // 浏览器携带会话 Cookie 直接下载，不经过内存
  const a = document.createElement('a')
  a.href = `/api/storage/files/${current.value?.key}/download?path=${encodeURIComponent(join(cwd.value, e.name))}`
  a.download = e.name
  a.click()
}

function pickFiles() {
  fileInput.value?.click()
}

async function onPicked() {
  const input = fileInput.value
  if (!input?.files?.length || !current.value) {
    return
  }
  uploading.value = true
  try {
    // 逐个文件上传，并携带大小供远程 WebDAV 使用
    for (const f of Array.from(input.files)) {
      const fd = new FormData()
      fd.append('file', f, f.name)
      await api.post(`/api/storage/files/${current.value.key}/upload?path=${encodeURIComponent(cwd.value)}&size=${f.size}`, fd)
    }
    ElMessage.success('上传完成')
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    input.value = ''
    uploading.value = false
    await refresh()
  }
}

async function newFolder() {
  try {
    const { value } = await ElMessageBox.prompt('目录名称', '新建目录', { inputPattern: /^[^/\\]+$/, inputErrorMessage: '名称不能为空且不含斜杠' })
    await api.post(`/api/storage/files/${current.value?.key}/mkdir`, { path: join(cwd.value, value) })
    await refresh()
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error((e as Error).message)
    }
  }
}

async function rename(e: Entry) {
  try {
    const { value } = await ElMessageBox.prompt('新名称', '重命名', { inputValue: e.name, inputPattern: /^[^/\\]+$/, inputErrorMessage: '名称不能为空且不含斜杠' })
    await api.post(`/api/storage/files/${current.value?.key}/rename`, { from: join(cwd.value, e.name), to: join(cwd.value, value) })
    await refresh()
  } catch (err) {
    if (err !== 'cancel') {
      ElMessage.error((err as Error).message)
    }
  }
}

async function del(e: Entry) {
  try {
    await api.del(`/api/storage/files/${current.value?.key}?path=${encodeURIComponent(join(cwd.value, e.name))}`)
    await refresh()
  } catch (err) {
    ElMessage.error((err as Error).message)
  }
}

onMounted(load)
</script>

<style scoped>
.toolbar {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.crumbs {
  margin-bottom: 12px;
}
.crumb,
.name {
  cursor: pointer;
}
.name {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.name:hover,
.crumb:hover {
  color: var(--el-color-primary);
}
.ico {
  flex: none;
  color: var(--el-text-color-secondary);
}
.mt-icon {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: var(--el-fill-color);
  color: var(--el-color-primary);
}
.active {
  outline: 2px solid var(--el-color-primary);
  outline-offset: -2px;
}
@media (max-width: 767.98px) {
  :deep(.hide-sm) {
    display: none;
  }
}
</style>
