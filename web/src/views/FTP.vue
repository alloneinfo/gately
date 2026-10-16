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

    <div v-if="settings" class="g-card">
      <div class="g-card-head">
        <div class="g-card-title"><Users :size="16" />用户 <span class="g-card-desc">每个用户可访问各自的目录</span></div>
        <el-button size="small" :icon="UserPlus" @click="settings.users.push({ name: '', password: '', root: '', readOnly: false })">添加用户</el-button>
      </div>
      <div class="g-card-body">
        <div v-if="settings.users.length" class="u-head">
          <span>用户名</span><span>密码</span><span>目录</span><span>只读</span><span />
        </div>
        <div v-for="(u, i) in settings.users" :key="i" class="u-row">
          <div class="u-cell"><label class="u-label">用户名</label><el-input v-model="u.name" /></div>
          <div class="u-cell"><label class="u-label">密码</label><el-input v-model="u.password" type="password" show-password /></div>
          <div class="u-cell u-root"><label class="u-label">目录</label><el-input v-model="u.root" placeholder="/data/share" class="mono" /></div>
          <div class="u-cell u-ro"><label class="u-label">只读</label><el-switch v-model="u.readOnly" /></div>
          <div class="u-cell u-del"><el-button text type="danger" :icon="Trash2" aria-label="删除用户" @click="settings.users.splice(i, 1)" /></div>
        </div>
        <el-empty v-if="!settings.users.length" description="暂无用户" :image-size="60" />
      </div>
    </div>

    <LogCard url="/api/ftp/logs" clearable />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { HardDrive, Save, Server, Trash2, UserPlus, Users } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import { useFormLayout } from '../composables/useIsMobile'

interface User {
  name: string
  password: string
  root: string
  readOnly: boolean
}
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
  users: User[]
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
  const r = await api.get<{ settings: Settings & { users: User[] | null }; running: boolean; error: string }>('/api/ftp/settings')
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

.u-head,
.u-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) minmax(0, 2fr) 60px 40px;
  gap: 10px;
  align-items: center;
}
.u-head {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  padding-bottom: 6px;
}
.u-row {
  padding: 8px 0;
  border-top: 1px solid var(--g-border);
}
.u-label {
  display: none;
}
@media (max-width: 767.98px) {
  .w-260 {
    width: 100%;
  }
  .u-head {
    display: none;
  }
  .u-row {
    grid-template-columns: 1fr 1fr;
    padding: 12px;
    margin-bottom: 10px;
    border: 1px solid var(--g-border);
    border-radius: 8px;
    position: relative;
  }
  .u-label {
    display: block;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin-bottom: 2px;
  }
  .u-root {
    grid-column: 1 / -1;
  }
  .u-ro {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .u-ro .u-label {
    margin: 0;
  }
  .u-del {
    display: flex;
    justify-content: flex-end;
  }
}
</style>
