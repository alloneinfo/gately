<template>
  <div>
    <div class="page-header">
      <h2>WebDAV</h2>
      <el-button type="primary" :loading="saving" @click="save">保存</el-button>
    </div>
    <el-alert v-if="error" type="error" :title="error" :closable="false" class="block" />
    <el-card v-if="settings" shadow="never" header="服务">
      <el-form label-width="100px">
        <el-form-item label="启用">
          <el-switch v-model="settings.enable" />
          <el-tag v-if="running" type="success" size="small" class="tag">运行中</el-tag>
        </el-form-item>
        <el-form-item label="监听地址"><el-input v-model="settings.listenIp" placeholder="留空表示所有地址" style="width: 260px" /></el-form-item>
        <el-form-item label="端口"><el-input-number v-model="settings.port" :min="1" :max="65535" /></el-form-item>
        <el-form-item label="HTTPS">
          <el-switch v-model="settings.tls" />
          <span class="help">证书按访问域名从「SSL 证书」中选择</span>
        </el-form-item>
      </el-form>
    </el-card>
    <el-card v-if="settings" shadow="never" class="block">
      <template #header>
        <div class="head">
          <span>用户</span>
          <el-button size="small" @click="settings.users.push({ name: '', password: '', root: '', readOnly: false })">添加用户</el-button>
        </div>
      </template>
      <el-table :data="settings.users" empty-text="暂无用户">
        <el-table-column label="用户名" min-width="140">
          <template #default="{ row }"><el-input v-model="row.name" size="small" /></template>
        </el-table-column>
        <el-table-column label="密码" min-width="160">
          <template #default="{ row }"><el-input v-model="row.password" type="password" show-password size="small" /></template>
        </el-table-column>
        <el-table-column label="目录" min-width="240">
          <template #default="{ row }"><el-input v-model="row.root" size="small" placeholder="/data/share" /></template>
        </el-table-column>
        <el-table-column label="只读" width="80">
          <template #default="{ row }"><el-switch v-model="row.readOnly" size="small" /></template>
        </el-table-column>
        <el-table-column width="70">
          <template #default="{ $index }"><el-button link type="danger" @click="settings.users.splice($index, 1)">删除</el-button></template>
        </el-table-column>
      </el-table>
    </el-card>
    <el-card shadow="never" header="日志" class="block">
      <LogViewer url="/api/webdav/logs" clearable />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import LogViewer from '../components/LogViewer.vue'

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
  tls: boolean
  users: User[]
}

const settings = ref<Settings>()
const running = ref(false)
const error = ref('')
const saving = ref(false)

async function load() {
  const r = await api.get<{ settings: Settings & { users: User[] | null }; running: boolean; error: string }>('/api/webdav/settings')
  settings.value = { ...r.settings, users: r.settings.users ?? [] }
  running.value = r.running
  error.value = r.error
}

async function save() {
  saving.value = true
  try {
    const r = await api.put<{ running: boolean; error: string }>('/api/webdav/settings', settings.value)
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
.block {
  margin-top: 16px;
}
.head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.tag {
  margin-left: 10px;
}
.help {
  margin-left: 10px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
