<template>
  <div>
    <el-row :gutter="16">
      <el-col :xs="24" :md="10">
        <el-card shadow="never" header="管理员账号">
          <el-form :model="account" label-width="90px">
            <el-form-item label="原密码"><el-input v-model="account.oldPassword" type="password" show-password /></el-form-item>
            <el-form-item label="新用户名"><el-input v-model="account.username" /></el-form-item>
            <el-form-item label="新密码"><el-input v-model="account.newPassword" type="password" show-password placeholder="至少 8 位" /></el-form-item>
            <el-button type="primary" @click="saveAccount">修改</el-button>
          </el-form>
        </el-card>
        <el-card shadow="never" header="两步验证" class="totp">
          <template v-if="totp.enabled">
            <el-alert type="success" title="已启用：登录时需要输入验证器中的 6 位数字" :closable="false" />
            <el-form label-width="90px" class="totp-form">
              <el-form-item label="当前密码"><el-input v-model="totp.password" type="password" show-password /></el-form-item>
              <el-button type="danger" plain @click="disableTOTP">关闭两步验证</el-button>
            </el-form>
          </template>
          <template v-else-if="totp.secret">
            <p class="desc">用 Google Authenticator、Microsoft Authenticator 等 App 扫描二维码，然后输入显示的验证码确认。</p>
            <img v-if="totp.qr" :src="totp.qr" class="qr" alt="二维码" />
            <div class="mono secret">{{ totp.secret }}</div>
            <el-form label-width="90px" class="totp-form">
              <el-form-item label="验证码"><el-input v-model="totp.code" maxlength="6" /></el-form-item>
              <el-form-item label="当前密码"><el-input v-model="totp.password" type="password" show-password /></el-form-item>
              <el-button type="primary" @click="enableTOTP">确认启用</el-button>
            </el-form>
          </template>
          <template v-else>
            <p class="desc">启用后登录需要额外输入验证器 App 生成的动态验证码。</p>
            <el-button @click="setupTOTP">开始设置</el-button>
          </template>
        </el-card>
      </el-col>
      <el-col :xs="24" :md="14">
        <el-card shadow="never" header="从 Lucky 导入">
          <p class="desc">
            选择 Lucky 配置目录（Docker 默认 <span class="mono">/app/conf</span>）中的 <span class="mono">lucky_*.lkcf</span>
            文件，可多选。先预览，确认后再导入；导入的规则追加到现有规则之后。
          </p>
          <el-upload v-model:file-list="files" :auto-upload="false" multiple accept=".lkcf,.json" drag>
            <div>拖入文件或点击选择</div>
          </el-upload>
          <div class="ops">
            <el-button :disabled="!files.length" :loading="busy" @click="runImport(true)">预览</el-button>
            <el-button type="primary" :disabled="!files.length || !previewed" :loading="busy" @click="runImport(false)">导入</el-button>
          </div>
          <div v-for="r in results" :key="r.file" class="result">
            <div class="file mono">{{ r.file }}</div>
            <el-alert v-if="r.error" type="error" :title="r.error" :closable="false" />
            <el-alert v-else-if="!r.reports.length" type="info" title="未包含 Gately 已支持的模块" :closable="false" />
            <div v-for="rep in r.reports" :key="rep.module">
              <el-alert
                :type="rep.warnings?.length ? 'warning' : 'success'"
                :title="`${rep.title}：共 ${rep.found} 条，${previewed && !imported ? '可导入' : '已导入'} ${rep.imported} 条`"
                :closable="false"
              >
                <div v-for="w in rep.warnings" :key="w">{{ w }}</div>
              </el-alert>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import QRCode from 'qrcode'
import { ElMessage, type UploadUserFile } from 'element-plus'
import { useRouter } from 'vue-router'
import { api, type ImportFileResult } from '../api'

const router = useRouter()
const totp = reactive({ enabled: false, secret: '', qr: '', code: '', password: '' })

async function loadTOTP() {
  const r = await api.get<{ enabled: boolean }>('/api/account/totp')
  totp.enabled = r.enabled
}

async function setupTOTP() {
  const r = await api.post<{ secret: string; url: string }>('/api/account/totp/setup')
  totp.secret = r.secret
  totp.qr = await QRCode.toDataURL(r.url, { width: 180, margin: 1 })
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
    await api.post('/api/account/totp/disable', { password: totp.password, code: '' })
    Object.assign(totp, { enabled: false, password: '' })
    ElMessage.success('两步验证已关闭')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(loadTOTP)
const account = reactive({ oldPassword: '', username: 'admin', newPassword: '' })
const files = ref<UploadUserFile[]>([])
const results = ref<ImportFileResult[]>([])
const busy = ref(false)
const previewed = ref(false)
const imported = ref(false)

// 文件列表变化后需要重新预览
watch(files, () => {
  previewed.value = false
  imported.value = false
  results.value = []
})

async function saveAccount() {
  try {
    await api.put('/api/account', account)
    ElMessage.success('已修改，请重新登录')
    router.replace({ name: 'login' })
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function runImport(dryRun: boolean) {
  const form = new FormData()
  for (const f of files.value) {
    if (f.raw) {
      form.append('files', f.raw, f.name)
    }
  }
  busy.value = true
  try {
    results.value = await api.post<ImportFileResult[]>(`/api/import/lucky${dryRun ? '?dryRun=1' : ''}`, form)
    previewed.value = true
    imported.value = !dryRun
    if (!dryRun) {
      ElMessage.success('导入完成')
    }
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.desc {
  margin-top: 0;
  font-size: 13px;
  color: var(--el-text-color-regular);
  line-height: 1.7;
}
.ops {
  margin: 12px 0;
}
.result {
  margin-top: 10px;
}
.file {
  font-size: 12px;
  margin-bottom: 4px;
  color: var(--el-text-color-secondary);
}
.totp {
  margin-top: 16px;
}
.totp-form {
  margin-top: 12px;
}
.qr {
  display: block;
  margin: 8px 0;
}
.secret {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}
.result .el-alert {
  margin-bottom: 6px;
}
</style>
