<template>
  <div>
    <div class="page-header">
      <h2>端口转发</h2>
      <div>
        <el-button @click="openSettings">全局设置</el-button>
        <el-button type="primary" @click="openEditor()">添加规则</el-button>
      </div>
    </div>

    <el-card shadow="never">
      <el-table :data="rules" v-loading="loading" empty-text="暂无规则，可添加或在设置中导入 Lucky 配置">
        <el-table-column label="启用" width="70">
          <template #default="{ row }">
            <el-switch :model-value="row.enable" size="small" @change="(v: string | number | boolean) => toggle(row, Boolean(v))" />
          </template>
        </el-table-column>
        <el-table-column label="名称" min-width="120">
          <template #default="{ row }">
            <div>{{ row.name || '未命名' }}</div>
            <div class="remark" v-if="row.remark">{{ row.remark }}</div>
          </template>
        </el-table-column>
        <el-table-column label="协议" min-width="120">
          <template #default="{ row }">
            <el-tag v-for="p in row.protocols" :key="p" size="small" effect="plain" class="proto">{{ p }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="监听" min-width="140">
          <template #default="{ row }">
            <span class="mono">{{ listenText(row) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="目标" min-width="180">
          <template #default="{ row }">
            <div v-for="t in row.targetAddresses" :key="t" class="mono">{{ t }}:{{ row.targetPorts || row.listenPorts }}</div>
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="160">
          <template #default="{ row }">
            <el-tag v-if="row.error" type="danger" size="small">{{ row.error }}</el-tag>
            <el-tag v-else-if="row.running" type="success" size="small">运行中</el-tag>
            <el-tag v-else type="info" size="small">已停止</el-tag>
            <el-tooltip v-if="row.details?.length" placement="top">
              <template #content>
                <div v-for="d in row.details" :key="d">{{ d }}</div>
              </template>
              <el-icon class="detail"><Warning /></el-icon>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="连接 / 流量" min-width="170">
          <template #default="{ row }">
            <div>{{ row.stats.activeConns }} 活动 · {{ row.stats.totalConns }} 累计</div>
            <div class="remark">↑ {{ formatBytes(row.stats.bytesIn) }} ↓ {{ formatBytes(row.stats.bytesOut) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="130" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditor(row)">编辑</el-button>
            <el-popconfirm title="确定删除这条规则？" @confirm="remove(row)">
              <template #reference>
                <el-button link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never" header="日志" class="logs">
      <LogViewer url="/api/portforward/logs" clearable />
    </el-card>

    <el-dialog v-model="editorVisible" :title="editing.key ? '编辑规则' : '添加规则'" width="560px">
      <el-form :model="editing" label-width="110px">
        <el-form-item label="名称">
          <el-input v-model="editing.name" placeholder="例如：NAS SSH" />
        </el-form-item>
        <el-form-item label="协议" required>
          <el-checkbox-group v-model="editing.protocols">
            <el-checkbox v-for="p in protocols" :key="p.value" :value="p.value">{{ p.label }}</el-checkbox>
          </el-checkbox-group>
          <div class="help">tcp/udp 为 IPv4+IPv6 双栈；带 4/6 后缀表示仅该协议族</div>
        </el-form-item>
        <el-form-item label="监听地址">
          <el-input v-model="editing.listenAddress" placeholder="留空表示所有地址" />
        </el-form-item>
        <el-form-item label="监听端口" required>
          <el-input v-model="editing.listenPorts" placeholder="80 或 80,443 或 8000-8010" />
        </el-form-item>
        <el-form-item label="目标地址" required>
          <el-input v-model="targetsText" type="textarea" :rows="2" placeholder="每行一个，多个地址时轮询并自动故障转移" />
        </el-form-item>
        <el-form-item label="目标端口">
          <el-input v-model="editing.targetPorts" placeholder="留空与监听端口相同；数量需一致或只填一个" />
        </el-form-item>
        <el-form-item label="最大连接数">
          <el-input-number v-model="editing.maxConnections" :min="0" :max="100000" />
          <span class="help inline">0 使用全局默认值</span>
        </el-form-item>
        <el-form-item label="UDP 超时(秒)">
          <el-input-number v-model="editing.udpSessionTimeout" :min="0" :max="3600" />
          <span class="help inline">0 使用全局默认值</span>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="editing.remark" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="editing.enable" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="settingsVisible" title="端口转发全局设置" width="480px">
      <el-form v-if="settings" :model="settings" label-width="150px">
        <el-form-item label="启用模块"><el-switch v-model="settings.enable" /></el-form-item>
        <el-form-item label="默认最大连接数"><el-input-number v-model="settings.defaultMaxConnections" :min="1" /></el-form-item>
        <el-form-item label="默认 UDP 超时(秒)"><el-input-number v-model="settings.defaultUDPSessionTimeout" :min="1" /></el-form-item>
        <el-form-item label="UDP 会话上限"><el-input-number v-model="settings.maxUDPSessions" :min="1" /></el-form-item>
        <el-form-item label="连接目标超时(秒)"><el-input-number v-model="settings.dialTimeout" :min="1" :max="60" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="settingsVisible = false">取消</el-button>
        <el-button type="primary" @click="saveSettings">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Warning } from '@element-plus/icons-vue'
import { api, formatBytes } from '../api'
import LogViewer from '../components/LogViewer.vue'

interface Rule {
  key?: string
  name: string
  enable: boolean
  protocols: string[]
  listenAddress: string
  listenPorts: string
  targetAddresses: string[]
  targetPorts: string
  maxConnections: number
  udpSessionTimeout: number
  remark: string
}

interface RuleView extends Rule {
  running: boolean
  error?: string
  details?: string[]
  stats: { activeConns: number; totalConns: number; bytesIn: number; bytesOut: number }
}

interface Settings {
  enable: boolean
  defaultMaxConnections: number
  defaultUDPSessionTimeout: number
  maxUDPSessions: number
  dialTimeout: number
}

const protocols = [
  { value: 'tcp', label: 'TCP' },
  { value: 'udp', label: 'UDP' },
  { value: 'tcp4', label: 'TCP4' },
  { value: 'udp4', label: 'UDP4' },
  { value: 'tcp6', label: 'TCP6' },
  { value: 'udp6', label: 'UDP6' },
]

const rules = ref<RuleView[]>([])
const loading = ref(false)
const editorVisible = ref(false)
const saving = ref(false)
const editing = ref<Rule>(emptyRule())
const targetsText = ref('')
const settingsVisible = ref(false)
const settings = ref<Settings>()
let timer: number | undefined

function emptyRule(): Rule {
  return {
    name: '',
    enable: true,
    protocols: ['tcp'],
    listenAddress: '',
    listenPorts: '',
    targetAddresses: [],
    targetPorts: '',
    maxConnections: 0,
    udpSessionTimeout: 0,
    remark: '',
  }
}

function listenText(r: Rule) {
  const host = r.listenAddress ? (r.listenAddress.includes(':') ? `[${r.listenAddress}]` : r.listenAddress) : '*'
  return `${host}:${r.listenPorts}`
}

async function load(showLoading = false) {
  loading.value = showLoading
  try {
    rules.value = await api.get<RuleView[]>('/api/portforward/rules')
  } finally {
    loading.value = false
  }
}

function openEditor(row?: RuleView) {
  if (row) {
    const { running: _r, error: _e, details: _d, stats: _s, ...rule } = row
    editing.value = JSON.parse(JSON.stringify(rule))
  } else {
    editing.value = emptyRule()
  }
  targetsText.value = editing.value.targetAddresses.join('\n')
  editorVisible.value = true
}

async function save() {
  editing.value.targetAddresses = targetsText.value
    .split(/[\n,]/)
    .map((s) => s.trim())
    .filter(Boolean)
  saving.value = true
  try {
    const { key, ...body } = editing.value
    if (key) {
      await api.put(`/api/portforward/rules/${key}`, body)
    } else {
      await api.post('/api/portforward/rules', body)
    }
    editorVisible.value = false
    ElMessage.success('已保存')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function toggle(row: RuleView, enable: boolean) {
  try {
    await api.put(`/api/portforward/rules/${row.key}/enable`, { enable })
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function remove(row: RuleView) {
  try {
    await api.del(`/api/portforward/rules/${row.key}`)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function openSettings() {
  settings.value = await api.get<Settings>('/api/portforward/settings')
  settingsVisible.value = true
}

async function saveSettings() {
  try {
    await api.put('/api/portforward/settings', settings.value)
    settingsVisible.value = false
    ElMessage.success('已保存，规则已按新设置重启')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(() => {
  load(true)
  // 定时刷新连接数与流量
  timer = window.setInterval(load, 3000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.remark {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.proto {
  margin-right: 4px;
}
.detail {
  margin-left: 6px;
  color: var(--el-color-warning);
  vertical-align: middle;
}
.help {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}
.help.inline {
  margin-left: 10px;
}
.logs {
  margin-top: 16px;
}
</style>
