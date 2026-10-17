<template>
  <div>
    <PageHeader title="FTP 服务" desc="把本机目录以 FTP / FTPS 协议共享给电脑、手机与老旧设备" :icon="HardDrive">
      <el-button type="primary" :icon="Save" :loading="saving" @click="save">保存</el-button>
    </PageHeader>

    <el-alert v-if="error && settings?.enable" type="error" :title="error" :closable="false" show-icon class="block-gap" />

    <div v-if="settings" class="g-card">
      <div class="g-card-head">
        <div class="g-card-title"><Server :size="16" />服务</div>
        <StatusBadge v-bind="svcStatus" />
      </div>
      <div class="g-card-body">
        <el-form label-width="100px" :label-position="labelPosition" class="svc-form">
          <el-form-item label="启用"><el-switch v-model="settings.enable" /></el-form-item>
          <el-form-item label="监听地址"><el-input v-model="settings.listenIp" placeholder="留空表示所有地址" class="w-260" /></el-form-item>
          <el-form-item label="端口"><el-input-number v-model="settings.port" :min="1" :max="65535" /></el-form-item>
          <el-form-item label="加密 (FTPS)">
            <el-select v-model="settings.tlsMode" class="w-260">
              <el-option :value="0" label="明文与 FTPS 均可" />
              <el-option :value="1" label="必须使用 FTPS（显式）" />
              <el-option :value="2" label="隐式 FTPS（连接即加密）" />
            </el-select>
            <span class="help">启用加密时证书按访问域名从「SSL 证书」中选择</span>
          </el-form-item>
          <el-form-item label="被动端口">
            <el-input-number v-model="settings.passiveStart" :min="0" :max="65535" :controls="false" class="w-110" />
            <span class="range-sep">—</span>
            <el-input-number v-model="settings.passiveEnd" :min="0" :max="65535" :controls="false" class="w-110" />
            <span class="help">0 表示随机；位于路由器或容器之后时请固定范围并转发这些端口</span>
          </el-form-item>
          <el-form-item label="被动模式 IP">
            <el-input v-model="settings.publicHost" placeholder="可选，NAT 后填写对外 IP" class="w-260" />
          </el-form-item>
          <el-form-item label="空闲超时"><el-input-number v-model="settings.idleTimeout" :min="1" :max="86400" /><span class="help">秒</span></el-form-item>
          <el-form-item label="禁用主动模式"><el-switch v-model="settings.disableActiveMode" /><span class="help">仅使用被动模式，更适合 NAT 环境</span></el-form-item>
        </el-form>
      </div>
    </div>

    <FileUsers v-if="settings" v-model="settings.users" />

    <LogCard url="/api/ftp/logs" clearable />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { HardDrive, Save, Server } from 'lucide-vue-next'
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
  passiveStart: number
  passiveEnd: number
  publicHost: string
  tlsMode: number
  idleTimeout: number
  connectionTimeout: number
  disableActiveMode: boolean
  users: FileUser[]
}

const { labelPosition } = useFormLayout()
const settings = ref<Settings>()
// savedEnable 为已保存的启用状态，状态徽标以它为准
const savedEnable = ref(false)
const running = ref(false)
const error = ref('')
const saving = ref(false)

const svcStatus = computed((): { type: StatusType; text: string; title?: string } => {
  if (!savedEnable.value) {
    return { type: 'off', text: '已停用' }
  }
  if (error.value) {
    return { type: 'error', text: '启动失败', title: error.value }
  }
  if (running.value) {
    return { type: 'running', text: '运行中' }
  }
  return { type: 'warn', text: '未运行' }
})

async function load() {
  const r = await api.get<{ settings: Settings & { users: FileUser[] | null }; running: boolean; error: string }>('/api/ftp/settings')
  settings.value = { ...r.settings, users: r.settings.users ?? [] }
  savedEnable.value = r.settings.enable
  running.value = r.running
  error.value = r.error
}

async function save() {
  saving.value = true
  try {
    const r = await api.put<{ running: boolean; error: string }>('/api/ftp/settings', settings.value)
    running.value = r.running
    error.value = r.error
    ElMessage.success('已保存')
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
  max-width: 560px;
}
.svc-form .el-form-item:last-child {
  margin-bottom: 0;
}
.w-260 {
  width: 260px;
}
.w-110 {
  width: 110px;
}
.range-sep {
  margin: 0 8px;
  color: var(--el-text-color-secondary);
}

@media (max-width: 767.98px) {
  .w-260 {
    width: 100%;
  }
}
</style>
