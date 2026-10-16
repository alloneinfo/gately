<template>
  <div>
    <PageHeader title="Cloudflare 隧道" desc="通过 cloudflared 把内网服务发布到 Cloudflare，无需公网 IP 与开放端口" :icon="Cloud">
      <el-button type="primary" :icon="Plus" @click="openEditor()">添加隧道</el-button>
    </PageHeader>

    <el-alert v-if="binError" type="warning" :title="binError" :closable="false" show-icon class="block-gap" />

    <div class="card-grid">
      <el-empty v-if="!tunnels.length" description="暂无隧道。令牌隧道在 Cloudflare Zero Trust 控制台创建，临时隧道无需账号" class="empty" />
      <EntityCard v-for="t in tunnels" :key="t.key" :off="!t.enable">
        <template #title>{{ t.name }}</template>
        <template #sub>{{ t.mode === 'token' ? '令牌隧道' : '临时隧道' }}<template v-if="t.remark"> · {{ t.remark }}</template></template>
        <template #status><StatusBadge v-bind="tunnelBadge(t)" /></template>
        <Field v-if="t.mode === 'quick'" label="本地服务" mono>{{ t.service }}</Field>
        <Field v-if="t.mode === 'quick'" label="公网地址" mono wide>
          <a v-if="t.url" :href="t.url" target="_blank" rel="noopener">{{ t.url }}</a>
          <span v-else class="muted">{{ t.state.running ? '申请中…' : '—' }}</span>
        </Field>
        <Field v-if="t.mode === 'token'" label="边缘连接" mono>{{ t.connections }}</Field>
        <Field label="协议" mono>{{ t.protocol || 'auto' }}</Field>
        <template #actions>
          <el-switch :model-value="t.enable" size="small" @change="(v: boolean | string | number) => toggle(t, Boolean(v))" />
          <el-button text size="small" :icon="RefreshCw" :disabled="!t.enable" @click="restart(t)">重启</el-button>
          <el-button text size="small" :icon="Pencil" @click="openEditor(t)">编辑</el-button>
          <el-popconfirm title="删除该隧道？" @confirm="remove(t)">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Trash2">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </EntityCard>
    </div>

    <div class="g-card">
      <div class="g-card-head">
        <div class="g-card-title"><Terminal :size="16" />cloudflared 程序 <span v-if="version" class="g-card-desc">{{ version }}</span></div>
        <el-button size="small" :icon="Save" @click="saveSettings">保存</el-button>
      </div>
      <div class="g-card-body">
        <el-form label-width="110px" :label-position="labelPosition" class="svc-form">
          <el-form-item label="可执行文件">
            <el-input v-model="binPath" :placeholder="binary || '留空自动查找（PATH 或配置目录 bin）'" class="mono" />
          </el-form-item>
        </el-form>
      </div>
    </div>

    <LogCard url="/api/cloudflared/logs" clearable />

    <el-dialog v-model="editorVisible" :title="editing.key ? '编辑隧道' : '添加隧道'" width="540px" :fullscreen="isMobile">
      <el-form :model="editing" label-width="100px" :label-position="labelPosition">
        <el-form-item label="名称"><el-input v-model="editing.name" /></el-form-item>
        <el-form-item label="模式">
          <el-radio-group v-model="editing.mode">
            <el-radio-button value="token">令牌隧道</el-radio-button>
            <el-radio-button value="quick">临时隧道</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="editing.mode === 'token'" label="隧道令牌">
          <el-input v-model="editing.token" type="password" show-password placeholder="eyJh…" />
          <div class="form-help">在 Zero Trust 控制台创建隧道后复制令牌；公网主机名与回源服务也在控制台配置</div>
        </el-form-item>
        <el-form-item v-else label="本地服务">
          <el-input v-model="editing.service" placeholder="http://localhost:8080" class="mono" />
          <div class="form-help">自动分配 trycloudflare.com 随机地址，重启后地址会变化，仅适合临时分享</div>
        </el-form-item>
        <el-form-item label="协议">
          <el-select v-model="editing.protocol">
            <el-option label="auto" value="auto" />
            <el-option label="quic" value="quic" />
            <el-option label="http2" value="http2" />
          </el-select>
          <div class="form-help">UDP 被限制的网络可选择 http2</div>
        </el-form-item>
        <el-form-item label="边缘 IP">
          <el-select v-model="editing.edgeIpVersion">
            <el-option label="auto" value="auto" />
            <el-option label="IPv4" value="4" />
            <el-option label="IPv6" value="6" />
          </el-select>
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
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Cloud, Pencil, Plus, RefreshCw, Save, Terminal, Trash2 } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import EntityCard from '../components/EntityCard.vue'
import Field from '../components/Field.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import { useFormLayout } from '../composables/useIsMobile'

interface Tunnel {
  key?: string
  name: string
  enable: boolean
  mode: string
  token: string
  service: string
  protocol: string
  edgeIpVersion: string
  remark: string
}
// TunnelView 是后端返回的隧道，附带运行状态
interface TunnelView extends Tunnel {
  state: { running: boolean }
  error: string
  url?: string
  connections: number
}

const { isMobile, labelPosition } = useFormLayout()
const tunnels = ref<TunnelView[]>([])
const binPath = ref('')
const binary = ref('')
const version = ref('')
const binError = ref('')
const editorVisible = ref(false)
const editing = ref<Tunnel>(empty())
let timer: number | undefined

function empty(): Tunnel {
  return { name: '', enable: true, mode: 'token', token: '', service: 'http://localhost:8080', protocol: 'auto', edgeIpVersion: 'auto', remark: '' }
}

function tunnelBadge(t: TunnelView): { type: StatusType; text: string; title?: string } {
  if (!t.enable) {
    return { type: 'off', text: '已停用' }
  }
  if (t.error) {
    return { type: 'error', text: '启动失败', title: t.error }
  }
  if (!t.state.running) {
    return { type: 'warn', text: '未运行' }
  }
  if (t.mode === 'token' && t.connections === 0) {
    return { type: 'pending', text: '连接中' }
  }
  return { type: 'running', text: '运行中' }
}

async function loadTunnels() {
  try {
    tunnels.value = await api.get<TunnelView[]>('/api/cloudflared/tunnels')
  } catch {
    // 轮询失败时保留上一次的状态
  }
}

async function loadSettings() {
  const r = await api.get<{ settings: { binPath: string }; binary: string; version: string; error: string }>('/api/cloudflared/settings')
  binPath.value = r.settings.binPath
  binary.value = r.binary
  version.value = r.version
  binError.value = r.error
}

async function saveSettings() {
  try {
    await api.put('/api/cloudflared/settings', { binPath: binPath.value })
    ElMessage.success('已保存')
    await Promise.all([loadSettings(), loadTunnels()])
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

function openEditor(t?: Tunnel) {
  editing.value = t ? JSON.parse(JSON.stringify(t)) : empty()
  editorVisible.value = true
}

async function save() {
  try {
    const { key, ...body } = editing.value
    if (key) {
      await api.put(`/api/cloudflared/tunnels/${key}`, body)
    } else {
      await api.post('/api/cloudflared/tunnels', body)
    }
    editorVisible.value = false
    await loadTunnels()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function toggle(t: Tunnel, enable: boolean) {
  try {
    await api.put(`/api/cloudflared/tunnels/${t.key}/enable`, { enable })
    await loadTunnels()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function restart(t: Tunnel) {
  try {
    await api.post(`/api/cloudflared/tunnels/${t.key}/restart`)
    await loadTunnels()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function remove(t: Tunnel) {
  try {
    await api.del(`/api/cloudflared/tunnels/${t.key}`)
    await loadTunnels()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(() => {
  loadSettings()
  loadTunnels()
  timer = window.setInterval(loadTunnels, 4000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.block-gap {
  margin-bottom: 16px;
}
.svc-form {
  max-width: 640px;
}
.svc-form .el-form-item:last-child {
  margin-bottom: 0;
}
.muted {
  color: var(--el-text-color-secondary);
}
.empty {
  grid-column: 1 / -1;
}
.card-grid + .g-card {
  margin-top: 16px;
}
</style>
