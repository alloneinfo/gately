<template>
  <el-dialog
    :model-value="modelValue"
    :title="editId ? `编辑容器 ${editName}` : '新建容器'"
    :width="'720px'"
    :fullscreen="isMobile"
    top="4vh"
    destroy-on-close
    class="ct-form-dialog"
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
    @open="onOpen"
  >
    <el-alert v-if="editId" type="warning" :closable="false" show-icon class="gap" title="保存后将重建容器" description="容器会按新配置删除并重新创建，期间服务短暂中断，匿名卷之外的数据卷与挂载保持不变。" />
    <div v-loading="loadingDetail">
      <el-form :model="f" :label-position="labelPosition" label-width="96px" class="ct-form">
        <el-collapse v-model="opened">
          <el-collapse-item title="基础" name="basic">
            <el-form-item label="名称"><el-input v-model="f.name" placeholder="留空则由 Docker 随机命名" /></el-form-item>
            <el-form-item label="镜像" required><el-input v-model="f.image" placeholder="例如 nginx:latest" class="mono" /></el-form-item>
            <el-form-item v-if="!editId" label="拉取镜像">
              <el-switch v-model="f.pull" />
              <span class="help">镜像不存在时先拉取（可能耗时，请耐心等待）</span>
            </el-form-item>
            <el-form-item v-if="!editId" label="创建后启动"><el-switch v-model="f.autoStart" /></el-form-item>
            <el-form-item label="重启策略">
              <el-select v-model="f.restart" class="w-select">
                <el-option label="不自动重启 (no)" value="no" />
                <el-option label="总是重启 (always)" value="always" />
                <el-option label="除非手动停止 (unless-stopped)" value="unless-stopped" />
                <el-option label="失败时重启 (on-failure)" value="on-failure" />
              </el-select>
              <el-input-number v-if="f.restart === 'on-failure'" v-model="f.restartMaxRetries" :min="0" class="gap-l" />
            </el-form-item>
          </el-collapse-item>

          <el-collapse-item title="命令" name="cmd">
            <el-form-item label="Command"><el-input v-model="f.command" placeholder="留空使用镜像默认" class="mono" /></el-form-item>
            <el-form-item label="Entrypoint"><el-input v-model="f.entrypoint" placeholder="留空使用镜像默认" class="mono" /></el-form-item>
            <el-form-item label="工作目录"><el-input v-model="f.workdir" placeholder="/app" class="mono" /></el-form-item>
            <el-form-item label="用户"><el-input v-model="f.user" placeholder="例如 1000:1000" class="mono" /></el-form-item>
            <el-form-item label="主机名"><el-input v-model="f.hostname" /></el-form-item>
            <el-form-item label="TTY"><el-switch v-model="f.tty" /></el-form-item>
            <el-form-item label="保持 stdin"><el-switch v-model="f.stdinOpen" /></el-form-item>
          </el-collapse-item>

          <el-collapse-item :title="`端口映射 (${f.ports.length})`" name="ports">
            <div v-for="(p, i) in f.ports" :key="i" class="row">
              <el-input v-model="p.hostIp" placeholder="主机 IP（可空）" class="c-ip" />
              <el-input-number v-model="p.host" :min="0" :max="65535" :controls="false" placeholder="主机端口" class="c-port" />
              <el-input-number v-model="p.container" :min="1" :max="65535" :controls="false" placeholder="容器端口" class="c-port" />
              <el-select v-model="p.proto" class="c-proto">
                <el-option label="TCP" value="tcp" />
                <el-option label="UDP" value="udp" />
              </el-select>
              <el-button text type="danger" :icon="Trash2" class="c-del" aria-label="删除" @click="f.ports.splice(i, 1)" />
            </div>
            <el-button :icon="Plus" @click="f.ports.push({ hostIp: '', host: null, container: null, proto: 'tcp' })">添加端口</el-button>
          </el-collapse-item>

          <el-collapse-item :title="`挂载 (${f.volumes.length})`" name="mounts">
            <div v-for="(v, i) in f.volumes" :key="i" class="row">
              <el-select v-model="v.type" class="c-type" @change="v.source = ''">
                <el-option label="目录 (bind)" value="bind" />
                <el-option label="卷 (volume)" value="volume" />
              </el-select>
              <el-select v-if="v.type === 'volume'" v-model="v.source" filterable allow-create default-first-option placeholder="卷名（可选择或输入新卷名）" class="c-grow">
                <el-option v-for="n in volumeNames" :key="n" :label="n" :value="n" />
              </el-select>
              <el-input v-else v-model="v.source" placeholder="主机路径 /data" class="c-grow mono" />
              <el-input v-model="v.target" placeholder="容器路径 /data" class="c-grow mono" />
              <el-checkbox v-model="v.readOnly">只读</el-checkbox>
              <el-button text type="danger" :icon="Trash2" class="c-del" aria-label="删除" @click="f.volumes.splice(i, 1)" />
            </div>
            <el-button :icon="Plus" @click="f.volumes.push({ type: 'bind', source: '', target: '', readOnly: false })">添加挂载</el-button>
          </el-collapse-item>

          <el-collapse-item :title="`环境变量 (${envRows.length})`" name="env">
            <div v-for="(e, i) in envRows" :key="i" class="row">
              <el-input v-model="e.k" placeholder="KEY" class="c-grow mono" />
              <el-input v-model="e.v" placeholder="VALUE" class="c-grow mono" />
              <el-button text type="danger" :icon="Trash2" class="c-del" aria-label="删除" @click="envRows.splice(i, 1)" />
            </div>
            <div class="btns">
              <el-button :icon="Plus" @click="envRows.push({ k: '', v: '' })">添加变量</el-button>
              <el-button :icon="ClipboardPaste" @click="pasteVisible = !pasteVisible">批量粘贴</el-button>
            </div>
            <div v-if="pasteVisible" class="paste">
              <el-input v-model="pasteText" type="textarea" :rows="5" placeholder="每行一个 KEY=VALUE，# 开头为注释" class="mono" />
              <el-button type="primary" class="gap-t" @click="applyPaste">导入</el-button>
            </div>
          </el-collapse-item>

          <el-collapse-item title="网络" name="net">
            <el-form-item label="主网络">
              <el-select v-model="f.network" filterable allow-create default-first-option placeholder="bridge" class="w-select">
                <el-option v-for="n in networkNames" :key="n" :label="n" :value="n" />
              </el-select>
            </el-form-item>
            <el-form-item label="额外网络">
              <el-select v-model="f.networks" multiple filterable allow-create default-first-option class="w-select">
                <el-option v-for="n in networkNames" :key="n" :label="n" :value="n" />
              </el-select>
            </el-form-item>
            <el-form-item label="固定 IP"><el-input v-model="f.ip" placeholder="可空，需自定义网络" class="mono" /></el-form-item>
            <el-form-item label="别名">
              <el-select v-model="f.aliases" multiple filterable allow-create default-first-option :reserve-keyword="false" placeholder="输入后回车" class="w-select" />
            </el-form-item>
            <el-form-item label="DNS">
              <el-select v-model="f.dns" multiple filterable allow-create default-first-option :reserve-keyword="false" placeholder="例如 8.8.8.8" class="w-select" />
            </el-form-item>
          </el-collapse-item>

          <el-collapse-item title="高级" name="adv">
            <el-form-item label="特权模式">
              <el-switch v-model="f.privileged" />
              <span class="help">特权容器拥有宿主机几乎全部权限，请谨慎开启</span>
            </el-form-item>
            <el-form-item label="添加能力">
              <el-select v-model="f.capAdd" multiple filterable allow-create default-first-option class="w-select">
                <el-option v-for="c in caps" :key="c" :label="c" :value="c" />
              </el-select>
            </el-form-item>
            <el-form-item label="移除能力">
              <el-select v-model="f.capDrop" multiple filterable allow-create default-first-option class="w-select">
                <el-option v-for="c in caps" :key="c" :label="c" :value="c" />
              </el-select>
            </el-form-item>
            <el-form-item label="内存 (MB)"><el-input-number v-model="f.memoryMB" :min="0" /><span class="help">0 表示不限制</span></el-form-item>
            <el-form-item label="CPU 核数"><el-input-number v-model="f.cpus" :min="0" :step="0.5" :precision="2" /><span class="help">0 表示不限制</span></el-form-item>
            <el-form-item label="设备">
              <div class="full">
                <div v-for="(d, i) in f.devices" :key="i" class="row">
                  <el-input v-model="d.host" placeholder="主机设备 /dev/ttyUSB0" class="c-grow mono" />
                  <el-input v-model="d.container" placeholder="容器内路径" class="c-grow mono" />
                  <el-input v-model="d.perm" placeholder="rwm" class="c-perm" />
                  <el-button text type="danger" :icon="Trash2" class="c-del" aria-label="删除" @click="f.devices.splice(i, 1)" />
                </div>
                <el-button :icon="Plus" @click="f.devices.push({ host: '', container: '', perm: 'rwm' })">添加设备</el-button>
              </div>
            </el-form-item>
            <el-form-item label="标签">
              <div class="full">
                <div v-for="(l, i) in labelRows" :key="i" class="row">
                  <el-input v-model="l.k" placeholder="key" class="c-grow mono" />
                  <el-input v-model="l.v" placeholder="value" class="c-grow mono" />
                  <el-button text type="danger" :icon="Trash2" class="c-del" aria-label="删除" @click="labelRows.splice(i, 1)" />
                </div>
                <el-button :icon="Plus" @click="labelRows.push({ k: '', v: '' })">添加标签</el-button>
              </div>
            </el-form-item>
          </el-collapse-item>
        </el-collapse>
      </el-form>
    </div>
    <template #footer>
      <div class="foot">
        <el-button @click="emit('update:modelValue', false)">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">{{ editId ? '保存并重建' : '创建' }}</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ClipboardPaste, Plus, Trash2 } from 'lucide-vue-next'
import { api } from '../../api'
import { useFormLayout } from '../../composables/useIsMobile'
import { parseEnvText } from './format'
import type { ContainerDetail, ContainerSpec, NetworkView, PortSpec, VolumeView } from './types'

const props = defineProps<{ modelValue: boolean; editId?: string; editName?: string }>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean]; saved: [id: string] }>()

const { isMobile, labelPosition } = useFormLayout()
const caps = ['NET_ADMIN', 'NET_RAW', 'SYS_ADMIN', 'SYS_PTRACE', 'SYS_TIME', 'SYS_MODULE', 'CHOWN', 'DAC_OVERRIDE', 'MKNOD', 'IPC_LOCK', 'ALL']

// 表单中端口允许暂时为空，提交时再校验
type FormSpec = Omit<ContainerSpec, 'ports' | 'env' | 'labels'> & { ports: PortSpec[] }

function blank(): FormSpec {
  return {
    name: '',
    image: '',
    pull: true,
    autoStart: true,
    command: '',
    entrypoint: '',
    ports: [],
    volumes: [],
    network: 'bridge',
    networks: [],
    ip: '',
    aliases: [],
    restart: 'unless-stopped',
    restartMaxRetries: 0,
    privileged: false,
    capAdd: [],
    capDrop: [],
    devices: [],
    user: '',
    workdir: '',
    hostname: '',
    dns: [],
    memoryMB: 0,
    cpus: 0,
    tty: false,
    stdinOpen: false,
  }
}

const f = reactive<FormSpec>(blank())
const envRows = ref<{ k: string; v: string }[]>([])
const labelRows = ref<{ k: string; v: string }[]>([])
const opened = ref<string[]>(['basic'])
const pasteVisible = ref(false)
const pasteText = ref('')
const networkNames = ref<string[]>([])
const volumeNames = ref<string[]>([])
const loadingDetail = ref(false)
const saving = ref(false)
// 编辑模式下保留详情里的原始 id，提交时使用
let detailState = ''

function splitEnv(line: string): { k: string; v: string } {
  const i = line.indexOf('=')
  return i < 0 ? { k: line, v: '' } : { k: line.slice(0, i), v: line.slice(i + 1) }
}

function applyPaste() {
  for (const line of parseEnvText(pasteText.value)) {
    const kv = splitEnv(line)
    const exist = envRows.value.find((r) => r.k === kv.k)
    if (exist) {
      exist.v = kv.v
    } else {
      envRows.value.push(kv)
    }
  }
  pasteText.value = ''
  pasteVisible.value = false
}

async function loadOptions() {
  try {
    const [nets, vols] = await Promise.all([api.get<NetworkView[]>('/api/docker/networks'), api.get<VolumeView[]>('/api/docker/volumes')])
    networkNames.value = Array.from(new Set(['bridge', 'host', 'none', ...(nets ?? []).map((n) => n.name)]))
    volumeNames.value = (vols ?? []).map((v) => v.name)
  } catch {
    networkNames.value = ['bridge', 'host', 'none']
  }
}

async function onOpen() {
  Object.assign(f, blank())
  envRows.value = []
  labelRows.value = []
  opened.value = ['basic']
  pasteVisible.value = false
  detailState = ''
  loadOptions()
  if (!props.editId) {
    return
  }
  loadingDetail.value = true
  try {
    const d = await api.get<ContainerDetail & { env: string[]; labels: Record<string, string> }>(`/api/docker/containers/${props.editId}`)
    const base = blank()
    for (const k of Object.keys(base) as (keyof FormSpec)[]) {
      const v = (d as unknown as Record<string, unknown>)[k]
      if (v !== undefined && v !== null) {
        ;(f as unknown as Record<string, unknown>)[k] = v
      }
    }
    f.ports = (d.ports ?? []).map((p) => ({ ...p, host: p.host || null }))
    f.volumes = d.volumes ?? []
    f.devices = d.devices ?? []
    f.pull = false
    f.autoStart = d.state === 'running'
    detailState = d.state
    envRows.value = (d.env ?? []).map(splitEnv)
    labelRows.value = Object.entries(d.labels ?? {}).map(([k, v]) => ({ k, v }))
  } catch (e) {
    ElMessage.error((e as Error).message)
    emit('update:modelValue', false)
  } finally {
    loadingDetail.value = false
  }
}

function buildSpec(): ContainerSpec {
  const labels: Record<string, string> = {}
  for (const l of labelRows.value) {
    if (l.k.trim()) {
      labels[l.k.trim()] = l.v
    }
  }
  return {
    ...(f as Omit<FormSpec, 'ports'>),
    name: f.name.trim(),
    image: f.image.trim(),
    ports: f.ports.filter((p) => p.container).map((p) => ({ ...p, host: p.host ?? 0 })) as PortSpec[],
    volumes: f.volumes.filter((v) => v.target.trim() !== ''),
    devices: f.devices.filter((d) => d.host.trim() !== ''),
    env: envRows.value.filter((e) => e.k.trim() !== '').map((e) => `${e.k.trim()}=${e.v}`),
    labels,
  }
}

async function submit() {
  if (!f.image.trim()) {
    ElMessage.warning('请填写镜像')
    opened.value = Array.from(new Set([...opened.value, 'basic']))
    return
  }
  if (props.editId) {
    try {
      await ElMessageBox.confirm('将删除并按新配置重建该容器，期间服务会短暂中断。继续？', '重建容器', { type: 'warning' })
    } catch {
      return
    }
  }
  saving.value = true
  try {
    const body = buildSpec()
    if (props.editId) {
      body.autoStart = detailState === 'running'
    }
    const r = await api.post<{ id: string }>(props.editId ? `/api/docker/containers/${props.editId}/edit` : '/api/docker/containers', body)
    ElMessage.success(props.editId ? '容器已重建' : '容器已创建')
    emit('update:modelValue', false)
    emit('saved', r?.id ?? '')
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.gap {
  margin-bottom: 12px;
}
.gap-l {
  margin-left: 8px;
}
.gap-t {
  margin-top: 8px;
}
.w-select {
  width: 100%;
  max-width: 420px;
}
.full {
  width: 100%;
}
.row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 8px;
}
.c-grow {
  flex: 1 1 160px;
  min-width: 0;
}
.c-ip {
  flex: 1 1 140px;
  min-width: 0;
}
.c-port {
  width: 110px;
}
.c-port :deep(.el-input__inner) {
  text-align: left;
}
.c-proto {
  width: 84px;
}
.c-type {
  width: 130px;
}
.c-perm {
  width: 80px;
}
.btns {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.btns .el-button + .el-button {
  margin-left: 0;
}
.paste {
  margin-top: 8px;
}
.foot {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-bottom: env(safe-area-inset-bottom);
}
:deep(.el-collapse-item__header) {
  font-weight: 600;
}
@media (max-width: 767.98px) {
  /* 动态列表项纵向堆叠，所有输入铺满一行 */
  .row {
    flex-direction: column;
    align-items: stretch;
    padding: 10px;
    border: 1px solid var(--g-border);
    border-radius: var(--g-radius-sm);
    position: relative;
  }
  .row > * {
    width: 100% !important;
    flex: none !important;
  }
  .c-del {
    align-self: flex-end;
    width: auto !important;
    min-height: 40px;
  }
  .btns .el-button,
  .foot .el-button {
    min-height: 40px;
  }
  .foot {
    justify-content: stretch;
  }
  .foot .el-button {
    flex: 1;
  }
  .w-select {
    max-width: none;
  }
  .ct-form :deep(input),
  .ct-form :deep(textarea) {
    font-size: 16px;
  }
}
</style>
