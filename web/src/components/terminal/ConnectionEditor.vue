<template>
  <el-dialog
    :model-value="modelValue"
    :title="form.key ? '编辑连接' : '添加连接'"
    width="640px"
    :fullscreen="isMobile"
    class="term-dialog"
    :close-on-click-modal="false"
    destroy-on-close
    @update:model-value="(v: boolean) => $emit('update:modelValue', v)"
    @open="onOpen"
  >
    <el-form :model="form" label-width="100px" :label-position="labelPosition">
      <el-form-item label="名称" required><el-input v-model="form.name" placeholder="例如：生产服务器" /></el-form-item>
      <el-form-item label="类型">
        <el-radio-group v-model="form.type" :disabled="!!form.key">
          <el-radio-button value="ssh">SSH</el-radio-button>
          <el-radio-button value="telnet">Telnet</el-radio-button>
          <el-radio-button value="local">本机 Shell</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="启用"><el-switch v-model="form.enable" /></el-form-item>

      <!-- SSH -->
      <template v-if="form.type === 'ssh'">
        <el-divider content-position="left">SSH 连接</el-divider>
        <el-form-item label="主机" required><el-input v-model="form.ssh.host" placeholder="IP 或域名" /></el-form-item>
        <el-form-item label="端口"><el-input-number v-model="form.ssh.port" :min="1" :max="65535" /></el-form-item>
        <el-form-item label="用户名"><el-input v-model="form.ssh.username" /></el-form-item>
        <el-form-item label="认证方式">
          <el-radio-group v-model="form.ssh.authType">
            <el-radio-button value="password">密码</el-radio-button>
            <el-radio-button value="key">私钥</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="form.ssh.authType === 'password'" label="密码">
          <el-input v-model="form.ssh.password" type="password" show-password placeholder="已保存的密码显示为 ********" />
        </el-form-item>
        <template v-else>
          <el-form-item label="私钥内容">
            <el-input v-model="form.ssh.privateKey" type="textarea" :rows="4" class="mono" placeholder="-----BEGIN OPENSSH PRIVATE KEY-----" />
            <span class="form-help">粘贴私钥全文；留空时使用下方「私钥文件」。</span>
          </el-form-item>
          <el-form-item label="私钥文件">
            <el-input v-model="form.ssh.keyFile" class="mono" placeholder="/root/.ssh/id_ed25519" />
            <span class="form-help">运行 Gately 的服务器上的私钥文件路径。</span>
          </el-form-item>
          <el-form-item label="私钥口令"><el-input v-model="form.ssh.passphrase" type="password" show-password placeholder="私钥未加密可留空" /></el-form-item>
        </template>
        <el-form-item label="工作目录">
          <el-input v-model="form.ssh.workDir" class="mono" placeholder="登录后进入的目录，留空为用户主目录" />
        </el-form-item>
        <el-form-item label="连接超时(秒)"><el-input-number v-model="form.ssh.timeout" :min="1" :max="300" /></el-form-item>
        <el-form-item label="保活(秒)">
          <el-input-number v-model="form.ssh.keepAlive" :min="0" :max="600" />
          <span class="help">每隔多少秒发送一次保活包，0 表示不发送</span>
        </el-form-item>
        <el-form-item v-if="form.ssh.hostKey" label="主机指纹">
          <span class="fp mono">{{ form.ssh.hostKey }}</span>
        </el-form-item>
        <ProxyForm :proxy="form.ssh.proxy" />
      </template>

      <!-- Telnet -->
      <template v-else-if="form.type === 'telnet'">
        <el-divider content-position="left">Telnet 连接</el-divider>
        <el-form-item label="主机" required><el-input v-model="form.telnet.host" placeholder="IP 或域名" /></el-form-item>
        <el-form-item label="端口"><el-input-number v-model="form.telnet.port" :min="1" :max="65535" /></el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="form.telnet.username" placeholder="可选，用于自动应答登录提示" />
        </el-form-item>
        <el-form-item label="密码"><el-input v-model="form.telnet.password" type="password" show-password placeholder="可选" /></el-form-item>
        <el-form-item label="连接超时(秒)"><el-input-number v-model="form.telnet.timeout" :min="1" :max="300" /></el-form-item>
        <ProxyForm :proxy="form.telnet.proxy" />
      </template>

      <!-- 本机 -->
      <template v-else>
        <el-divider content-position="left">本机 Shell</el-divider>
        <el-alert v-if="!allowLocalShell" type="warning" :closable="false" class="gap" title="尚未在设置中开启「允许本机 Shell」，开启前无法连接。" />
        <el-form-item label="Shell">
          <el-select v-model="form.local.shell" filterable allow-create default-first-option clearable placeholder="留空使用系统默认" class="full">
            <el-option v-for="s in shells" :key="s" :label="s" :value="s" />
          </el-select>
        </el-form-item>
        <el-form-item label="工作目录"><el-input v-model="form.local.workDir" class="mono" placeholder="留空为运行用户主目录" /></el-form-item>
        <el-form-item label="环境变量">
          <div class="env">
            <div v-for="(r, i) in envRows" :key="i" class="env-row">
              <el-input v-model="r.k" placeholder="变量名" class="mono" />
              <el-input v-model="r.v" placeholder="值" class="mono" />
              <el-button text type="danger" :icon="Trash2" aria-label="删除变量" @click="envRows.splice(i, 1)" />
            </div>
            <el-button size="small" :icon="Plus" @click="envRows.push({ k: '', v: '' })">添加变量</el-button>
          </div>
        </el-form-item>
      </template>

      <el-divider content-position="left">其他</el-divider>
      <el-form-item v-if="form.type !== 'telnet'" label="快速访问目录">
        <el-input v-model="quickText" type="textarea" :rows="3" class="mono" placeholder="每行一个目录，例如 /etc" />
        <span class="form-help">文件管理（SFTP）面板中的常用目录快捷入口。</span>
      </el-form-item>
      <el-form-item label="备注"><el-input v-model="form.remark" /></el-form-item>
    </el-form>
    <template #footer>
      <div class="foot">
        <el-button v-if="form.type !== 'local'" :loading="testing" :icon="Plug" @click="test">测试连接</el-button>
        <span class="grow" />
        <el-button @click="$emit('update:modelValue', false)">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Plug, Plus, Trash2 } from 'lucide-vue-next'
import { api } from '../../api'
import { useFormLayout } from '../../composables/useIsMobile'
import ProxyForm from './ProxyForm.vue'
import { emptyConn, normalizeConn, type Conn } from './types'

const props = defineProps<{ modelValue: boolean; conn: Conn | null; allowLocalShell: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean]; saved: [] }>()

const { isMobile, labelPosition } = useFormLayout()
const form = reactive<Conn>(emptyConn())
const envRows = ref<{ k: string; v: string }[]>([])
const quickText = ref('')
const shells = ref<string[]>([])
const saving = ref(false)
const testing = ref(false)

function onOpen() {
  Object.assign(form, normalizeConn(props.conn ?? emptyConn()))
  envRows.value = Object.entries(form.local.env).map(([k, v]) => ({ k, v }))
  quickText.value = form.quickAccessDirs.join('\n')
  loadShells()
}

async function loadShells() {
  try {
    shells.value = (await api.get<string[] | null>('/api/webterminal/shells')) ?? []
  } catch {
    shells.value = []
  }
}

// 切换到本机类型时再确保已加载 Shell 列表
watch(
  () => form.type,
  (t) => {
    if (t === 'local' && !shells.value.length) {
      loadShells()
    }
  },
)

// buildBody 把表单整理为提交给后端的 Connection。
function buildBody(): Conn {
  const body = JSON.parse(JSON.stringify(form)) as Conn
  const env: Record<string, string> = {}
  for (const r of envRows.value) {
    if (r.k.trim()) {
      env[r.k.trim()] = r.v
    }
  }
  body.local.env = env
  body.quickAccessDirs = quickText.value
    .split('\n')
    .map((s) => s.trim())
    .filter((s) => s)
  return body
}

function validate(body: Conn): string {
  if (!body.name.trim()) {
    return '请填写连接名称'
  }
  if (body.type === 'ssh' && !body.ssh.host.trim()) {
    return '请填写 SSH 主机'
  }
  if (body.type === 'telnet' && !body.telnet.host.trim()) {
    return '请填写 Telnet 主机'
  }
  return ''
}

async function save() {
  const body = buildBody()
  const msg = validate(body)
  if (msg) {
    ElMessage.warning(msg)
    return
  }
  saving.value = true
  try {
    const { key, ...rest } = body
    if (key) {
      await api.put(`/api/webterminal/connections/${encodeURIComponent(key)}`, rest)
    } else {
      await api.post('/api/webterminal/connections', rest)
    }
    emit('update:modelValue', false)
    emit('saved')
    ElMessage.success('已保存')
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

// test 测试当前表单中尚未保存的配置；编辑已有连接时带上 key，让后端合并已保存的密文。
async function test() {
  const body = buildBody()
  const msg = validate(body)
  if (msg) {
    ElMessage.warning(msg)
    return
  }
  testing.value = true
  try {
    const r = await api.post<{ message?: string } | null>('/api/webterminal/connections/test', body)
    ElMessage.success(r?.message || '连接成功')
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    testing.value = false
  }
}
</script>

<style scoped>
.full {
  width: 100%;
}
.gap {
  margin-bottom: 14px;
}
.fp {
  font-size: 12px;
  word-break: break-all;
  color: var(--el-text-color-secondary);
}
.env {
  width: 100%;
}
.env-row {
  display: flex;
  gap: 6px;
  margin-bottom: 6px;
}
.env-row .el-input:first-child {
  flex: 0 0 36%;
}
.foot {
  display: flex;
  align-items: center;
  gap: 8px;
}
.grow {
  flex: 1;
}
@media (max-width: 767.98px) {
  .foot {
    flex-wrap: wrap;
  }
}
</style>
