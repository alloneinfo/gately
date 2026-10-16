<template>
  <div>
    <div class="toolbar">
      <el-input v-model="keyword" placeholder="搜索名称、镜像或项目" clearable :prefix-icon="Search" class="t-search" />
      <el-radio-group v-model="stateFilter" class="t-filter">
        <el-radio-button value="all">全部</el-radio-button>
        <el-radio-button value="running">运行中</el-radio-button>
        <el-radio-button value="stopped">已停止</el-radio-button>
      </el-radio-group>
      <el-checkbox v-model="grouped">按项目分组</el-checkbox>
      <div class="t-ops">
        <el-button type="primary" :icon="Plus" @click="openForm()">新建容器</el-button>
        <el-button :icon="SearchCheck" :loading="checkingAll" @click="checkAll">检查全部更新</el-button>
        <el-button :icon="ArrowUpCircle" :disabled="!selected.length" @click="upgradeSelected">升级选中{{ selected.length ? ` (${selected.length})` : '' }}</el-button>
        <el-button :icon="RefreshCw" :loading="loading" @click="load">刷新</el-button>
      </div>
    </div>

    <div v-loading="loading && !containers.length" class="groups">
      <el-empty v-if="!loading && !filtered.length" description="没有容器" />
      <div v-for="g in groups" :key="g.key" class="group">
        <button v-if="grouped" type="button" class="g-head" @click="toggle(g.key)">
          <ChevronRight :size="16" class="chev" :class="{ open: !collapsed[g.key] }" />
          <span class="g-name">{{ g.name }}</span>
          <span class="g-count">{{ g.items.length }} 个容器 · {{ g.items.filter((c) => c.state === 'running').length }} 运行中</span>
        </button>
        <div v-show="!grouped || !collapsed[g.key]" class="card-list">
          <ContainerCard v-for="c in g.items" :key="c.id" :c="c" :selected="selectedIds.has(c.id)" :hide-project="grouped" @command="onCommand" @select="onSelect" />
        </div>
      </div>
    </div>

    <ContainerDetail v-model="detailVisible" :container="detailTarget" :initial-tab="detailTab" />
    <ContainerForm v-model="formVisible" :edit-id="formEditId" :edit-name="formEditName" @saved="load" />

    <el-dialog v-model="delVisible" title="删除容器" width="440px" :fullscreen="false" append-to-body>
      <p class="dlg-text">确定删除容器 <b>{{ delTarget?.name }}</b>？运行中的容器会被强制停止，该操作无法恢复。</p>
      <el-checkbox v-model="delVolumes">同时删除匿名卷</el-checkbox>
      <template #footer>
        <el-button @click="delVisible = false">取消</el-button>
        <el-button type="danger" :loading="delBusy" @click="doDelete">删除</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="commitVisible" title="提交为镜像" width="480px" :fullscreen="isMobile" append-to-body>
      <el-form :label-position="labelPosition" label-width="80px">
        <el-form-item label="仓库名" required><el-input v-model="commitForm.repo" placeholder="例如 myapp/web" class="mono" /></el-form-item>
        <el-form-item label="标签"><el-input v-model="commitForm.tag" placeholder="latest" class="mono" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="commitForm.comment" /></el-form-item>
        <el-form-item label="暂停容器"><el-switch v-model="commitForm.pause" /><span class="help">提交期间暂停以保证一致性</span></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="commitVisible = false">取消</el-button>
        <el-button type="primary" :loading="commitBusy" @click="doCommit">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowUpCircle, ChevronRight, Plus, RefreshCw, Search, SearchCheck } from 'lucide-vue-next'
import { api } from '../../api'
import { useFormLayout } from '../../composables/useIsMobile'
import ContainerCard from './ContainerCard.vue'
import ContainerDetail from './ContainerDetail.vue'
import ContainerForm from './ContainerForm.vue'
import { reportError } from './state'
import { trackTask } from './tasks'
import type { ContainerView } from './types'

const { isMobile, labelPosition } = useFormLayout()
const containers = ref<ContainerView[]>([])
const loading = ref(false)
const keyword = ref('')
const stateFilter = ref('all')
const grouped = ref(true)
const collapsed = reactive<Record<string, boolean>>({})
const selectedIds = ref(new Set<string>())
const checkingAll = ref(false)

const detailVisible = ref(false)
const detailTarget = ref<ContainerView>()
const detailTab = ref('overview')
const formVisible = ref(false)
const formEditId = ref<string>()
const formEditName = ref('')
const delVisible = ref(false)
const delTarget = ref<ContainerView>()
const delVolumes = ref(false)
const delBusy = ref(false)
const commitVisible = ref(false)
const commitTarget = ref<ContainerView>()
const commitBusy = ref(false)
const commitForm = reactive({ repo: '', tag: 'latest', comment: '', pause: true })

const selected = computed(() => Array.from(selectedIds.value))

const filtered = computed(() => {
  const k = keyword.value.trim().toLowerCase()
  return containers.value.filter((c) => {
    if (stateFilter.value === 'running' && c.state !== 'running') {
      return false
    }
    if (stateFilter.value === 'stopped' && c.state === 'running') {
      return false
    }
    return !k || c.name.toLowerCase().includes(k) || c.image.toLowerCase().includes(k) || (c.project ?? '').toLowerCase().includes(k)
  })
})

const groups = computed(() => {
  if (!grouped.value) {
    return filtered.value.length ? [{ key: '*', name: '', items: filtered.value }] : []
  }
  const map = new Map<string, ContainerView[]>()
  for (const c of filtered.value) {
    const key = c.project || ''
    if (!map.has(key)) {
      map.set(key, [])
    }
    map.get(key)!.push(c)
  }
  // 项目在前按名称排序，独立容器放最后
  const keys = Array.from(map.keys()).sort((a, b) => {
    if (a === '' || b === '') {
      return a === '' ? 1 : -1
    }
    return a.localeCompare(b)
  })
  return keys.map((key) => ({ key: key || '__none', name: key || '独立容器', items: map.get(key)! }))
})

function toggle(key: string) {
  collapsed[key] = !collapsed[key]
}

function onSelect(id: string, v: boolean) {
  const s = new Set(selectedIds.value)
  if (v) {
    s.add(id)
  } else {
    s.delete(id)
  }
  selectedIds.value = s
}

async function load() {
  loading.value = true
  try {
    containers.value = (await api.get<ContainerView[]>('/api/docker/containers')) ?? []
    reportError('')
    const alive = new Set(containers.value.map((c) => c.id))
    selectedIds.value = new Set(Array.from(selectedIds.value).filter((id) => alive.has(id)))
  } catch (e) {
    reportError((e as Error).message)
  } finally {
    loading.value = false
  }
}

function openForm(c?: ContainerView) {
  formEditId.value = c?.id
  formEditName.value = c?.name ?? ''
  formVisible.value = true
}

function openDetail(c: ContainerView, tab: string) {
  detailTarget.value = c
  detailTab.value = tab
  detailVisible.value = true
}

async function simpleAction(c: ContainerView, action: string) {
  try {
    await api.post(`/api/docker/containers/${c.id}/${action}`)
    ElMessage.success('已执行')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function confirmThen(msg: string, title: string, fn: () => Promise<void>) {
  try {
    await ElMessageBox.confirm(msg, title, { type: 'warning' })
  } catch {
    return
  }
  await fn()
}

async function onCommand(c: ContainerView, cmd: string) {
  switch (cmd) {
    case 'start':
    case 'restart':
    case 'pause':
    case 'unpause':
      await simpleAction(c, cmd)
      break
    case 'stop':
      await confirmThen(`停止 ${c.name}？`, '停止容器', () => simpleAction(c, 'stop'))
      break
    case 'kill':
      await confirmThen(`强制停止（kill）${c.name}？进程将被立即终止，可能丢失未保存的数据。`, '强制停止', () => simpleAction(c, 'kill'))
      break
    case 'detail':
      openDetail(c, 'overview')
      break
    case 'logs':
    case 'term':
    case 'files':
      openDetail(c, cmd)
      break
    case 'edit':
      openForm(c)
      break
    case 'rename':
      await rename(c)
      break
    case 'commit':
      commitTarget.value = c
      commitForm.repo = c.image.split(':')[0].replace(/^sha256$/, '')
      commitForm.tag = 'latest'
      commitForm.comment = ''
      commitForm.pause = true
      commitVisible.value = true
      break
    case 'check':
      await check(c)
      break
    case 'upgrade':
      await upgrade(c)
      break
    case 'delete':
      delTarget.value = c
      delVolumes.value = false
      delVisible.value = true
      break
  }
}

async function rename(c: ContainerView) {
  try {
    const { value } = await ElMessageBox.prompt('新的容器名称', '重命名', { inputValue: c.name, inputPattern: /^[a-zA-Z0-9][a-zA-Z0-9_.-]*$/, inputErrorMessage: '仅允许字母、数字、_ . -' })
    await api.post(`/api/docker/containers/${c.id}/rename`, { name: value })
    await load()
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error((e as Error).message)
    }
  }
}

async function doDelete() {
  if (!delTarget.value) {
    return
  }
  delBusy.value = true
  try {
    await api.del(`/api/docker/containers/${delTarget.value.id}${delVolumes.value ? '?volumes=1' : ''}`)
    delVisible.value = false
    ElMessage.success('已删除')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    delBusy.value = false
  }
}

async function doCommit() {
  if (!commitTarget.value || !commitForm.repo.trim()) {
    ElMessage.warning('请填写仓库名')
    return
  }
  commitBusy.value = true
  try {
    await api.post(`/api/docker/containers/${commitTarget.value.id}/commit`, { repo: commitForm.repo.trim(), tag: commitForm.tag.trim() || 'latest', comment: commitForm.comment, pause: commitForm.pause })
    ElMessage.success('已提交为镜像')
    commitVisible.value = false
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    commitBusy.value = false
  }
}

async function check(c: ContainerView) {
  c.busy = true
  try {
    await api.post(`/api/docker/containers/${c.id}/check`)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    c.busy = false
  }
}

async function upgrade(c: ContainerView) {
  await confirmThen(`将拉取最新镜像并按原配置重建 ${c.name}，期间服务会短暂中断；新容器启动失败时自动恢复旧容器。`, '升级容器', async () => {
    c.busy = true
    try {
      await api.post(`/api/docker/containers/${c.id}/upgrade`)
      ElMessage.success('升级完成')
      await load()
    } catch (e) {
      ElMessage.error((e as Error).message)
    } finally {
      c.busy = false
    }
  })
}

async function checkAll() {
  checkingAll.value = true
  try {
    const r = await api.post<{ task: string }>('/api/docker/containers/check-all')
    trackTask(r.task, '检查全部容器更新', () => {
      checkingAll.value = false
      load()
    })
  } catch (e) {
    checkingAll.value = false
    ElMessage.error((e as Error).message)
  }
}

async function upgradeSelected() {
  await confirmThen(`将依次升级选中的 ${selected.value.length} 个容器（仅升级有新版本的会实际重建），期间相关服务会短暂中断。`, '批量升级', async () => {
    try {
      const r = await api.post<{ task: string }>('/api/docker/containers/upgrade-batch', { ids: selected.value })
      trackTask(r.task, '批量升级容器', () => load())
    } catch (e) {
      ElMessage.error((e as Error).message)
    }
  })
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
.group + .group {
  margin-top: 16px;
}
.g-head {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-height: 40px;
  padding: 4px 4px;
  margin-bottom: 8px;
  border: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.chev {
  transition: transform 0.15s;
  color: var(--el-text-color-secondary);
}
.chev.open {
  transform: rotate(90deg);
}
.g-name {
  font-weight: 600;
  word-break: break-all;
}
.g-count {
  font-size: 12px;
  color: var(--el-text-color-secondary);
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
  .t-filter {
    width: 100%;
  }
  .t-filter :deep(.el-radio-button) {
    flex: 1;
  }
  .t-filter :deep(.el-radio-button__inner) {
    width: 100%;
    min-height: 40px;
    line-height: 26px;
  }
  .t-ops {
    width: 100%;
    margin-left: 0;
  }
  .t-ops .el-button {
    flex: 1 1 40%;
    min-height: 40px;
  }
  .toolbar :deep(input) {
    font-size: 16px;
  }
}
</style>
