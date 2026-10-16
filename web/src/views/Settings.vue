<template>
  <div>
    <PageHeader title="设置" desc="账号安全、访问控制、运行参数、备份迁移与系统信息" :icon="Settings2" />

    <div v-if="form?.restartRequired" class="restart-banner">
      <RotateCw :size="18" class="rb-icon" />
      <div class="rb-text">
        <div class="rb-title">部分设置需要重启后生效</div>
        <div class="rb-desc">监听地址等修改已保存，重启 Gately 后生效。</div>
      </div>
      <el-button type="warning" :icon="RotateCw" @click="confirmRestart">立即重启</el-button>
    </div>

    <el-tabs v-model="tab" class="settings-tabs">
      <!-- ============ 账号与安全 ============ -->
      <el-tab-pane name="account">
        <template #label><span class="tab-label"><UserRound :size="15" />账号与安全</span></template>
        <div class="grid-2">
          <div class="g-card">
            <div class="g-card-head">
              <div class="g-card-title"><KeyRound :size="16" />管理员账号</div>
            </div>
            <div class="g-card-body">
              <el-form :model="account" label-width="100px" :label-position="labelPosition" class="narrow">
                <el-form-item label="当前密码" required>
                  <el-input v-model="account.oldPassword" type="password" show-password autocomplete="current-password" />
                </el-form-item>
                <el-form-item label="用户名" required>
                  <el-input v-model="account.username" autocomplete="username" />
                </el-form-item>
                <el-form-item label="新密码" required>
                  <el-input v-model="account.newPassword" type="password" show-password placeholder="至少 8 位" autocomplete="new-password" />
                </el-form-item>
                <el-form-item label="确认新密码" required>
                  <el-input v-model="account.confirm" type="password" show-password autocomplete="new-password" />
                </el-form-item>
                <el-form-item label=" ">
                  <el-button type="primary" :loading="accountSaving" @click="saveAccount">修改账号密码</el-button>
                  <div class="form-help">修改后所有已登录的会话都会失效，需要重新登录。</div>
                </el-form-item>
              </el-form>
            </div>
          </div>

          <div class="g-card">
            <div class="g-card-head">
              <div class="g-card-title"><Smartphone :size="16" />两步验证</div>
              <StatusBadge :type="totp.enabled ? 'ok' : 'off'" :text="totp.enabled ? '已启用' : '未启用'" />
            </div>
            <div class="g-card-body">
              <template v-if="totp.enabled">
                <p class="desc">登录时除密码外还需要输入验证器 App 中的 6 位动态验证码。</p>
                <el-form label-width="100px" :label-position="labelPosition" class="narrow">
                  <el-form-item label="当前密码"><el-input v-model="totp.password" type="password" show-password /></el-form-item>
                  <el-form-item label="验证码"><el-input v-model="totp.code" maxlength="6" inputmode="numeric" class="mono" placeholder="验证器中的 6 位数字" /></el-form-item>
                  <el-form-item label=" "><el-button type="danger" plain @click="disableTOTP">关闭两步验证</el-button></el-form-item>
                </el-form>
              </template>
              <template v-else-if="totp.secret">
                <p class="desc">用 Google Authenticator、Microsoft Authenticator 等 App 扫描二维码，然后输入显示的验证码确认。</p>
                <div class="qr-wrap">
                  <img v-if="totp.qr" :src="totp.qr" class="qr" alt="二维码" />
                  <div class="secret-box">
                    <div class="meta">无法扫码时手动输入密钥</div>
                    <div class="mono secret">{{ totp.secret }}</div>
                    <el-button size="small" text :icon="Copy" @click="copy(totp.secret)">复制密钥</el-button>
                  </div>
                </div>
                <el-form label-width="100px" :label-position="labelPosition" class="narrow">
                  <el-form-item label="验证码"><el-input v-model="totp.code" maxlength="6" inputmode="numeric" class="mono" /></el-form-item>
                  <el-form-item label="当前密码"><el-input v-model="totp.password" type="password" show-password /></el-form-item>
                  <el-form-item label=" ">
                    <el-button type="primary" @click="enableTOTP">确认启用</el-button>
                    <el-button @click="cancelTOTP">取消</el-button>
                  </el-form-item>
                </el-form>
              </template>
              <template v-else>
                <p class="desc">启用后登录需要额外输入验证器 App 生成的动态验证码，即使密码泄露也能保护管理界面。</p>
                <el-button type="primary" plain :icon="ShieldCheck" @click="setupTOTP">开始设置</el-button>
              </template>
            </div>
          </div>
        </div>

        <div v-if="form" class="g-card">
          <div class="g-card-head">
            <div class="g-card-title"><LockKeyhole :size="16" />登录保护</div>
          </div>
          <div class="g-card-body">
            <el-form :model="form" label-width="140px" :label-position="labelPosition">
              <el-form-item label="会话有效期">
                <el-input-number v-model="form.sessionHours" :min="1" :max="8760" />
                <span class="help">小时；超过后需要重新登录</span>
              </el-form-item>
              <el-form-item label="连续失败上限">
                <el-input-number v-model="form.maxLoginFailures" :min="0" :max="100" />
                <span class="help">同一来源连续输错密码达到次数后锁定，0 使用默认值</span>
              </el-form-item>
              <el-form-item label="锁定时长">
                <el-input-number v-model="form.lockMinutes" :min="0" :max="10080" />
                <span class="help">分钟，0 使用默认值</span>
              </el-form-item>
            </el-form>
          </div>
        </div>

        <div v-if="form" class="g-card">
          <div class="g-card-head">
            <div class="g-card-title"><Code2 :size="16" />API 令牌</div>
            <StatusBadge :type="form.apiToken ? 'ok' : 'off'" :text="form.apiToken ? '已启用' : '未启用'" />
          </div>
          <div class="g-card-body">
            <p class="desc">
              设置后，脚本可以通过请求头 <span class="mono code">Authorization: Bearer &lt;令牌&gt;</span> 调用管理 API，无需登录。令牌等同于管理员权限，请妥善保管。
            </p>
            <div class="token-row">
              <el-input
                v-model="form.apiToken"
                :placeholder="'未设置（不允许令牌访问）'"
                class="mono token-input"
                :readonly="form.apiToken === mask"
              />
              <div class="token-ops">
                <el-button :icon="Sparkles" @click="genToken">生成</el-button>
                <el-button :icon="Copy" :disabled="!form.apiToken || form.apiToken === mask" @click="copy(form.apiToken)">复制</el-button>
                <el-button :icon="Eraser" :disabled="!form.apiToken" @click="form.apiToken = ''">清空</el-button>
              </div>
            </div>
            <div class="form-help">
              <template v-if="form.apiToken === mask">已保存的令牌不会再次显示；如遗失请生成新令牌。</template>
              <template v-else-if="form.apiToken">新令牌保存后生效，请先复制保存。</template>
              <template v-else>保存后将停用令牌访问。</template>
            </div>
          </div>
        </div>
        <SaveBar :dirty="dirty" :saving="saving" @save="saveSettings" @reset="resetSettings" />
      </el-tab-pane>

      <!-- ============ 访问控制 ============ -->
      <el-tab-pane name="access">
        <template #label><span class="tab-label"><ShieldCheck :size="15" />访问控制</span></template>
        <div v-if="form" class="g-card">
          <div class="g-card-head">
            <div class="g-card-title"><Network :size="16" />管理界面监听</div>
          </div>
          <div class="g-card-body">
            <el-form :model="form" label-width="140px" :label-position="labelPosition">
              <el-form-item label="HTTP 监听">
                <el-input v-model="form.listen" placeholder=":16680" class="w-280 mono" />
                <span class="help"><el-tag size="small" type="warning">需重启</el-tag> 格式为 :端口 或 IP:端口</span>
              </el-form-item>
              <el-form-item label="HTTPS 监听">
                <el-input v-model="form.httpsListen" placeholder="留空不启用，例如 :16643" class="w-280 mono" />
                <span class="help"><el-tag size="small" type="warning">需重启</el-tag> 证书按访问域名从「SSL 证书」中选择</span>
              </el-form-item>
              <el-form-item label="强制 HTTPS">
                <el-switch v-model="form.forceHTTPS" :disabled="!form.httpsListen && !form.forceHTTPS" />
                <span class="help">HTTP 访问自动跳转到 HTTPS（仅当证书模块有该域名的证书时）</span>
              </el-form-item>
            </el-form>
          </div>
        </div>
        <div v-if="form" class="g-card">
          <div class="g-card-head">
            <div class="g-card-title"><DoorClosed :size="16" />访问限制</div>
          </div>
          <div class="g-card-body">
            <el-form :model="form" label-width="140px" :label-position="labelPosition">
              <el-form-item label="安全入口">
                <el-input v-model="form.safePath" placeholder="留空不启用，例如 /my-entry" class="w-280 mono">
                  <template #append>
                    <el-button :icon="Sparkles" aria-label="随机生成" @click="randomEntry" />
                  </template>
                </el-input>
                <div class="form-help">
                  设置后需访问 <span class="mono code">http://IP:端口{{ entryPath || '/<入口>' }}</span> 才能打开管理界面，直接访问其他路径返回 404，可防止扫描器发现管理界面。请牢记入口地址。
                </div>
              </el-form-item>
              <el-form-item label="仅允许内网访问">
                <el-switch v-model="form.lanOnly" />
                <span class="help">只允许内网地址（含 Tailscale 等 100.64.0.0/10 地址）访问管理界面</span>
              </el-form-item>
            </el-form>
          </div>
        </div>
        <SaveBar :dirty="dirty" :saving="saving" @save="saveSettings" @reset="resetSettings" />
      </el-tab-pane>

      <!-- ============ 网络与运行 ============ -->
      <el-tab-pane name="runtime">
        <template #label><span class="tab-label"><Cpu :size="15" />网络与运行</span></template>
        <div v-if="form" class="g-card">
          <div class="g-card-head">
            <div class="g-card-title"><Globe :size="16" />网络</div>
          </div>
          <div class="g-card-body">
            <el-form :model="form" label-width="140px" :label-position="labelPosition">
              <el-form-item label="DNS 服务器">
                <el-select
                  v-model="dnsServers"
                  multiple
                  filterable
                  allow-create
                  default-first-option
                  :reserve-keyword="false"
                  placeholder="留空使用系统 DNS；可输入 IP 或 IP:端口"
                  class="w-480"
                >
                  <el-option v-for="d in dnsPresets" :key="d.value" :label="`${d.value}（${d.label}）`" :value="d.value" />
                </el-select>
                <div class="form-help">影响所有出站域名解析（DDNS、Webhook、ACME 等），多个时随机选择并自动故障转移。</div>
              </el-form-item>
              <el-form-item label="时区">
                <el-select v-model="form.timeZone" filterable allow-create default-first-option clearable placeholder="跟随系统" class="w-280">
                  <el-option v-for="z in timeZones" :key="z.value" :label="`${z.label}  ${z.value}`" :value="z.value" />
                </el-select>
                <div class="form-help">日志、计划任务等使用的时区，可输入任意 IANA 时区名；当前生效：{{ info?.timeZone || '-' }}</div>
              </el-form-item>
            </el-form>
          </div>
        </div>
        <div v-if="form" class="g-card">
          <div class="g-card-head">
            <div class="g-card-title"><Gauge :size="16" />运行参数</div>
          </div>
          <div class="g-card-body">
            <el-form :model="form" label-width="140px" :label-position="labelPosition">
              <el-form-item label="日志保留条数">
                <el-input-number v-model="form.logLines" :min="100" :max="20000" :step="500" />
                <span class="help">每个模块在内存中保留的日志条数</span>
              </el-form-item>
              <el-form-item label="GC 百分比">
                <el-input-number v-model="form.gcPercent" :min="0" :max="1000" :step="10" />
                <span class="help">0 为默认（100）；调小可降低内存占用，但增加 CPU 开销</span>
              </el-form-item>
              <el-form-item label="最大 CPU 核数">
                <el-input-number v-model="form.maxProcs" :min="0" :max="1024" />
                <span class="help">0 为自动（本机 {{ info?.numCPU ?? '-' }} 核）</span>
              </el-form-item>
            </el-form>
          </div>
        </div>
        <div class="g-card">
          <div class="g-card-head">
            <div class="g-card-title"><RotateCw :size="16" />重启</div>
          </div>
          <div class="g-card-body restart-row">
            <span class="muted">重启期间管理界面与所有转发、代理服务会短暂中断（通常数秒）。</span>
            <el-button type="warning" plain :icon="RotateCw" @click="confirmRestart">重启 Gately</el-button>
          </div>
        </div>
        <SaveBar :dirty="dirty" :saving="saving" @save="saveSettings" @reset="resetSettings" />
      </el-tab-pane>

      <!-- ============ 备份与迁移 ============ -->
      <el-tab-pane name="backup">
        <template #label><span class="tab-label"><ArchiveRestore :size="15" />备份与迁移</span></template>
        <div class="g-card">
          <div class="g-card-head">
            <div class="g-card-title"><Import :size="16" />从 Lucky 导入</div>
          </div>
          <div class="g-card-body">
            <p class="desc">
              支持 Lucky 配置目录（Docker 默认 <span class="mono code">/app/conf</span>）中的 <span class="mono code">*.lkcf</span> 文件（可多选），以及 Lucky 后台导出的整站备份
              <span class="mono code">.zip</span>。先预览，确认后再导入；导入的条目追加到现有条目之后，同名条目会跳过。
            </p>
            <el-upload v-model:file-list="files" :auto-upload="false" multiple accept=".lkcf,.zip" drag class="upload">
              <UploadCloud :size="32" :stroke-width="1.5" class="upload-icon" />
              <div class="upload-text">拖入文件或 <em>点击选择</em></div>
              <div class="meta">.lkcf 配置文件或整站备份 .zip</div>
            </el-upload>
            <div class="keep-enable">
              <el-switch v-model="keepEnable" />
              <div>
                <div class="ke-title">导入后保持 Lucky 中的启用状态</div>
                <div class="ke-desc">
                  默认关闭：导入的规则全部为停用状态，检查无误后再逐条启用。开启后，在 Lucky 中启用的规则会<b>立即监听端口、更新 DNS 解析</b>，如果 Lucky 仍在运行，可能出现端口冲突或两边同时更新解析。
                </div>
              </div>
            </div>
            <div class="ops">
              <el-button :icon="Eye" :disabled="!files.length" :loading="busy && !importing" @click="runImport(true)">预览</el-button>
              <el-button type="primary" :icon="Import" :disabled="!files.length || !previewed || imported" :loading="importing" @click="confirmImport">确认导入</el-button>
              <span v-if="previewed && !imported" class="meta">预览完成，确认无误后导入</span>
              <span v-if="imported" class="text-ok ops-done"><CircleCheck :size="14" />已导入</span>
            </div>

            <div v-for="r in results" :key="r.file" class="result">
              <div class="result-file"><FileText :size="14" /><span class="mono">{{ r.file }}</span></div>
              <el-alert v-if="r.error" type="error" :title="r.error" :closable="false" show-icon />
              <el-alert v-else-if="!r.reports.length" type="info" title="未包含 Gately 已支持的模块" :closable="false" show-icon />
              <el-table v-else :data="r.reports" size="small" row-key="module" class="result-table">
                <el-table-column type="expand" width="32">
                  <template #default="{ row }">
                    <ul v-if="row.warnings?.length" class="warns">
                      <li v-for="w in row.warnings" :key="w">{{ w }}</li>
                    </ul>
                    <div v-else class="meta warns">没有警告</div>
                  </template>
                </el-table-column>
                <el-table-column prop="title" label="模块" min-width="90" />
                <el-table-column prop="found" label="发现" width="60" align="right" />
                <el-table-column :label="imported ? '已导入' : '可导入'" width="70" align="right">
                  <template #default="{ row }"><b>{{ row.imported }}</b></template>
                </el-table-column>
                <el-table-column label="警告" width="70" align="right">
                  <template #default="{ row }">
                    <span v-if="row.warnings?.length" class="warn-count">{{ row.warnings.length }}</span>
                    <span v-else class="muted">0</span>
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </div>
        </div>

        <div class="grid-2">
          <div class="g-card">
            <div class="g-card-head">
              <div class="g-card-title"><DownloadCloud :size="16" />备份 Gately 配置</div>
            </div>
            <div class="g-card-body">
              <p class="desc">下载包含全部配置（账号、规则、证书等）的 zip 文件。备份中含有密码哈希、密钥与证书私钥，请妥善保存。</p>
              <el-button type="primary" plain :icon="Download" :loading="downloading" @click="downloadBackup">下载备份</el-button>
            </div>
          </div>
          <div class="g-card">
            <div class="g-card-head">
              <div class="g-card-title"><UploadCloud :size="16" />恢复配置</div>
            </div>
            <div class="g-card-body">
              <p class="desc">用备份 zip <b>覆盖当前全部配置</b>，原配置会先保存到配置目录下的 backup-时间 文件夹，恢复后 Gately 自动重启。</p>
              <div class="restore-row">
                <label class="file-btn">
                  <FileArchive :size="14" />
                  <span>{{ restoreFile ? restoreFile.name : '选择备份文件' }}</span>
                  <input type="file" accept=".zip" @change="onRestorePick" />
                </label>
                <el-button type="danger" :icon="ArchiveRestore" :disabled="!restoreFile" :loading="restoring" @click="confirmRestore">恢复</el-button>
              </div>
            </div>
          </div>
        </div>
      </el-tab-pane>

    </el-tabs>

    <RestartOverlay ref="overlay" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import QRCode from 'qrcode'
import { ElMessage, ElMessageBox, type UploadUserFile } from 'element-plus'
import {
  ArchiveRestore,
  CircleCheck,
  Code2,
  Copy,
  Cpu,
  DoorClosed,
  Download,
  DownloadCloud,
  Eraser,
  Eye,
  FileArchive,
  FileText,
  Gauge,
  Globe,
  Import,
  KeyRound,
  LockKeyhole,
  Network,
  RotateCw,
  Settings2,
  ShieldCheck,
  Smartphone,
  Sparkles,
  UploadCloud,
  UserRound,
} from 'lucide-vue-next'
import { api, type ImportFileResult, type SystemInfo, type SystemSettings } from '../api'
import PageHeader from '../components/PageHeader.vue'
import StatusBadge from '../components/StatusBadge.vue'
import SaveBar from '../components/SaveBar.vue'
import RestartOverlay from '../components/RestartOverlay.vue'
import { useFormLayout } from '../composables/useIsMobile'

const route = useRoute()
const router = useRouter()
const { labelPosition } = useFormLayout()
const mask = '********'
const tabs = ['account', 'access', 'runtime', 'backup']
const tab = ref(typeof route.query.tab === 'string' && tabs.includes(route.query.tab) ? route.query.tab : 'account')
const overlay = ref<InstanceType<typeof RestartOverlay>>()

// “关于”已独立成页，兼容旧链接
if (route.query.tab === 'about') {
  router.replace('/about')
}

// 标签页与地址栏同步，便于从用户菜单直接跳转
watch(tab, (t) => {
  router.replace({ query: { ...route.query, tab: t } })
})

// ---------- 全局设置 ----------
const form = ref<SystemSettings>()
const savedJSON = ref('')
const saving = ref(false)
const dirty = computed(() => !!form.value && JSON.stringify(normalized(form.value)) !== savedJSON.value)

function normalized(s: SystemSettings) {
  return { ...s, dnsServers: s.dnsServers ?? [], restartRequired: false }
}

const dnsServers = computed({
  get: () => form.value?.dnsServers ?? [],
  set: (v: string[]) => {
    if (form.value) {
      form.value.dnsServers = v.map((s) => s.trim()).filter(Boolean)
    }
  },
})

const entryPath = computed(() => {
  const p = (form.value?.safePath ?? '').trim()
  if (!p) {
    return ''
  }
  return p.startsWith('/') ? p : '/' + p
})

function applySettings(s: SystemSettings) {
  form.value = { ...s, dnsServers: s.dnsServers ?? [] }
  savedJSON.value = JSON.stringify(normalized(form.value))
}

async function loadSettings() {
  applySettings(await api.get<SystemSettings>('/api/system/settings'))
}

async function saveSettings() {
  if (!form.value) {
    return
  }
  saving.value = true
  try {
    const r = await api.put<SystemSettings>('/api/system/settings', form.value)
    applySettings(r)
    ElMessage.success(r.restartRequired ? '已保存，部分设置需重启后生效' : '已保存并生效')
    loadInfo()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

function resetSettings() {
  if (savedJSON.value) {
    form.value = { ...JSON.parse(savedJSON.value), restartRequired: form.value?.restartRequired ?? false }
  }
}

async function genToken() {
  try {
    const r = await api.post<{ token: string }>('/api/system/token')
    if (form.value) {
      form.value.apiToken = r.token
    }
    ElMessage.success('已生成新令牌，复制后点击保存生效')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

function randomEntry() {
  if (form.value) {
    form.value.safePath = '/' + Math.random().toString(36).slice(2, 10)
  }
}

async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch {
    // 非 HTTPS 环境下剪贴板 API 不可用，退回到选择复制
    const ta = document.createElement('textarea')
    ta.value = text
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    ta.remove()
    ElMessage.success('已复制')
  }
}

const dnsPresets = [
  { value: '223.5.5.5', label: '阿里' },
  { value: '119.29.29.29', label: '腾讯' },
  { value: '180.76.76.76', label: '百度' },
  { value: '114.114.114.114', label: '114' },
  { value: '1.1.1.1', label: 'Cloudflare' },
  { value: '8.8.8.8', label: 'Google' },
  { value: '9.9.9.9', label: 'Quad9' },
]

const timeZones = [
  { value: 'Asia/Shanghai', label: '中国' },
  { value: 'Asia/Hong_Kong', label: '香港' },
  { value: 'Asia/Taipei', label: '台北' },
  { value: 'Asia/Tokyo', label: '东京' },
  { value: 'Asia/Seoul', label: '首尔' },
  { value: 'Asia/Singapore', label: '新加坡' },
  { value: 'Europe/London', label: '伦敦' },
  { value: 'Europe/Berlin', label: '柏林' },
  { value: 'Europe/Moscow', label: '莫斯科' },
  { value: 'America/New_York', label: '纽约' },
  { value: 'America/Chicago', label: '芝加哥' },
  { value: 'America/Los_Angeles', label: '洛杉矶' },
  { value: 'Australia/Sydney', label: '悉尼' },
  { value: 'UTC', label: '协调世界时' },
]

// ---------- 账号 ----------
const account = reactive({ oldPassword: '', username: '', newPassword: '', confirm: '' })
const accountSaving = ref(false)

async function saveAccount() {
  if (account.newPassword !== account.confirm) {
    ElMessage.error('两次输入的新密码不一致')
    return
  }
  accountSaving.value = true
  try {
    await api.put('/api/account', { oldPassword: account.oldPassword, username: account.username, newPassword: account.newPassword })
    ElMessage.success('已修改，请重新登录')
    router.replace({ name: 'login' })
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    accountSaving.value = false
  }
}

// ---------- 两步验证 ----------
const totp = reactive({ enabled: false, secret: '', qr: '', code: '', password: '' })

async function loadTOTP() {
  const r = await api.get<{ enabled: boolean }>('/api/account/totp')
  totp.enabled = r.enabled
}

async function setupTOTP() {
  try {
    const r = await api.post<{ secret: string; url: string }>('/api/account/totp/setup')
    totp.secret = r.secret
    totp.qr = await QRCode.toDataURL(r.url, { width: 180, margin: 1 })
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

function cancelTOTP() {
  Object.assign(totp, { secret: '', qr: '', code: '', password: '' })
}

async function enableTOTP() {
  try {
    await api.post('/api/account/totp/enable', { password: totp.password, code: totp.code })
    Object.assign(totp, { enabled: true, secret: '', qr: '', code: '', password: '' })
    ElMessage.success('两步验证已启用')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function disableTOTP() {
  try {
    // 关闭时后端要求同时校验密码与当前验证码
    await api.post('/api/account/totp/disable', { password: totp.password, code: totp.code })
    Object.assign(totp, { enabled: false, password: '', code: '' })
    ElMessage.success('两步验证已关闭')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

// ---------- Lucky 导入 ----------
const files = ref<UploadUserFile[]>([])
const results = ref<ImportFileResult[]>([])
const keepEnable = ref(false)
const busy = ref(false)
const importing = ref(false)
const previewed = ref(false)
const imported = ref(false)

// 文件列表或启用选项变化后需要重新预览
watch([files, keepEnable], () => {
  previewed.value = false
  imported.value = false
  results.value = []
})

async function runImport(dryRun: boolean) {
  const fd = new FormData()
  for (const f of files.value) {
    if (f.raw) {
      fd.append('files', f.raw, f.name)
    }
  }
  const q = new URLSearchParams()
  if (dryRun) {
    q.set('dryRun', '1')
  }
  if (keepEnable.value) {
    q.set('enable', '1')
  }
  busy.value = true
  importing.value = !dryRun
  try {
    const qs = q.toString()
    const r = await api.post<ImportFileResult[]>(`/api/import/lucky${qs ? '?' + qs : ''}`, fd)
    results.value = r
    previewed.value = true
    imported.value = !dryRun
    if (!dryRun) {
      const total = r.reduce((n, f) => n + f.reports.reduce((m, x) => m + x.imported, 0), 0)
      ElMessage.success(`导入完成，共 ${total} 条`)
    }
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    busy.value = false
    importing.value = false
  }
}

async function confirmImport() {
  const total = results.value.reduce((n, f) => n + f.reports.reduce((m, x) => m + x.imported, 0), 0)
  try {
    await ElMessageBox.confirm(
      keepEnable.value
        ? `将导入 ${total} 条配置，并保持 Lucky 中的启用状态：已启用的规则会立即生效。确定继续？`
        : `将导入 ${total} 条配置，导入后全部为停用状态。确定继续？`,
      '确认导入',
      { type: keepEnable.value ? 'warning' : 'info', confirmButtonText: '导入', cancelButtonText: '取消' },
    )
  } catch {
    return
  }
  await runImport(false)
}

// ---------- 备份与恢复 ----------
const downloading = ref(false)
const restoreFile = ref<File>()
const restoring = ref(false)

async function downloadBackup() {
  downloading.value = true
  try {
    const resp = await fetch('/api/system/backup', { credentials: 'same-origin' })
    if (!resp.ok) {
      const t = await resp.text()
      let msg = `下载失败 (${resp.status})`
      try {
        msg = JSON.parse(t).error ?? msg
      } catch {
        // 非 JSON 错误响应保持默认提示
      }
      throw new Error(msg)
    }
    const blob = await resp.blob()
    const cd = resp.headers.get('Content-Disposition') ?? ''
    const m = /filename\*?=(?:UTF-8'')?"?([^";]+)"?/i.exec(cd)
    const name = m ? decodeURIComponent(m[1]) : `gately-backup-${new Date().toISOString().slice(0, 10)}.zip`
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = name
    document.body.appendChild(a)
    a.click()
    a.remove()
    window.setTimeout(() => URL.revokeObjectURL(url), 1000)
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    downloading.value = false
  }
}

function onRestorePick(e: Event) {
  restoreFile.value = (e.target as HTMLInputElement).files?.[0]
}

async function confirmRestore() {
  if (!restoreFile.value) {
    return
  }
  try {
    await ElMessageBox.prompt(
      `将用「${restoreFile.value.name}」覆盖当前全部配置并自动重启，当前配置会先备份到配置目录。请输入「恢复」两个字确认。`,
      '恢复配置',
      {
        type: 'warning',
        confirmButtonText: '恢复并重启',
        cancelButtonText: '取消',
        confirmButtonClass: 'el-button--danger',
        inputPlaceholder: '恢复',
        inputValidator: (v: string) => v === '恢复' || '请输入「恢复」',
      },
    )
  } catch {
    return
  }
  const fd = new FormData()
  fd.append('file', restoreFile.value, restoreFile.value.name)
  restoring.value = true
  try {
    const r = await api.post<{ restored: number; previous: string }>('/api/system/restore', fd)
    ElMessage.success(`已恢复 ${r.restored} 个配置文件，原配置保存在 ${r.previous}`)
    overlay.value?.start('正在应用备份并重启…')
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    restoring.value = false
  }
}

// ---------- 系统信息 / 重启 ----------
const info = ref<SystemInfo>()

async function loadInfo() {
  try {
    info.value = await api.get<SystemInfo>('/api/system/info')
  } catch {
    // 忽略，相关提示显示为“-”
  }
}

async function confirmRestart() {
  try {
    await ElMessageBox.confirm('重启期间管理界面与所有转发、代理服务会短暂中断（通常数秒）。确定重启 Gately？', '重启 Gately', {
      type: 'warning',
      confirmButtonText: '重启',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  try {
    await api.post('/api/system/restart')
    overlay.value?.start('正在重启 Gately…')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(async () => {
  loadTOTP()
  loadInfo()
  loadSettings()
  try {
    const s = await api.get<{ username: string }>('/api/session')
    account.username = s.username
  } catch {
    // 未登录时 api 已跳转
  }
})
</script>

<style scoped>
.restart-banner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  margin-bottom: 12px;
  border-radius: var(--g-radius);
  border: 1px solid var(--el-color-warning-light-5);
  background: var(--el-color-warning-light-9);
}
.rb-icon {
  color: var(--el-color-warning);
  flex-shrink: 0;
}
.rb-text {
  flex: 1;
  min-width: 0;
}
.rb-title {
  font-weight: 600;
}
.rb-desc {
  font-size: 12px;
  color: var(--el-text-color-regular);
}
.settings-tabs :deep(.el-tabs__header) {
  margin-bottom: 16px;
}
.tab-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.grid-2 {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  align-items: start;
}
.grid-2 + .g-card,
.g-card + .grid-2 {
  margin-top: 16px;
}
.grid-2 .g-card + .g-card {
  margin-top: 0;
}
.narrow {
  max-width: 460px;
}
.desc {
  margin: 0 0 14px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--el-text-color-regular);
}
.code {
  padding: 1px 5px;
  border-radius: 4px;
  background: var(--el-fill-color-light);
  font-size: 12px;
  word-break: break-all;
}
.qr-wrap {
  display: flex;
  gap: 16px;
  align-items: center;
  flex-wrap: wrap;
  margin-bottom: 14px;
}
.qr {
  display: block;
  border-radius: 8px;
  border: 1px solid var(--g-border);
  background: #fff;
  padding: 6px;
}
.secret-box {
  min-width: 0;
  flex: 1;
}
.secret {
  font-size: 13px;
  word-break: break-all;
  margin: 4px 0;
}
.token-row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.token-input {
  flex: 1;
  min-width: 240px;
}
.token-ops {
  display: flex;
  gap: 8px;
}
.token-ops .el-button + .el-button {
  margin-left: 0;
}
.w-280 {
  width: 280px;
}
.w-480 {
  width: 480px;
}
.upload :deep(.el-upload-dragger) {
  padding: 22px 16px;
  border-radius: 10px;
}
.upload-icon {
  color: var(--el-color-primary);
}
.upload-text {
  margin-top: 6px;
  font-size: 14px;
  color: var(--el-text-color-regular);
}
.upload-text em {
  color: var(--el-color-primary);
  font-style: normal;
}
.keep-enable {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  margin-top: 14px;
  padding: 12px 14px;
  border-radius: 8px;
  border: 1px solid var(--g-border);
  background: var(--el-fill-color-lighter);
}
.ke-title {
  font-weight: 500;
}
.ke-desc {
  margin-top: 2px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}
.ke-desc b {
  color: var(--el-color-warning);
  font-weight: 600;
}
.ops {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 10px;
  margin: 14px 0 4px;
}
.ops .el-button + .el-button {
  margin-left: 0;
}
.ops-done {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
}
.result {
  margin-top: 16px;
}
.result-file {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 6px;
  word-break: break-all;
}
.result-table {
  border: 1px solid var(--g-border);
  border-radius: 8px;
}
.warns {
  margin: 0;
  padding: 4px 16px 4px 36px;
  font-size: 12px;
  line-height: 1.7;
  color: var(--el-color-warning-dark-2, var(--el-color-warning));
}
div.warns {
  padding-left: 36px;
}
.warn-count {
  display: inline-block;
  min-width: 20px;
  padding: 0 6px;
  border-radius: 10px;
  background: var(--el-color-warning-light-9);
  color: var(--el-color-warning);
  font-weight: 600;
  text-align: center;
}
.restore-row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.file-btn {
  flex: 1;
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 12px;
  border: 1px dashed var(--el-border-color);
  border-radius: 6px;
  font-size: 13px;
  color: var(--el-text-color-regular);
  cursor: pointer;
  overflow: hidden;
}
.file-btn span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.file-btn:hover {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}
.file-btn input {
  display: none;
}
.restart-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
  font-size: 13px;
}
@media (max-width: 1000px) {
  .grid-2 {
    grid-template-columns: 1fr;
  }
}
@media (max-width: 767.98px) {
  .restart-banner {
    flex-wrap: wrap;
  }
  .restart-banner .el-button {
    width: 100%;
  }
  .w-280,
  .w-480 {
    width: 100%;
  }
  .token-input {
    min-width: 0;
    flex-basis: 100%;
  }
  .token-ops {
    width: 100%;
  }
  .token-ops .el-button {
    flex: 1;
  }
  .settings-tabs :deep(.el-tabs__item) {
    padding: 0 12px;
  }
}
</style>
