<template>
  <div>
    <PageHeader title="端口转发" desc="把本机端口的 TCP / UDP 流量转发到内网或外部地址" :icon="ArrowLeftRight">
      <el-button :icon="SlidersHorizontal" @click="openSettings">全局设置</el-button>
      <el-button type="primary" :icon="Plus" @click="openEditor()">添加规则</el-button>
    </PageHeader>

    <ModuleBanner v-if="settings && !settings.enable" title="端口转发模块已停用" :loading="enabling" @enable="enableModule" />

    <div v-loading="loading" class="card-list">
      <el-empty v-if="!loading && !rules.length" description="暂无规则，可添加或在「设置 → 备份与迁移」中导入 Lucky 配置" />
      <EntityCard v-for="r in rules" :key="r.key" :off="!r.enable || moduleOff">
        <template #lead>
          <el-switch :model-value="r.enable" @change="(v: string | number | boolean) => toggle(r, Boolean(v))" />
        </template>
        <template #title>
          <span>{{ r.name || '未命名' }}</span>
          <span class="tags">
            <el-tag v-for="p in r.protocols" :key="p" size="small" effect="plain" class="mono">{{ p.toUpperCase() }}</el-tag>
          </span>
        </template>
        <template v-if="r.remark" #sub>{{ r.remark }}</template>
        <template #status>
          <el-tooltip v-if="r.details?.length && r.enable" placement="top">
            <template #content>
              <div v-for="d in r.details" :key="d">{{ d }}</div>
            </template>
            <TriangleAlert :size="16" class="detail" />
          </el-tooltip>
          <StatusBadge v-bind="ruleStatus(r)" />
        </template>
        <Field label="监听" mono>{{ listenText(r) }}</Field>
        <Field label="目标" mono>
          <div v-for="t in r.targetAddresses" :key="t">{{ t }}:{{ r.targetPorts || r.listenPorts }}</div>
        </Field>
        <Field label="连接">{{ r.stats.activeConns }} 活动 · {{ r.stats.totalConns }} 累计</Field>
        <Field label="流量">↑ {{ formatBytes(r.stats.bytesIn) }} · ↓ {{ formatBytes(r.stats.bytesOut) }}</Field>
        <template #actions>
          <el-button text size="small" :icon="Pencil" @click="openEditor(r)">编辑</el-button>
          <el-popconfirm title="确定删除这条规则？" @confirm="remove(r)">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Trash2">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </EntityCard>
    </div>

    <LogCard url="/api/portforward/logs" clearable />

    <el-dialog v-model="editorVisible" :title="editing.key ? '编辑规则' : '添加规则'" width="600px" :fullscreen="isMobile">
      <el-form :model="editing" label-width="110px" :label-position="labelPosition">
        <el-form-item label="名称">
          <el-input v-model="editing.name" placeholder="例如：NAS SSH" />
        </el-form-item>
        <el-form-item label="协议" required>
          <el-checkbox-group v-model="editing.protocols">
            <el-checkbox v-for="p in protocols" :key="p.value" :value="p.value">{{ p.label }}</el-checkbox>
          </el-checkbox-group>
          <div class="form-help">tcp/udp 为 IPv4+IPv6 双栈；带 4/6 后缀表示仅该协议族</div>
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
          <span class="help">0 使用全局默认值</span>
        </el-form-item>
        <el-form-item label="UDP 超时(秒)">
          <el-input-number v-model="editing.udpSessionTimeout" :min="0" :max="3600" />
          <span class="help">0 使用全局默认值</span>
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

    <el-dialog v-model="settingsVisible" title="端口转发全局设置" width="520px" :fullscreen="isMobile">
      <el-form v-if="settingsForm" :model="settingsForm" label-width="150px" :label-position="labelPosition">
        <el-form-item label="启用模块"><el-switch v-model="settingsForm.enable" /></el-form-item>
        <el-form-item label="默认最大连接数"><el-input-number v-model="settingsForm.defaultMaxConnections" :min="1" /></el-form-item>
        <el-form-item label="默认 UDP 超时(秒)"><el-input-number v-model="settingsForm.defaultUDPSessionTimeout" :min="1" /></el-form-item>
        <el-form-item label="UDP 会话上限"><el-input-number v-model="settingsForm.maxUDPSessions" :min="1" /></el-form-item>
        <el-form-item label="连接目标超时(秒)"><el-input-number v-model="settingsForm.dialTimeout" :min="1" :max="60" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="settingsVisible = false">取消</el-button>
        <el-button type="primary" @click="saveSettings">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowLeftRight, Pencil, Plus, SlidersHorizontal, Trash2, TriangleAlert } from 'lucide-vue-next'
import { api, formatBytes } from '../api'
import PageHeader from '../components/PageHeader.vue'
import ModuleBanner from '../components/ModuleBanner.vue'
import EntityCard from '../components/EntityCard.vue'
import Field from '../components/Field.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import { useFormLayout } from '../composables/useIsMobile'

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

const { isMobile, labelPosition } = useFormLayout()
const rules = ref<RuleView[]>([])
const loading = ref(true)
const editorVisible = ref(false)
const saving = ref(false)
const editing = ref<Rule>(emptyRule())
const targetsText = ref('')
const settings = ref<Settings>()
const settingsForm = ref<Settings>()
const settingsVisible = ref(false)
const enabling = ref(false)
let timer: number | undefined

const moduleOff = computed(() => settings.value?.enable === false)

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

// ruleStatus 计算规则的展示状态：停用的规则只显示“已停用”，不显示为异常
function ruleStatus(r: RuleView): { type: StatusType; text: string; title?: string } {
  if (moduleOff.value) {
    return { type: 'off', text: '模块已停用' }
  }
  if (!r.enable) {
    return { type: 'off', text: '已停用' }
  }
  if (r.error) {
    return { type: 'error', text: r.error, title: r.error }
  }
  if (r.running) {
    return { type: 'running', text: '运行中' }
  }
  return { type: 'warn', text: '未运行' }
}

async function load() {
  try {
    rules.value = await api.get<RuleView[]>('/api/portforward/rules')
  } finally {
    loading.value = false
  }
}

async function loadSettings() {
  settings.value = await api.get<Settings>('/api/portforward/settings')
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
  await loadSettings()
  settingsForm.value = JSON.parse(JSON.stringify(settings.value))
  settingsVisible.value = true
}

async function saveSettings() {
  try {
    await api.put('/api/portforward/settings', settingsForm.value)
    settingsVisible.value = false
    ElMessage.success('已保存，规则已按新设置重启')
    await Promise.all([loadSettings(), load()])
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

// enableModule 在横幅中一键启用模块
async function enableModule() {
  if (!settings.value) {
    return
  }
  enabling.value = true
  try {
    await api.put('/api/portforward/settings', { ...settings.value, enable: true })
    ElMessage.success('端口转发模块已启用')
    await Promise.all([loadSettings(), load()])
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    enabling.value = false
  }
}

onMounted(() => {
  load()
  loadSettings()
  // 定时刷新连接数与流量
  timer = window.setInterval(load, 3000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.detail {
  color: var(--el-color-warning);
}
</style>
