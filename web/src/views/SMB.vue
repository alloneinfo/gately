<template>
  <div>
    <PageHeader :title="t('smb.title')" :desc="t('smb.desc')" :icon="FolderTree">
      <el-button type="primary" :icon="Save" :loading="saving" @click="save">{{ t('common.save') }}</el-button>
    </PageHeader>

    <el-alert v-if="!supported" type="warning" :title="t('smb.unsupported')" :closable="false" show-icon class="block-gap" />
    <el-alert v-else-if="error && settings?.enable" type="error" :title="error" :closable="false" show-icon class="block-gap" />

    <div v-if="settings" class="g-card">
      <div class="g-card-head">
        <div class="g-card-title"><Server :size="16" />{{ t('smb.service') }}</div>
        <StatusBadge v-bind="svcStatus" />
      </div>
      <div class="g-card-body">
        <el-form label-width="110px" :label-position="labelPosition" class="svc-form">
          <el-form-item :label="t('common.enable')"><el-switch v-model="settings.enable" :disabled="!supported" /></el-form-item>
          <el-form-item :label="t('smb.listenIp')"><el-input v-model="settings.listenIp" :placeholder="t('smb.listenIpPlaceholder')" class="w-260" /></el-form-item>
          <el-form-item :label="t('smb.port')">
            <el-input-number v-model="settings.port" :min="1" :max="65535" />
            <span class="help">{{ t('smb.portHelp') }}</span>
          </el-form-item>
          <el-form-item :label="t('smb.encryption')">
            <el-switch v-model="settings.requireEncryption" />
            <span class="help">{{ t('smb.encryptionHelp') }}</span>
          </el-form-item>
          <el-form-item :label="t('smb.mdns')">
            <el-switch v-model="settings.mdns" />
            <span class="help">{{ t('smb.mdnsHelp') }}</span>
          </el-form-item>
        </el-form>
      </div>
    </div>

    <FileUsers
      v-if="settings"
      v-model="settings.users"
      :desc="t('smb.usersDesc')"
      mount-summary-key="smb.mountSummary"
      :mount-name-placeholder="t('smb.shareName')"
    />

    <div v-if="settings && savedShares.length" class="g-card">
      <div class="g-card-head">
        <div class="g-card-title"><Link2 :size="16" />{{ t('smb.connect') }} <span class="g-card-desc">{{ t('smb.connectDesc') }}</span></div>
      </div>
      <div class="g-card-body">
        <div class="c-head"><span>{{ t('smb.user') }}</span><span>{{ t('smb.shareName') }}</span><span>Windows</span><span>macOS / Linux</span></div>
        <div v-for="s in savedShares" :key="s.user + '/' + s.name" class="c-row">
          <span>{{ s.user }}</span>
          <span>{{ s.name }}<el-tag v-if="s.readOnly" size="small" type="info" class="ro-tag">{{ t('fileUsers.readOnly') }}</el-tag></span>
          <code v-if="savedPort === 445" class="mono">{{ uncPath(s.name) }}</code>
          <span v-else class="help-inline">{{ t('smb.windowsPortOnly') }}</span>
          <code class="mono">{{ smbURL(s.name) }}</code>
        </div>
      </div>
    </div>

    <LogCard url="/api/smb/logs" clearable />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { FolderTree, Link2, Save, Server } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import FileUsers, { type FileUser } from '../components/FileUsers.vue'
import { useFormLayout } from '../composables/useIsMobile'

interface Settings {
  enable: boolean
  listenIp: string
  port: number
  requireEncryption: boolean
  mdns: boolean
  users: FileUser[]
}

interface ShareRow {
  user: string
  name: string
  readOnly: boolean
}

const { t } = useI18n()
const { labelPosition } = useFormLayout()
const settings = ref<Settings>()
// savedEnable 为已保存的启用状态，状态徽标以它为准
const savedEnable = ref(false)
const savedPort = ref(445)
// savedShares 为已保存配置中的全部共享，用于展示连接地址
const savedShares = ref<ShareRow[]>([])
const running = ref(false)
const error = ref('')
const supported = ref(true)
const saving = ref(false)

const svcStatus = computed((): { type: StatusType; text: string; title?: string } => {
  if (!savedEnable.value) {
    return { type: 'off', text: t('smb.statusOff') }
  }
  if (error.value) {
    return { type: 'error', text: t('smb.statusError'), title: error.value }
  }
  if (running.value) {
    return { type: 'running', text: t('common.running') }
  }
  return { type: 'warn', text: t('smb.statusNotRunning') }
})

// host 为连接地址中的主机：监听具体地址时用该地址，否则用当前访问管理界面的主机名
function host(): string {
  const ip = settings.value?.listenIp
  const h = ip && ip !== '0.0.0.0' && ip !== '::' ? ip : window.location.hostname
  return h.includes(':') ? `[${h}]` : h
}

function uncPath(share: string): string {
  return `\\\\${host().replace(/^\[|\]$/g, '')}\\${share}`
}

function smbURL(share: string): string {
  const port = savedPort.value === 445 ? '' : `:${savedPort.value}`
  // 只转义空格：中文等字符原样显示更易读，访达与 Linux 客户端都能识别
  return `smb://${host()}${port}/${share.replace(/ /g, "%20")}`
}

// sharesOf 与后端 User.Shares 一致：单目录用户的共享名为用户名，多目录用户每个目录一个共享
function sharesOf(users: FileUser[]): ShareRow[] {
  const out: ShareRow[] = []
  for (const u of users) {
    if (u.mounts?.length) {
      for (const mt of u.mounts) {
        out.push({ user: u.name, name: mt.name, readOnly: mt.readOnly })
      }
    } else {
      out.push({ user: u.name, name: u.name, readOnly: u.readOnly })
    }
  }
  return out
}

async function load() {
  const r = await api.get<{ settings: Settings & { users: FileUser[] | null }; running: boolean; error: string; supported: boolean }>('/api/smb/settings')
  settings.value = { ...r.settings, users: r.settings.users ?? [] }
  savedEnable.value = r.settings.enable
  savedPort.value = r.settings.port
  savedShares.value = r.settings.enable ? sharesOf(settings.value.users) : []
  running.value = r.running
  error.value = r.error
  supported.value = r.supported
}

async function save() {
  saving.value = true
  try {
    const r = await api.put<{ running: boolean; error: string }>('/api/smb/settings', settings.value)
    running.value = r.running
    error.value = r.error
    ElMessage.success(t('common.saved'))
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

onMounted(load)
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
.w-260 {
  width: 260px;
}
.c-head,
.c-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) minmax(0, 2fr) minmax(0, 2fr);
  gap: 10px;
  align-items: center;
}
.c-head {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  padding-bottom: 6px;
}
.c-row {
  padding: 8px 0;
  border-top: 1px solid var(--g-border);
  font-size: 13px;
}
.c-row code {
  word-break: break-all;
  user-select: all;
}
.ro-tag {
  margin-left: 6px;
}
.help-inline {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

@media (max-width: 767.98px) {
  .w-260 {
    width: 100%;
  }
  .c-head {
    display: none;
  }
  .c-row {
    grid-template-columns: 1fr 1fr;
  }
  .c-row code,
  .c-row .help-inline {
    grid-column: 1 / -1;
  }
}
</style>
