<template>
  <div class="terminal-page">
    <el-card shadow="never" class="side">
      <template #header>
        <div class="side-head">
          <span>连接</span>
          <div>
            <el-button link @click="openSettings">设置</el-button>
            <el-button link type="primary" @click="openEditor()">添加</el-button>
          </div>
        </div>
      </template>
      <el-empty v-if="!conns.length" description="暂无连接" :image-size="60" />
      <div v-for="c in conns" :key="c.key" class="conn" @dblclick="connect(c)">
        <div class="conn-main">
          <div class="conn-name">{{ c.name }}</div>
          <div class="conn-addr mono">{{ c.type === 'local' ? '本机 Shell' : `${c.ssh.username}@${c.ssh.host}:${c.ssh.port}` }}</div>
        </div>
        <div class="conn-ops">
          <el-button size="small" type="primary" plain @click="connect(c)">连接</el-button>
          <el-dropdown trigger="click" @command="(cmd: string) => onCommand(c, cmd)">
            <el-button size="small" link>更多</el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="edit">编辑</el-dropdown-item>
                <el-dropdown-item v-if="c.type === 'ssh'" command="resetkey">重置主机密钥</el-dropdown-item>
                <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </el-card>

    <div class="main">
      <el-tabs v-if="tabs.length" v-model="active" type="card" closable class="tabs" @tab-remove="closeTab">
        <el-tab-pane v-for="t in tabs" :key="t.id" :name="t.id" :label="t.title">
          <div class="term-wrap">
            <TermPane :conn-key="t.connKey" />
          </div>
        </el-tab-pane>
      </el-tabs>
      <el-empty v-else description="选择左侧连接开始会话（双击或点击连接）" class="placeholder" />
    </div>

    <el-dialog v-model="editorVisible" :title="editing.key ? '编辑连接' : '添加连接'" width="600px">
      <el-form :model="editing" label-width="100px">
        <el-form-item label="名称"><el-input v-model="editing.name" /></el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model="editing.type">
            <el-radio-button value="ssh">SSH</el-radio-button>
            <el-radio-button value="local">本机 Shell</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <template v-if="editing.type === 'ssh'">
          <el-form-item label="主机"><el-input v-model="editing.ssh.host" /></el-form-item>
          <el-form-item label="端口"><el-input-number v-model="editing.ssh.port" :min="1" :max="65535" /></el-form-item>
          <el-form-item label="用户名"><el-input v-model="editing.ssh.username" /></el-form-item>
          <el-form-item label="认证方式">
            <el-radio-group v-model="editing.ssh.authType">
              <el-radio-button value="password">密码</el-radio-button>
              <el-radio-button value="key">私钥</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="editing.ssh.authType === 'password'" label="密码">
            <el-input v-model="editing.ssh.password" type="password" show-password placeholder="已保存的密码显示为 ********" />
          </el-form-item>
          <template v-else>
            <el-form-item label="私钥">
              <el-input v-model="editing.ssh.privateKey" type="textarea" :rows="4" class="mono" placeholder="-----BEGIN OPENSSH PRIVATE KEY-----" />
            </el-form-item>
            <el-form-item label="私钥密码"><el-input v-model="editing.ssh.passphrase" type="password" show-password /></el-form-item>
          </template>
          <el-form-item label="保活(秒)"><el-input-number v-model="editing.ssh.keepAlive" :min="0" :max="600" /></el-form-item>
          <el-form-item v-if="editing.ssh.hostKey" label="主机指纹">
            <span class="mono small">{{ editing.ssh.hostKey }}</span>
          </el-form-item>
        </template>
        <template v-else>
          <el-form-item label="Shell"><el-input v-model="editing.local.shell" placeholder="留空使用系统默认，例如 /bin/bash" /></el-form-item>
          <el-form-item label="工作目录"><el-input v-model="editing.local.workDir" /></el-form-item>
        </template>
        <el-form-item label="备注"><el-input v-model="editing.remark" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="settingsVisible" title="Web 终端设置" width="480px">
      <el-form v-if="settings" label-width="150px">
        <el-form-item label="允许本机 Shell">
          <el-switch v-model="settings.allowLocalShell" />
        </el-form-item>
        <el-alert v-if="settings.allowLocalShell" type="warning" :closable="false" class="warn">
          本机 Shell 以 Gately 的运行用户执行命令，等同于服务器的完全控制权。请确保管理界面不暴露在公网且使用强密码。
        </el-alert>
        <el-form-item label="最大并发会话"><el-input-number v-model="settings.maxSessions" :min="1" :max="100" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="settingsVisible = false">取消</el-button>
        <el-button type="primary" @click="saveSettings">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import TermPane from '../components/TermPane.vue'

interface Conn {
  key?: string
  name: string
  type: string
  ssh: { host: string; port: number; username: string; authType: string; password: string; privateKey: string; passphrase: string; hostKey: string; keepAlive: number }
  local: { shell: string; workDir: string }
  remark: string
  enable: boolean
}

interface Tab {
  id: string
  connKey: string
  title: string
}

const conns = ref<(Conn & { key: string })[]>([])
const tabs = ref<Tab[]>([])
const active = ref('')
const editorVisible = ref(false)
const editing = ref<Conn>(emptyConn())
const settingsVisible = ref(false)
const settings = ref<{ allowLocalShell: boolean; maxSessions: number }>()
let seq = 0

function emptyConn(): Conn {
  return {
    name: '',
    type: 'ssh',
    ssh: { host: '', port: 22, username: 'root', authType: 'password', password: '', privateKey: '', passphrase: '', hostKey: '', keepAlive: 60 },
    local: { shell: '', workDir: '' },
    remark: '',
    enable: true,
  }
}

async function load() {
  conns.value = await api.get('/api/webterminal/connections')
}

function connect(c: Conn & { key: string }) {
  const id = `t${++seq}`
  tabs.value.push({ id, connKey: c.key, title: c.name })
  active.value = id
}

function closeTab(id: string | number) {
  const i = tabs.value.findIndex((t) => t.id === id)
  tabs.value.splice(i, 1)
  if (active.value === id && tabs.value.length) {
    active.value = tabs.value[Math.max(0, i - 1)].id
  }
}

function openEditor(c?: Conn) {
  editing.value = c ? JSON.parse(JSON.stringify(c)) : emptyConn()
  editorVisible.value = true
}

async function save() {
  try {
    const { key, ...body } = editing.value
    if (key) {
      await api.put(`/api/webterminal/connections/${key}`, body)
    } else {
      await api.post('/api/webterminal/connections', body)
    }
    editorVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function onCommand(c: Conn & { key: string }, cmd: string) {
  if (cmd === 'edit') {
    openEditor(c)
  } else if (cmd === 'resetkey') {
    await api.del(`/api/webterminal/connections/${c.key}/hostkey`)
    ElMessage.success('已重置，下次连接时重新记录主机指纹')
    await load()
  } else if (cmd === 'delete') {
    await ElMessageBox.confirm(`删除连接 ${c.name}？`, '确认', { type: 'warning' })
    await api.del(`/api/webterminal/connections/${c.key}`)
    await load()
  }
}

async function openSettings() {
  const r = await api.get<{ settings: { allowLocalShell: boolean; maxSessions: number } }>('/api/webterminal/settings')
  settings.value = r.settings
  settingsVisible.value = true
}

async function saveSettings() {
  try {
    await api.put('/api/webterminal/settings', settings.value)
    settingsVisible.value = false
    ElMessage.success('已保存')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(load)
</script>

<style scoped>
.terminal-page {
  display: flex;
  gap: 14px;
  height: calc(100vh - 100px);
}
.side {
  width: 280px;
  flex-shrink: 0;
  overflow: auto;
}
.side-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.conn {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 4px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.conn-name {
  font-weight: 500;
}
.conn-addr,
.small {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.conn-ops {
  display: flex;
  gap: 4px;
  align-items: center;
}
.main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
}
.tabs :deep(.el-tabs__content) {
  flex: 1;
}
.tabs :deep(.el-tab-pane) {
  height: 100%;
}
.term-wrap {
  height: calc(100vh - 170px);
}
.placeholder {
  margin-top: 120px;
}
.warn {
  margin-bottom: 14px;
}
</style>
