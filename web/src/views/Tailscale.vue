<template>
  <div>
    <PageHeader title="Tailscale" desc="把本机加入 Tailscale 或 Headscale 组网，可发布子网路由与出口节点" :icon="Network">
      <el-button :icon="RefreshCw" :loading="restarting" @click="restart">重启</el-button>
      <el-button type="primary" :icon="Save" :loading="saving" @click="save">保存</el-button>
    </PageHeader>

    <el-alert v-if="status?.error" type="error" :title="status.error" :closable="false" show-icon class="block-gap" />
    <el-alert v-if="status?.authUrl" type="warning" :closable="false" show-icon class="block-gap" title="需要登录">
      <a :href="status.authUrl" target="_blank" rel="noopener" class="auth-link">打开此链接完成设备认证</a>
    </el-alert>

    <div class="g-card">
      <div class="g-card-head">
        <div class="g-card-title"><Activity :size="16" />状态</div>
        <StatusBadge v-bind="badge" />
      </div>
      <div class="g-card-body">
        <div v-if="status?.self" class="info-grid">
          <Field label="本机" mono>{{ status.self.name }}</Field>
          <Field label="Tailscale IP" mono>
            <div v-for="ip in status.self.ips" :key="ip">{{ ip }}</div>
          </Field>
          <Field v-if="status.tailnet" label="Tailnet" mono>{{ status.tailnet }}</Field>
          <Field v-if="status.magicDnsSuffix" label="MagicDNS" mono>{{ status.magicDnsSuffix }}</Field>
          <Field v-if="status.version" label="版本" mono>{{ status.version }}</Field>
        </div>
        <div v-else class="help block">
          {{ settings.enable ? '尚未连接到 tailnet' : '未启用。填写下方配置并打开「启用」后保存即可。' }}
        </div>
        <div v-if="status?.health?.length" class="health">
          <el-tag v-for="h in status.health" :key="h" type="warning" size="small">{{ h }}</el-tag>
        </div>
      </div>
    </div>

    <div v-if="status?.peers?.length" class="g-card">
      <div class="g-card-head">
        <div class="g-card-title"><Waypoints :size="16" />节点 <span class="g-card-desc">{{ status.peers.length }} 台</span></div>
      </div>
      <div class="g-card-body peers">
        <div v-for="p in status.peers" :key="p.dnsName || p.name" class="peer">
          <StatusBadge :type="p.online ? 'running' : 'off'" :text="p.online ? '在线' : '离线'" />
          <div class="peer-main">
            <div class="peer-name">
              {{ p.name }}
              <el-tag v-if="p.exitNodeOption" size="small" type="info">出口节点</el-tag>
              <el-tag v-if="p.routes?.length" size="small" type="info">子网路由</el-tag>
            </div>
            <div class="peer-sub mono">{{ p.ips.join('  ') }}<template v-if="p.routes?.length"> · {{ p.routes.join(', ') }}</template></div>
          </div>
          <span class="peer-os">{{ p.os }}</span>
        </div>
      </div>
    </div>

    <div v-if="loaded" class="g-card">
      <div class="g-card-head">
        <div class="g-card-title"><SlidersHorizontal :size="16" />配置</div>
      </div>
      <div class="g-card-body">
        <el-form label-width="130px" :label-position="labelPosition" class="svc-form">
          <el-form-item label="启用"><el-switch v-model="settings.enable" /></el-form-item>
          <el-form-item label="认证密钥">
            <el-input v-model="settings.authKey" type="password" show-password placeholder="tskey-auth-…（留空则通过登录链接认证）" />
          </el-form-item>
          <el-form-item label="设备名称"><el-input v-model="settings.hostname" placeholder="留空使用系统主机名" /></el-form-item>
          <el-form-item label="控制服务器">
            <el-input v-model="settings.loginServer" placeholder="留空使用 Tailscale 官方；Headscale 填 https://hs.example.com" />
          </el-form-item>
          <el-form-item label="子网路由">
            <el-input v-model="routesText" placeholder="如 192.168.1.0/24，多个用逗号分隔" class="mono" />
            <div class="form-help">把这些网段共享给 tailnet 内其他设备，需在控制台批准</div>
          </el-form-item>
          <el-form-item label="出口节点"><el-switch v-model="settings.advertiseExitNode" /><span class="help">允许其他设备通过本机访问互联网</span></el-form-item>
          <el-form-item label="接受路由"><el-switch v-model="settings.acceptRoutes" /><span class="help">使用其他设备发布的子网路由</span></el-form-item>
          <el-form-item label="接受 DNS"><el-switch v-model="settings.acceptDns" /><span class="help">使用 tailnet 的 MagicDNS 配置</span></el-form-item>
        </el-form>
      </div>
    </div>

    <div v-if="loaded" class="g-card">
      <div class="g-card-head">
        <div class="g-card-title"><Cpu :size="16" />运行方式</div>
      </div>
      <div class="g-card-body">
        <el-form label-width="130px" :label-position="labelPosition" class="svc-form">
          <el-form-item label="托管 tailscaled">
            <el-switch v-model="settings.manageDaemon" />
            <span class="help">关闭后使用系统已运行的 tailscaled，仅执行 tailscale up</span>
          </el-form-item>
          <template v-if="settings.manageDaemon">
            <el-form-item label="用户态网络">
              <el-switch v-model="settings.userspace" />
              <span class="help">无需 TUN 设备与特权，容器中推荐开启</span>
            </el-form-item>
            <el-form-item v-if="settings.userspace" label="SOCKS5 代理">
              <el-input v-model="settings.socks5" placeholder="如 localhost:1055，留空不开启" class="mono" />
              <div class="form-help">用户态网络下，本机程序通过该代理访问 tailnet 内的设备</div>
            </el-form-item>
            <el-form-item label="WireGuard 端口">
              <el-input-number v-model="settings.port" :min="0" :max="65535" controls-position="right" />
              <span class="help">0 表示自动选择；固定后可在防火墙放行该 UDP 端口以便直连</span>
            </el-form-item>
            <el-form-item label="状态目录"><el-input v-model="settings.stateDir" :placeholder="defaultStateDir" class="mono" /></el-form-item>
            <el-form-item label="tailscaled 路径"><el-input v-model="settings.tailscaledPath" placeholder="留空自动查找" class="mono" /></el-form-item>
            <el-form-item label="tailscaled 参数">
              <el-input v-model="settings.daemonExtraArgs" placeholder="追加到 tailscaled 命令行，如 --verbose=1" class="mono" />
            </el-form-item>
            <el-form-item label="环境变量">
              <el-input v-model="settings.daemonEnv" type="textarea" :rows="3" placeholder="每行一个 KEY=VALUE，如 TS_DEBUG_FIREWALL_MODE=nftables" class="mono" />
              <div class="form-help">宿主机 iptables 为 nf_tables 后端时，需设置 TS_DEBUG_FIREWALL_MODE=nftables，否则子网转发规则会落到另一套表里</div>
            </el-form-item>
          </template>
          <el-form-item v-else label="控制套接字">
            <el-input v-model="settings.socket" placeholder="留空使用系统默认，如 /var/run/tailscale/tailscaled.sock" class="mono" />
          </el-form-item>
          <el-form-item label="tailscale 路径"><el-input v-model="settings.tailscalePath" placeholder="留空自动查找" class="mono" /></el-form-item>
          <el-form-item label="额外参数">
            <el-input v-model="settings.extraUpArgs" placeholder="追加到 tailscale up，如 --ssh" class="mono" />
          </el-form-item>
        </el-form>
      </div>
    </div>

    <LogCard url="/api/tailscale/logs" clearable />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Activity, Cpu, Network, RefreshCw, Save, SlidersHorizontal, Waypoints } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import Field from '../components/Field.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import { useFormLayout } from '../composables/useIsMobile'

interface Settings {
  enable: boolean
  manageDaemon: boolean
  tailscaledPath: string
  tailscalePath: string
  stateDir: string
  socket: string
  userspace: boolean
  socks5: string
  authKey: string
  hostname: string
  loginServer: string
  advertiseRoutes: string[]
  advertiseExitNode: boolean
  acceptRoutes: boolean
  acceptDns: boolean
  extraUpArgs: string
  port: number
  daemonExtraArgs: string
  daemonEnv: string
}
interface Peer {
  name: string
  dnsName: string
  ips: string[]
  os: string
  online: boolean
  exitNodeOption: boolean
  routes?: string[]
}
interface Status {
  daemon: { running: boolean }
  version: string
  error: string
  backendState: string
  authUrl?: string
  tailnet?: string
  magicDnsSuffix?: string
  health?: string[]
  self?: Peer
  peers: Peer[]
}

const { labelPosition } = useFormLayout()
const settings = reactive<Settings>({
  enable: false,
  manageDaemon: true,
  tailscaledPath: '',
  tailscalePath: '',
  stateDir: '',
  socket: '',
  userspace: true,
  socks5: '',
  authKey: '',
  hostname: '',
  loginServer: '',
  advertiseRoutes: [],
  advertiseExitNode: false,
  acceptRoutes: false,
  acceptDns: false,
  extraUpArgs: '',
  port: 0,
  daemonExtraArgs: '',
  daemonEnv: '',
})
// loaded 表示配置已从后端读取，避免表单先以默认值闪现
const loaded = ref(false)
const defaultStateDir = ref('')
const status = ref<Status>()
const saving = ref(false)
const restarting = ref(false)
let timer: number | undefined

const routesText = computed({
  get: () => settings.advertiseRoutes.join(', '),
  set: (v: string) => (settings.advertiseRoutes = v.split(/[,\s]+/).filter(Boolean)),
})

// badge 把 tailscale 的后端状态映射为状态徽标
const badge = computed((): { type: StatusType; text: string } => {
  const s = status.value
  if (!settings.enable) {
    return { type: 'off', text: '未启用' }
  }
  if (s?.error) {
    return { type: 'error', text: '异常' }
  }
  switch (s?.backendState) {
    case 'Running':
      return { type: 'running', text: '已连接' }
    case 'NeedsLogin':
    case 'NeedsMachineAuth':
      return { type: 'warn', text: '等待登录' }
    case 'Stopped':
      return { type: 'off', text: '已停止' }
    default:
      return { type: 'pending', text: '连接中' }
  }
})

async function loadStatus() {
  try {
    status.value = await api.get<Status>('/api/tailscale/status')
  } catch {
    // 轮询失败时保留上一次的状态
  }
}

async function loadSettings() {
  const r = await api.get<{ settings: Settings; defaultStateDir: string }>('/api/tailscale/settings')
  Object.assign(settings, r.settings)
  loaded.value = true
  defaultStateDir.value = r.defaultStateDir
}

async function save() {
  saving.value = true
  try {
    await api.put('/api/tailscale/settings', settings)
    ElMessage.success('已保存')
    await loadSettings()
    await loadStatus()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function restart() {
  restarting.value = true
  try {
    await api.post('/api/tailscale/restart')
    await loadStatus()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    restarting.value = false
  }
}

onMounted(() => {
  loadSettings()
  loadStatus()
  timer = window.setInterval(loadStatus, 5000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.block-gap {
  margin-bottom: 16px;
}
.auth-link {
  color: var(--el-color-primary);
  font-weight: 500;
}
.svc-form {
  max-width: 640px;
}
.svc-form .el-form-item:last-child {
  margin-bottom: 0;
}
.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
  gap: 14px;
}
.health {
  margin-top: 12px;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.peers {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.peer {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid var(--g-border);
}
.peer:last-child {
  border-bottom: none;
}
.peer-main {
  flex: 1;
  min-width: 0;
}
.peer-name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
}
.peer-sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  overflow-wrap: anywhere;
}
.peer-os {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
