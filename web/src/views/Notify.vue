<template>
  <div>
    <PageHeader title="通知中心" desc="把 DDNS 变更、证书到期、WAF 拦截、进程异常等事件推送到 Telegram、邮件、企业微信等渠道" :icon="Bell">
      <el-button :icon="Plus" @click="openRule()">添加规则</el-button>
      <el-button type="primary" :icon="Plus" @click="openChannel()">添加渠道</el-button>
    </PageHeader>

    <h3 class="section">通知渠道</h3>
    <div class="card-grid">
      <el-empty v-if="!channels.length" description="暂无渠道，先添加一个用于接收通知的渠道" class="empty" />
      <EntityCard v-for="c in channels" :key="c.key" :off="!c.enable">
        <template #lead>
          <el-switch :model-value="c.enable" @change="(v: string | number | boolean) => toggleChannel(c, Boolean(v))" />
        </template>
        <template #title>{{ c.name }}</template>
        <template #sub>{{ kindLabel(c.kind) }}</template>
        <Field label="目标" mono>{{ channelTarget(c) }}</Field>
        <Field v-if="c.remark" label="备注">{{ c.remark }}</Field>
        <template #actions>
          <el-button text size="small" :icon="Send" :loading="testing[c.key ?? '']" @click="test(c)">测试</el-button>
          <el-button text size="small" :icon="Pencil" @click="openChannel(c)">编辑</el-button>
          <el-popconfirm title="删除该渠道？引用它的规则将不再向它推送" @confirm="removeChannel(c)">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Trash2">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </EntityCard>
    </div>

    <h3 class="section">通知规则</h3>
    <div class="card-grid">
      <el-empty v-if="!rules.length" description="暂无规则，添加规则选择哪些事件推送到哪些渠道" class="empty" />
      <EntityCard v-for="r in rules" :key="r.key" :off="!r.enable">
        <template #lead>
          <el-switch :model-value="r.enable" @change="(v: string | number | boolean) => toggleRule(r, Boolean(v))" />
        </template>
        <template #title>{{ r.name }}</template>
        <template #sub>级别 ≥ {{ levelLabel(r.minLevel) }}<span v-if="r.silence">，静默 {{ r.silence }} 秒</span></template>
        <Field label="事件">
          <div class="tags">
            <el-tag v-for="e in r.events" :key="e" size="small" effect="plain">{{ eventName(e) }}</el-tag>
          </div>
        </Field>
        <Field label="渠道">{{ r.channels.map(channelName).join('、') || '-' }}</Field>
        <template #actions>
          <el-button text size="small" :icon="Pencil" @click="openRule(r)">编辑</el-button>
          <el-popconfirm title="删除该规则？" @confirm="removeRule(r)">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Trash2">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </EntityCard>
    </div>

    <h3 class="section">最近投递</h3>
    <el-table :data="history" size="small" stripe empty-text="暂无投递记录">
      <el-table-column label="时间" width="170">
        <template #default="{ row }">{{ new Date(row.time).toLocaleString() }}</template>
      </el-table-column>
      <el-table-column label="事件" min-width="200" prop="title" show-overflow-tooltip />
      <el-table-column label="规则" width="140" prop="rule" show-overflow-tooltip />
      <el-table-column label="渠道" width="140" prop="channel" show-overflow-tooltip />
      <el-table-column label="结果" width="90">
        <template #default="{ row }">
          <StatusBadge v-if="row.ok" type="running" text="成功" />
          <StatusBadge v-else type="error" text="失败" :title="row.error" />
        </template>
      </el-table-column>
    </el-table>

    <LogCard url="/api/notify/logs" clearable />

    <el-dialog v-model="channelVisible" :title="editingChannel.key ? '编辑渠道' : '添加渠道'" width="560px" :fullscreen="isMobile">
      <el-form :model="editingChannel" label-width="100px" :label-position="labelPosition">
        <el-form-item label="名称"><el-input v-model="editingChannel.name" /></el-form-item>
        <el-form-item label="类型">
          <el-select v-model="editingChannel.kind" :disabled="!!editingChannel.key" style="width: 100%">
            <el-option v-for="k in kinds" :key="k.value" :label="k.label" :value="k.value" />
          </el-select>
        </el-form-item>

        <template v-if="editingChannel.kind === 'webhook'">
          <el-form-item label="请求地址"><el-input v-model="editingChannel.webhook.url" placeholder="https://example.com/hook" /></el-form-item>
          <el-form-item label="请求方法">
            <el-select v-model="editingChannel.webhook.method" style="width: 120px">
              <el-option v-for="m in ['GET', 'POST', 'PUT', 'PATCH']" :key="m" :label="m" :value="m" />
            </el-select>
          </el-form-item>
          <el-form-item label="请求头"><el-input v-model="headersText" type="textarea" :rows="2" placeholder="每行一个，格式 Key: Value" /></el-form-item>
          <el-form-item v-if="editingChannel.webhook.method !== 'GET'" label="请求体">
            <el-input v-model="editingChannel.webhook.body" type="textarea" :rows="4" class="mono" />
          </el-form-item>
          <el-form-item label="可用变量">
            <div class="form-help mono">{title} {message} {text} {type} {source} {level} {time}</div>
          </el-form-item>
        </template>

        <template v-else-if="editingChannel.kind === 'telegram'">
          <el-form-item label="Bot Token"><el-input v-model="editingChannel.secret" show-password :placeholder="secretHint" /></el-form-item>
          <el-form-item label="Chat ID"><el-input v-model="editingChannel.target" /></el-form-item>
          <el-form-item label="API 地址"><el-input v-model="editingChannel.url" placeholder="留空使用 https://api.telegram.org" /></el-form-item>
        </template>

        <template v-else-if="editingChannel.kind === 'bark'">
          <el-form-item label="设备 Key"><el-input v-model="editingChannel.secret" show-password :placeholder="secretHint" /></el-form-item>
          <el-form-item label="服务地址"><el-input v-model="editingChannel.url" placeholder="留空使用 https://api.day.app" /></el-form-item>
        </template>

        <template v-else-if="editingChannel.kind === 'smtp'">
          <el-form-item label="服务器"><el-input v-model="editingChannel.host" placeholder="smtp.example.com" /></el-form-item>
          <el-form-item label="加密方式">
            <el-select v-model="editingChannel.security" style="width: 180px">
              <el-option label="STARTTLS（587）" value="starttls" />
              <el-option label="SSL/TLS（465）" value="tls" />
              <el-option label="无加密（25）" value="none" />
            </el-select>
          </el-form-item>
          <el-form-item label="端口"><el-input-number v-model="editingChannel.port" :min="0" :max="65535" /></el-form-item>
          <el-form-item label="用户名"><el-input v-model="editingChannel.user" /></el-form-item>
          <el-form-item label="密码"><el-input v-model="editingChannel.secret" show-password :placeholder="secretHint" /></el-form-item>
          <el-form-item label="发件人"><el-input v-model="editingChannel.from" placeholder="留空使用用户名" /></el-form-item>
          <el-form-item label="收件人"><el-input v-model="editingChannel.target" placeholder="多个用逗号分隔" /></el-form-item>
        </template>

        <template v-else>
          <el-form-item label="机器人地址"><el-input v-model="editingChannel.url" placeholder="https://..." /></el-form-item>
          <el-form-item v-if="editingChannel.kind !== 'wecom'" label="签名密钥">
            <el-input v-model="editingChannel.secret" show-password :placeholder="secretHint" />
            <div class="form-help">机器人启用「加签」时填写，否则留空</div>
          </el-form-item>
        </template>

        <el-form-item label="备注"><el-input v-model="editingChannel.remark" /></el-form-item>
        <el-form-item label="启用"><el-switch v-model="editingChannel.enable" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="channelVisible = false">取消</el-button>
        <el-button type="primary" @click="saveChannel">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="ruleVisible" :title="editingRule.key ? '编辑规则' : '添加规则'" width="560px" :fullscreen="isMobile">
      <el-form :model="editingRule" label-width="100px" :label-position="labelPosition">
        <el-form-item label="名称"><el-input v-model="editingRule.name" /></el-form-item>
        <el-form-item label="事件">
          <el-select v-model="editingRule.events" multiple filterable allow-create default-first-option style="width: 100%" placeholder="选择事件；也可输入 ddns.* 这样的通配">
            <el-option label="全部事件" value="*" />
            <el-option v-for="t in eventTypes" :key="t.type" :label="t.name" :value="t.type" />
          </el-select>
        </el-form-item>
        <el-form-item label="最低级别">
          <el-select v-model="editingRule.minLevel" style="width: 140px">
            <el-option label="信息" value="info" />
            <el-option label="警告" value="warn" />
            <el-option label="错误" value="error" />
          </el-select>
        </el-form-item>
        <el-form-item label="渠道">
          <el-select v-model="editingRule.channels" multiple style="width: 100%" placeholder="选择接收通知的渠道">
            <el-option v-for="c in channels" :key="c.key" :label="c.name" :value="c.key ?? ''" />
          </el-select>
        </el-form-item>
        <el-form-item label="静默(秒)">
          <el-input-number v-model="editingRule.silence" :min="0" :step="60" />
          <div class="form-help">同一来源的同类事件在该时间内只推送一次，0 表示不去重</div>
        </el-form-item>
        <el-form-item label="启用"><el-switch v-model="editingRule.enable" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ruleVisible = false">取消</el-button>
        <el-button type="primary" @click="saveRule">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Bell, Pencil, Plus, Send, Trash2 } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import EntityCard from '../components/EntityCard.vue'
import Field from '../components/Field.vue'
import StatusBadge from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import { useFormLayout } from '../composables/useIsMobile'

interface WebhookCfg {
  enable: boolean
  url: string
  method: string
  headers: string[]
  body: string
  successContent: string[]
  timeout: number
  insecureSkipVerify: boolean
}

interface Channel {
  key?: string
  name: string
  enable: boolean
  kind: string
  url: string
  secret: string
  hasSecret?: boolean
  target: string
  host: string
  port: number
  security: string
  user: string
  from: string
  webhook: WebhookCfg
  remark: string
}

interface Rule {
  key?: string
  name: string
  enable: boolean
  events: string[]
  minLevel: string
  channels: string[]
  silence: number
}

interface EventType {
  type: string
  name: string
  level: string
}

interface DeliveryRecord {
  time: string
  type: string
  title: string
  rule: string
  channel: string
  ok: boolean
  error?: string
}

const kinds = [
  { value: 'webhook', label: 'Webhook' },
  { value: 'telegram', label: 'Telegram' },
  { value: 'bark', label: 'Bark' },
  { value: 'wecom', label: '企业微信机器人' },
  { value: 'dingtalk', label: '钉钉机器人' },
  { value: 'feishu', label: '飞书机器人' },
  { value: 'smtp', label: '邮件 SMTP' },
]

const { isMobile, labelPosition } = useFormLayout()
const channels = ref<Channel[]>([])
const rules = ref<Rule[]>([])
const eventTypes = ref<EventType[]>([])
const history = ref<DeliveryRecord[]>([])
const testing = reactive<Record<string, boolean>>({})
const channelVisible = ref(false)
const ruleVisible = ref(false)
const editingChannel = ref<Channel>(emptyChannel())
const editingRule = ref<Rule>(emptyRule())
let timer: number | undefined

const secretHint = computed(() => (editingChannel.value.hasSecret ? '已设置，留空保持不变' : ''))
const headersText = computed({
  get: () => (editingChannel.value.webhook.headers ?? []).join('\n'),
  set: (v: string) => (editingChannel.value.webhook.headers = v.split('\n').map((s) => s.trim()).filter(Boolean)),
})

function emptyChannel(): Channel {
  return {
    name: '',
    enable: true,
    kind: 'telegram',
    url: '',
    secret: '',
    target: '',
    host: '',
    port: 0,
    security: 'starttls',
    user: '',
    from: '',
    webhook: { enable: true, url: '', method: 'POST', headers: [], body: '{"text":"{title}\\n{text}"}', successContent: [], timeout: 0, insecureSkipVerify: false },
    remark: '',
  }
}

function emptyRule(): Rule {
  return { name: '', enable: true, events: [], minLevel: 'info', channels: [], silence: 300 }
}

function kindLabel(k: string) {
  return kinds.find((x) => x.value === k)?.label ?? k
}

function levelLabel(l: string) {
  return ({ info: '信息', warn: '警告', error: '错误' } as Record<string, string>)[l] ?? l
}

function eventName(e: string) {
  if (e === '*') {
    return '全部事件'
  }
  return eventTypes.value.find((t) => t.type === e)?.name ?? e
}

function channelName(key: string) {
  return channels.value.find((c) => c.key === key)?.name ?? '（已删除）'
}

function channelTarget(c: Channel) {
  switch (c.kind) {
    case 'webhook':
      return c.webhook.url
    case 'telegram':
      return `Chat ${c.target}`
    case 'bark':
      return c.url || 'api.day.app'
    case 'smtp':
      return `${c.target} via ${c.host}:${c.port}`
    default:
      return c.url
  }
}

async function load() {
  const [c, r, h] = await Promise.all([
    api.get<Channel[]>('/api/notify/channels'),
    api.get<Rule[]>('/api/notify/rules'),
    api.get<DeliveryRecord[]>('/api/notify/history'),
  ])
  channels.value = c
  rules.value = r
  history.value = h
}

function openChannel(c?: Channel) {
  editingChannel.value = c ? { ...emptyChannel(), ...JSON.parse(JSON.stringify(c)) } : emptyChannel()
  channelVisible.value = true
}

function openRule(r?: Rule) {
  editingRule.value = r ? JSON.parse(JSON.stringify(r)) : emptyRule()
  ruleVisible.value = true
}

async function saveChannel() {
  try {
    const { key, hasSecret: _h, ...body } = editingChannel.value
    if (key) {
      await api.put(`/api/notify/channels/${key}`, body)
    } else {
      await api.post('/api/notify/channels', body)
    }
    channelVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function saveRule() {
  try {
    const { key, ...body } = editingRule.value
    if (key) {
      await api.put(`/api/notify/rules/${key}`, body)
    } else {
      await api.post('/api/notify/rules', body)
    }
    ruleVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function toggleChannel(c: Channel, enable: boolean) {
  try {
    await api.put(`/api/notify/channels/${c.key}/enable`, { enable })
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function toggleRule(r: Rule, enable: boolean) {
  try {
    await api.put(`/api/notify/rules/${r.key}/enable`, { enable })
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function removeChannel(c: Channel) {
  try {
    await api.del(`/api/notify/channels/${c.key}`)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function removeRule(r: Rule) {
  try {
    await api.del(`/api/notify/rules/${r.key}`)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function test(c: Channel) {
  const k = c.key ?? ''
  testing[k] = true
  try {
    await api.post(`/api/notify/channels/${k}/test`)
    ElMessage.success('测试通知已发送')
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    testing[k] = false
    await load()
  }
}

onMounted(async () => {
  eventTypes.value = await api.get('/api/notify/event-types')
  await load()
  timer = window.setInterval(load, 15000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.section {
  margin: 22px 0 10px;
  font-size: 15px;
  font-weight: 600;
}
.empty {
  grid-column: 1 / -1;
}
.tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
</style>
