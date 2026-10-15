<template>
  <div>
    <div class="page-header">
      <h2>网络唤醒</h2>
      <el-button type="primary" @click="openEditor()">添加设备</el-button>
    </div>
    <el-card shadow="never">
      <el-table :data="devices" empty-text="暂无设备，可添加或在设置中导入 Lucky 配置">
        <el-table-column label="设备" min-width="160">
          <template #default="{ row }">
            <div>{{ row.name }}</div>
            <div class="meta" v-if="row.remark">{{ row.remark }}</div>
          </template>
        </el-table-column>
        <el-table-column label="MAC" min-width="170">
          <template #default="{ row }">
            <div v-for="m in row.macs" :key="m" class="mono">{{ m }}</div>
          </template>
        </el-table-column>
        <el-table-column label="广播" min-width="160">
          <template #default="{ row }">
            <span class="mono">{{ (row.broadcastIps ?? []).length ? row.broadcastIps.join(', ') : '自动' }}:{{ row.port }}</span>
            <span class="meta"> ×{{ row.repeat }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag v-if="status[row.key] === 'online'" type="success" size="small">在线</el-tag>
            <el-tag v-else-if="status[row.key] === 'offline'" type="info" size="small">离线</el-tag>
            <span v-else class="meta">未探测</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="wake(row)">唤醒</el-button>
            <el-button link type="primary" @click="openEditor(row)">编辑</el-button>
            <el-popconfirm title="删除该设备？" @confirm="remove(row)">
              <template #reference>
                <el-button link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    <el-card shadow="never" header="日志" class="logs">
      <LogViewer url="/api/wol/logs" clearable />
    </el-card>

    <el-dialog v-model="editorVisible" :title="editing.key ? '编辑设备' : '添加设备'" width="520px">
      <el-form :model="editing" label-width="110px">
        <el-form-item label="名称"><el-input v-model="editing.name" /></el-form-item>
        <el-form-item label="MAC 地址"><el-input v-model="macsText" type="textarea" :rows="2" class="mono" placeholder="每行一个，如 aa:bb:cc:dd:ee:ff" /></el-form-item>
        <el-form-item label="广播地址"><el-input v-model="bcText" placeholder="留空自动；多个用逗号分隔，如 192.168.1.255" /></el-form-item>
        <el-form-item label="端口"><el-input-number v-model="editing.port" :min="1" :max="65535" /></el-form-item>
        <el-form-item label="重复次数"><el-input-number v-model="editing.repeat" :min="1" :max="20" /></el-form-item>
        <el-form-item label="在线探测"><el-input v-model="editing.checkAddr" placeholder="可选，如 192.168.1.10:3389" /></el-form-item>
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
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import LogViewer from '../components/LogViewer.vue'

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

const devices = ref<Device[]>([])
const status = ref<Record<string, string>>({})
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
  try {
    const r = await api.post<{ sent: number }>(`/api/wol/devices/${d.key}/wake`)
    ElMessage.success(`已发送 ${r.sent} 个唤醒包`)
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function remove(d: Device) {
  await api.del(`/api/wol/devices/${d.key}`)
  await load()
}

onMounted(() => {
  load()
  timer = window.setInterval(load, 10000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.logs {
  margin-top: 16px;
}
</style>
