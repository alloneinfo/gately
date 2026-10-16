<template>
  <div>
    <div class="toolbar">
      <el-input v-model="keyword" placeholder="搜索卷" clearable :prefix-icon="Search" class="t-search" />
      <div class="t-ops">
        <el-button type="primary" :icon="Plus" @click="openCreate">创建卷</el-button>
        <el-button :icon="PackageOpen" @click="openImport()">导入</el-button>
        <el-button :icon="Eraser" @click="prune">清理未使用</el-button>
        <el-button :icon="RefreshCw" :loading="loading" @click="load">刷新</el-button>
      </div>
    </div>

    <div v-loading="loading && !volumes.length" class="card-list">
      <el-empty v-if="!loading && !filtered.length" description="没有卷" />
      <EntityCard v-for="v in filtered" :key="v.name">
        <template #lead>
          <div class="ico"><Database :size="18" /></div>
        </template>
        <template #title><span class="name">{{ v.name }}</span></template>
        <template #sub>{{ formatTime(v.created) }}</template>
        <template #status>
          <StatusBadge v-if="v.refCount > 0" type="ok" :text="`${v.refCount} 个容器使用`" />
          <StatusBadge v-else type="off" text="未使用" />
        </template>
        <Field label="大小">{{ formatSize(v.size) }}</Field>
        <Field label="驱动">{{ v.driver }}</Field>
        <Field label="挂载点" wide mono>{{ v.mountpoint }}</Field>
        <Field label="使用者" wide>
          <el-tag v-for="u in v.usedBy ?? []" :key="u.id" size="small" type="info" class="used">{{ u.name }}</el-tag>
          <span v-if="!(v.usedBy ?? []).length" class="muted">无</span>
        </Field>
        <template #actions>
          <el-button text :icon="Download" class="btn" @click="download(`/api/docker/volumes/${encodeURIComponent(v.name)}/export`)">导出</el-button>
          <el-button text :icon="Upload" class="btn" @click="openImport(v.name)">导入</el-button>
          <el-button text type="danger" :icon="Trash2" class="btn" @click="openDelete(v)">删除</el-button>
        </template>
      </EntityCard>
    </div>

    <el-dialog v-model="createVisible" title="创建卷" width="440px" :fullscreen="isMobile" append-to-body>
      <el-form :label-position="labelPosition" label-width="70px" @submit.prevent>
        <el-form-item label="名称" required><el-input v-model="cf.name" class="mono" /></el-form-item>
        <el-form-item label="驱动"><el-input v-model="cf.driver" placeholder="local" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="busy" @click="create">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="importVisible" title="导入卷数据" width="480px" :fullscreen="isMobile" append-to-body>
      <el-form :label-position="labelPosition" label-width="70px" @submit.prevent>
        <el-form-item label="卷名" required>
          <el-input v-model="importName" placeholder="不存在时自动创建" class="mono" />
        </el-form-item>
        <el-form-item label="文件" required>
          <div class="full">
            <el-button :icon="Upload" @click="picker?.click()">{{ importFile ? importFile.name : '选择 tar.gz 文件' }}</el-button>
            <input ref="picker" type="file" accept=".gz,.tgz,application/gzip" class="hidden" @change="(e) => (importFile = (e.target as HTMLInputElement).files?.[0])" />
            <el-progress v-if="busy" :percentage="importPercent" class="gap-t" />
          </div>
        </el-form-item>
      </el-form>
      <div class="help block">压缩包内容会解压到卷中，同名文件将被覆盖。</div>
      <template #footer>
        <el-button @click="importVisible = false">取消</el-button>
        <el-button type="primary" :loading="busy" @click="doImport">导入</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="delVisible" title="删除卷" width="440px" append-to-body>
      <p class="dlg-text">确定删除卷 <b class="mono">{{ delTarget?.name }}</b>？卷内数据将永久丢失。<span v-if="delTarget?.refCount">该卷正被 {{ delTarget.refCount }} 个容器使用。</span></p>
      <el-checkbox v-model="delForce">强制删除</el-checkbox>
      <template #footer>
        <el-button @click="delVisible = false">取消</el-button>
        <el-button type="danger" :loading="busy" @click="doDelete">删除</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Database, Download, Eraser, PackageOpen, Plus, RefreshCw, Search, Trash2, Upload } from 'lucide-vue-next'
import { api } from '../../api'
import EntityCard from '../../components/EntityCard.vue'
import Field from '../../components/Field.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import { useFormLayout } from '../../composables/useIsMobile'
import { download, formatBytes, formatSize, formatTime, uploadFile } from './format'
import { reportError } from './state'
import type { VolumeView } from './types'

const { isMobile, labelPosition } = useFormLayout()
const volumes = ref<VolumeView[]>([])
const loading = ref(false)
const keyword = ref('')
const busy = ref(false)
const createVisible = ref(false)
const cf = reactive({ name: '', driver: 'local' })
const importVisible = ref(false)
const importName = ref('')
const importFile = ref<File>()
const importPercent = ref(0)
const picker = ref<HTMLInputElement>()
const delVisible = ref(false)
const delTarget = ref<VolumeView>()
const delForce = ref(false)

const filtered = computed(() => {
  const k = keyword.value.trim().toLowerCase()
  return k ? volumes.value.filter((v) => v.name.toLowerCase().includes(k)) : volumes.value
})

async function load() {
  loading.value = true
  try {
    volumes.value = (await api.get<VolumeView[]>('/api/docker/volumes')) ?? []
    reportError('')
  } catch (e) {
    reportError((e as Error).message)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  cf.name = ''
  cf.driver = 'local'
  createVisible.value = true
}

async function create() {
  if (!cf.name.trim()) {
    ElMessage.warning('请填写卷名')
    return
  }
  busy.value = true
  try {
    await api.post('/api/docker/volumes', { name: cf.name.trim(), driver: cf.driver.trim() || 'local' })
    createVisible.value = false
    ElMessage.success('已创建')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    busy.value = false
  }
}

function openImport(name = '') {
  importName.value = name
  importFile.value = undefined
  importPercent.value = 0
  importVisible.value = true
}

async function doImport() {
  if (!importName.value.trim() || !importFile.value) {
    ElMessage.warning('请填写卷名并选择文件')
    return
  }
  busy.value = true
  try {
    await uploadFile(`/api/docker/volumes/${encodeURIComponent(importName.value.trim())}/import`, importFile.value, 'application/gzip', (p) => (importPercent.value = p))
    importVisible.value = false
    ElMessage.success('已导入')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    busy.value = false
  }
}

function openDelete(v: VolumeView) {
  delTarget.value = v
  delForce.value = false
  delVisible.value = true
}

async function doDelete() {
  if (!delTarget.value) {
    return
  }
  busy.value = true
  try {
    await api.del(`/api/docker/volumes/${encodeURIComponent(delTarget.value.name)}${delForce.value ? '?force=1' : ''}`)
    delVisible.value = false
    ElMessage.success('已删除')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    busy.value = false
  }
}

async function prune() {
  try {
    await ElMessageBox.confirm('清理所有未被容器使用的卷？卷内数据将永久丢失。', '清理未使用卷', { type: 'warning' })
    const r = await api.post<{ deleted: number; spaceReclaimed: number }>('/api/docker/prune', { target: 'volumes' })
    ElMessage.success(`已清理 ${r.deleted} 个卷，释放 ${formatBytes(r.spaceReclaimed)}`)
    await load()
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error((e as Error).message)
    }
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
  flex-wrap: wrap;
  gap: 8px;
  margin-left: auto;
}
.t-ops .el-button + .el-button {
  margin-left: 0;
}
.ico {
  width: 34px;
  height: 34px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}
.name {
  font-family: var(--g-mono);
  font-size: 14px;
}
.used {
  margin: 0 4px 4px 0;
}
.hidden {
  display: none;
}
.full {
  width: 100%;
}
.gap-t {
  margin-top: 8px;
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
    flex: 1 1 40%;
    min-height: 40px;
  }
  .btn {
    min-height: 40px;
  }
  .toolbar :deep(input),
  :deep(.el-dialog input) {
    font-size: 16px;
  }
}
</style>
