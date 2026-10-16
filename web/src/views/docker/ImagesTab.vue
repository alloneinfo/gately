<template>
  <div>
    <div class="toolbar">
      <el-input v-model="keyword" placeholder="搜索标签或 ID" clearable :prefix-icon="Search" class="t-search" />
      <div class="t-ops">
        <el-button type="primary" :icon="CloudDownload" @click="pullVisible = true">拉取</el-button>
        <el-button :icon="Hammer" @click="buildVisible = true">构建</el-button>
        <el-button :icon="PackageOpen" :loading="importing" @click="picker?.click()">导入</el-button>
        <el-button :icon="Eraser" @click="pruneDangling">清理悬空</el-button>
        <el-button :icon="RefreshCw" :loading="loading" @click="load">刷新</el-button>
        <input ref="picker" type="file" accept=".tar,application/x-tar" class="hidden" @change="onImport" />
      </div>
    </div>
    <el-progress v-if="importing" :percentage="importPercent" :stroke-width="10" class="gap" />

    <div v-loading="loading && !images.length" class="card-list">
      <el-empty v-if="!loading && !filtered.length" description="没有镜像" />
      <EntityCard v-for="img in filtered" :key="img.id" :off="img.containers === 0 && img.dangling">
        <template #lead>
          <div class="ico"><Disc3 :size="18" /></div>
        </template>
        <template #title>
          <span v-if="img.tags?.length" class="mono tag-main">{{ img.tags[0] }}</span>
          <span v-else class="muted">&lt;none&gt;</span>
          <el-tag v-if="img.tags?.length > 1" size="small" type="info">+{{ img.tags.length - 1 }}</el-tag>
          <el-tag v-if="img.dangling" size="small" type="warning">悬空</el-tag>
        </template>
        <template #sub><span class="mono">{{ shortId(img.id) }}</span></template>
        <template #status>
          <StatusBadge v-if="img.containers > 0" type="ok" :text="`${img.containers} 个容器使用`" />
          <StatusBadge v-else type="off" text="未使用" />
        </template>
        <Field label="大小">{{ formatBytes(img.size) }}</Field>
        <Field label="创建时间">{{ formatTime(img.created) }}</Field>
        <Field v-if="img.tags?.length > 1" label="全部标签" wide mono>
          <div v-for="t in img.tags" :key="t">{{ t }}</div>
        </Field>
        <template #actions>
          <el-button text :icon="Info" class="btn" @click="showDetail(img)">详情</el-button>
          <el-button text :icon="Tag" class="btn" @click="openTag(img)">标签</el-button>
          <el-dropdown trigger="click" @command="(c: string) => onCommand(img, c)">
            <el-button text :icon="Ellipsis" class="btn">更多</el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="push" :icon="CloudUpload" :disabled="!img.tags?.length">推送</el-dropdown-item>
                <el-dropdown-item command="export" :icon="Download">导出 tar</el-dropdown-item>
                <el-dropdown-item command="delete" :icon="Trash2" divided>删除</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
      </EntityCard>
    </div>

    <ImagePullDialog v-model="pullVisible" @done="load" />
    <ImageBuildDialog v-model="buildVisible" @done="load" />
    <ImageDetailDialog v-model="detailVisible" :image-id="detailId" />

    <el-dialog v-model="tagVisible" title="添加标签" width="460px" :fullscreen="isMobile" append-to-body>
      <el-form :label-position="labelPosition" label-width="70px" @submit.prevent>
        <el-form-item label="仓库名" required><el-input v-model="tagForm.repo" placeholder="例如 myrepo/app 或 registry.example.com/app" class="mono" /></el-form-item>
        <el-form-item label="标签"><el-input v-model="tagForm.tag" placeholder="latest" class="mono" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="tagVisible = false">取消</el-button>
        <el-button type="primary" @click="doTag">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="pushVisible" title="推送镜像" width="460px" :fullscreen="isMobile" append-to-body>
      <el-form :label-position="labelPosition" label-width="70px" @submit.prevent>
        <el-form-item label="镜像">
          <el-select v-model="pushRef" class="full">
            <el-option v-for="t in pushTags" :key="t" :label="t" :value="t" />
          </el-select>
        </el-form-item>
      </el-form>
      <div class="help block">使用「系统 → 设置」中配置的私有仓库账号认证。</div>
      <template #footer>
        <el-button @click="pushVisible = false">取消</el-button>
        <el-button type="primary" @click="doPush">推送</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="delVisible" title="删除镜像" width="460px" append-to-body>
      <p class="dlg-text">确定删除镜像 <b class="mono">{{ delTarget?.tags?.[0] || shortId(delTarget?.id ?? '') }}</b>？<span v-if="(delTarget?.tags?.length ?? 0) > 1">其全部 {{ delTarget?.tags.length }} 个标签都会被移除。</span></p>
      <el-checkbox v-model="delForce">强制删除（即使被已停止的容器引用）</el-checkbox>
      <template #footer>
        <el-button @click="delVisible = false">取消</el-button>
        <el-button type="danger" :loading="delBusy" @click="doDelete">删除</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CloudDownload, CloudUpload, Disc3, Download, Ellipsis, Eraser, Hammer, Info, PackageOpen, RefreshCw, Search, Tag, Trash2 } from 'lucide-vue-next'
import { api } from '../../api'
import EntityCard from '../../components/EntityCard.vue'
import Field from '../../components/Field.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import { useFormLayout } from '../../composables/useIsMobile'
import ImageBuildDialog from './ImageBuildDialog.vue'
import ImageDetailDialog from './ImageDetailDialog.vue'
import ImagePullDialog from './ImagePullDialog.vue'
import { download, formatBytes, formatTime, shortId, uploadFile } from './format'
import { reportError } from './state'
import { trackTask } from './tasks'
import type { ImageView } from './types'

const { isMobile, labelPosition } = useFormLayout()
const images = ref<ImageView[]>([])
const loading = ref(false)
const keyword = ref('')
const pullVisible = ref(false)
const buildVisible = ref(false)
const detailVisible = ref(false)
const detailId = ref<string>()
const picker = ref<HTMLInputElement>()
const importing = ref(false)
const importPercent = ref(0)
const tagVisible = ref(false)
const tagTarget = ref<ImageView>()
const tagForm = reactive({ repo: '', tag: 'latest' })
const pushVisible = ref(false)
const pushTags = ref<string[]>([])
const pushRef = ref('')
const delVisible = ref(false)
const delTarget = ref<ImageView>()
const delForce = ref(false)
const delBusy = ref(false)

const filtered = computed(() => {
  const k = keyword.value.trim().toLowerCase()
  return k ? images.value.filter((i) => i.id.toLowerCase().includes(k) || (i.tags ?? []).some((t) => t.toLowerCase().includes(k))) : images.value
})

async function load() {
  loading.value = true
  try {
    images.value = (await api.get<ImageView[]>('/api/docker/images')) ?? []
    reportError('')
  } catch (e) {
    reportError((e as Error).message)
  } finally {
    loading.value = false
  }
}

function showDetail(img: ImageView) {
  detailId.value = img.id
  detailVisible.value = true
}

function openTag(img: ImageView) {
  tagTarget.value = img
  tagForm.repo = ''
  tagForm.tag = 'latest'
  tagVisible.value = true
}

async function doTag() {
  if (!tagTarget.value || !tagForm.repo.trim()) {
    ElMessage.warning('请填写仓库名')
    return
  }
  try {
    await api.post(`/api/docker/images/${encodeURIComponent(tagTarget.value.id)}/tag`, { repo: tagForm.repo.trim(), tag: tagForm.tag.trim() || 'latest' })
    tagVisible.value = false
    ElMessage.success('已添加标签')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function doPush() {
  if (!pushRef.value) {
    return
  }
  try {
    const t = await api.post<{ task: string }>('/api/docker/images/push', { ref: pushRef.value })
    pushVisible.value = false
    trackTask(t.task, `推送 ${pushRef.value}`)
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function doDelete() {
  if (!delTarget.value) {
    return
  }
  delBusy.value = true
  try {
    await api.del(`/api/docker/images/${encodeURIComponent(delTarget.value.id)}${delForce.value ? '?force=1' : ''}`)
    delVisible.value = false
    ElMessage.success('已删除')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    delBusy.value = false
  }
}

function onCommand(img: ImageView, cmd: string) {
  if (cmd === 'push') {
    pushTags.value = img.tags ?? []
    pushRef.value = pushTags.value[0] ?? ''
    pushVisible.value = true
  } else if (cmd === 'export') {
    download(`/api/docker/images/${encodeURIComponent(img.id)}/export`)
  } else if (cmd === 'delete') {
    delTarget.value = img
    delForce.value = false
    delVisible.value = true
  }
}

async function pruneDangling() {
  try {
    await ElMessageBox.confirm('清理所有悬空镜像（无标签且未被使用）？', '清理悬空镜像', { type: 'warning' })
    const r = await api.post<{ deleted: number; spaceReclaimed: number }>('/api/docker/prune', { target: 'images' })
    ElMessage.success(`已清理 ${r.deleted} 个，释放 ${formatBytes(r.spaceReclaimed)}`)
    await load()
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error((e as Error).message)
    }
  }
}

async function onImport(ev: Event) {
  const input = ev.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) {
    return
  }
  importing.value = true
  importPercent.value = 0
  try {
    await uploadFile('/api/docker/images/load', file, 'application/x-tar', (p) => (importPercent.value = p))
    ElMessage.success('镜像已导入')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    importing.value = false
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
.hidden {
  display: none;
}
.gap {
  margin-bottom: 12px;
}
.full {
  width: 100%;
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
.tag-main {
  font-size: 14px;
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
    flex: 1 1 28%;
    min-height: 40px;
  }
  .btn {
    min-height: 40px;
    padding: 0 8px;
  }
  .toolbar :deep(input),
  :deep(.el-dialog input) {
    font-size: 16px;
  }
}
</style>
