<template>
  <div>
    <PageHeader title="网络唤醒" desc="向局域网设备发送 Wake-on-LAN 魔术包，远程开机" :icon="Power">
      <el-button type="primary" :icon="Plus" @click="openEditor()">添加设备</el-button>
    </PageHeader>

    <div class="card-grid">
      <el-empty v-if="!devices.length" description="暂无设备，可添加或在「设置 → 备份与迁移」中导入 Lucky 配置" class="empty" />
      <EntityCard v-for="d in devices" :key="d.key">
        <template #lead>
          <div class="dev-icon" :class="{ online: status[d.key ?? ''] === 'online' }"><Monitor :size="18" /></div>
        </template>
        <template #title>{{ d.name }}</template>
        <template v-if="d.remark" #sub>{{ d.remark }}</template>
        <template #status>
          <StatusBadge v-bind="devStatus(d)" />
        </template>
        <Field label="MAC 地址" mono>
          <div v-for="m in d.macs" :key="m">{{ m }}</div>
        </Field>
        <Field label="广播" mono>{{ (d.broadcastIps ?? []).length ? (d.broadcastIps ?? []).join(', ') : '自动' }}:{{ d.port }} ×{{ d.repeat }}</Field>
        <Field v-if="d.checkAddr" label="在线探测" mono>{{ d.checkAddr }}</Field>
        <template #actions>
          <el-button text size="small" :icon="Pencil" @click="openEditor(d)">编辑</el-button>
          <el-popconfirm title="删除该设备？" @confirm="remove(d)">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Trash2">删除</el-button>
            </template>
          </el-popconfirm>
          <el-button type="primary" size="small" :icon="Zap" :loading="waking[d.key ?? '']" class="wake" @click="wake(d)">唤醒</el-button>
        </template>
      </EntityCard>
    </div>

    <LogCard url="/api/wol/logs" clearable />

    <el-dialog v-model="editorVisible" :title="editing.key ? '编辑设备' : '添加设备'" width="540px" :fullscreen="isMobile">
      <el-form :model="editing" label-width="110px" :label-position="labelPosition">
        <el-form-item label="名称"><el-input v-model="editing.name" /></el-form-item>
        <el-form-item label="MAC 地址"><el-input v-model="macsText" type="textarea" :rows="2" class="mono" placeholder="每行一个，如 aa:bb:cc:dd:ee:ff" /></el-form-item>
        <el-form-item label="广播地址"><el-input v-model="bcText" placeholder="留空自动；多个用逗号分隔，如 192.168.1.255" /></el-form-item>
        <el-form-item label="端口"><el-input-number v-model="editing.port" :min="1" :max="65535" /></el-form-item>
        <el-form-item label="重复次数"><el-input-number v-model="editing.repeat" :min="1" :max="20" /></el-form-item>
        <el-form-item label="在线探测">
          <el-input v-model="editing.checkAddr" placeholder="可选，如 192.168.1.10:3389" />
          <div class="form-help">定期以 TCP 连接该地址判断设备是否在线</div>
        </el-form-item>
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
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Monitor, Pencil, Plus, Power, Trash2, Zap } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import EntityCard from '../components/EntityCard.vue'
import Field from '../components/Field.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import { useFormLayout } from '../composables/useIsMobile'

interface Device {
  key?: string
  name: string
  macs: string[]
  broadcastIps: string[] | null
  port: number
  repeat: number
  checkAddr: string
  remark: string
  enable: boolean
}

const { isMobile, labelPosition } = useFormLayout()
const devices = ref<Device[]>([])
const status = ref<Record<string, string>>({})
const waking = reactive<Record<string, boolean>>({})
const editorVisible = ref(false)
const editing = ref<Device>(empty())
let timer: number | undefined

const macsText = computed({
  get: () => editing.value.macs.join('\n'),
  set: (v: string) => (editing.value.macs = v.split(/[\n,]/).map((s) => s.trim()).filter(Boolean)),
})
const bcText = computed({
  get: () => (editing.value.broadcastIps ?? []).join(', '),
  set: (v: string) => (editing.value.broadcastIps = v.split(',').map((s) => s.trim()).filter(Boolean)),
})

function empty(): Device {
  return { name: '', macs: [], broadcastIps: [], port: 9, repeat: 3, checkAddr: '', remark: '', enable: true }
}

// devStatus 计算设备在线状态；未配置探测地址时明确提示，而不是显示为等待
function devStatus(d: Device): { type: StatusType; text: string } {
  if (!d.checkAddr) {
    return { type: 'info', text: '未配置探测' }
  }
  const s = status.value[d.key ?? '']
  if (s === 'online') {
    return { type: 'running', text: '在线' }
  }
  if (s === 'offline') {
    return { type: 'off', text: '离线' }
  }
  return { type: 'pending', text: '探测中' }
}

async function load() {
  devices.value = await api.get('/api/wol/devices')
  status.value = await api.get('/api/wol/status')
}

function openEditor(d?: Device) {
  editing.value = d ? JSON.parse(JSON.stringify(d)) : empty()
  editorVisible.value = true
}

async function save() {
  try {
    const { key, ...body } = editing.value
    if (key) {
      await api.put(`/api/wol/devices/${key}`, body)
    } else {
      await api.post('/api/wol/devices', body)
    }
    editorVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function wake(d: Device) {
  const k = d.key ?? ''
  waking[k] = true
  try {
    const r = await api.post<{ sent: number }>(`/api/wol/devices/${k}/wake`)
    ElMessage.success(`已发送 ${r.sent} 个唤醒包`)
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    waking[k] = false
  }
}

async function remove(d: Device) {
  try {
    await api.del(`/api/wol/devices/${d.key}`)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(() => {
  load()
  timer = window.setInterval(load, 10000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.dev-icon {
  width: 34px;
  height: 34px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}
.dev-icon.online {
  background: var(--el-color-success-light-9);
  color: var(--el-color-success);
}
.wake {
  margin-left: 6px !important;
}
.empty {
  grid-column: 1 / -1;
}
</style>
