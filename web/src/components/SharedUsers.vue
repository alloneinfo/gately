<template>
  <div class="shared-users">
    <div v-for="(u, i) in model" :key="i" class="user">
      <div class="user-row">
        <el-input v-model="u.name" :placeholder="t('sharedUsers.namePlaceholder')" class="w-160" />
        <el-input
          v-model="u.password"
          type="password"
          show-password
          autocomplete="new-password"
          :placeholder="u.hasPassword ? t('sharedUsers.passwordKeep') : t('sharedUsers.passwordPlaceholder')"
        />
        <el-button text type="danger" :icon="Trash2" :aria-label="t('common.delete')" @click="model.splice(i, 1)" />
      </div>
      <div class="user-row">
        <el-tag v-if="u.totpSecret" size="small" type="success">{{ t('sharedUsers.totpOn') }}</el-tag>
        <el-tag v-else size="small" type="info">{{ t('sharedUsers.totpOff') }}</el-tag>
        <el-button size="small" :loading="loadingIndex === i" @click="setupTOTP(u, i)">
          {{ u.totpSecret ? t('sharedUsers.totpReset') : t('sharedUsers.totpEnable') }}
        </el-button>
        <el-button v-if="u.totpSecret" size="small" :loading="showingIndex === i" @click="showTOTP(u, i)">{{ t('sharedUsers.totpShow') }}</el-button>
        <el-button v-if="u.totpSecret" size="small" text type="danger" @click="u.totpSecret = ''">{{ t('sharedUsers.totpDisable') }}</el-button>
        <el-input v-model="u.remark" :placeholder="t('common.remark')" class="remark" />
      </div>
      <!-- 可选的身份信息：OIDC 身份提供方把它们作为 email、name、groups 声明提供给应用 -->
      <div class="user-row profile">
        <el-input v-model="u.email" :placeholder="t('sharedUsers.emailPlaceholder')" class="half" />
        <el-input v-model="u.displayName" :placeholder="t('sharedUsers.displayNamePlaceholder')" class="half" />
      </div>
      <div class="user-row">
        <el-select
          v-model="u.groups"
          multiple
          filterable
          allow-create
          default-first-option
          :reserve-keyword="false"
          :placeholder="t('sharedUsers.groupsPlaceholder')"
          class="groups"
        >
          <el-option v-for="g in allGroups" :key="g" :label="g" :value="g" />
        </el-select>
      </div>
    </div>
    <el-button size="small" :icon="Plus" @click="add">{{ t('sharedUsers.add') }}</el-button>

    <el-dialog v-model="qrVisible" :title="t('sharedUsers.qrTitle', { name: qrName })" width="360px" append-to-body>
      <div class="qr-wrap">
        <img v-if="qr" :src="qr" class="qr" :alt="t('sharedUsers.qrTitle', { name: qrName })" />
        <div class="mono secret">{{ qrSecret }}</div>
        <div class="form-help">{{ t('sharedUsers.qrHelp') }}</div>
      </div>
      <template #footer>
        <el-button type="primary" @click="qrVisible = false">{{ t('common.close') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, toRaw, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Plus, Trash2 } from 'lucide-vue-next'
import QRCode from 'qrcode'
import { api } from '../api'
import { SECRET_MASK, fetchSecret } from '../secrets'
import type { AuthUser } from '../sharedUsers'

const { t } = useI18n()
const model = defineModel<AuthUser[]>({ required: true })
const qrVisible = ref(false)
const qr = ref('')
const qrSecret = ref('')
const qrName = ref('')
const loadingIndex = ref(-1)
const showingIndex = ref(-1)
// savedNames 记录每个用户载入时的用户名：TOTP 密钥明文表按已保存的用户名索引，界面上改名后仍按原名取回
const savedNames = new WeakMap<AuthUser, string>()
watch(
  model,
  (list) => {
    for (const u of list) {
      if (!savedNames.has(toRaw(u))) {
        savedNames.set(toRaw(u), u.name)
      }
    }
  },
  { immediate: true },
)
// 已用过的分组，作为其他用户的候选项
const allGroups = computed(() => [...new Set(model.value.flatMap((u) => u.groups ?? []))].sort())

function add() {
  model.value.push({ name: '', password: '', totpSecret: '', remark: '', email: '', displayName: '', groups: [] })
}

// totpURL 生成验证器 App 使用的 otpauth 地址，格式与后端 auth.TOTPURL 一致
function totpURL(secret: string, account: string): string {
  const q = new URLSearchParams({ digits: '6', issuer: 'Gately', period: '30', secret })
  return `otpauth://totp/${encodeURIComponent('Gately:' + account)}?${q.toString()}`
}

// showTOTP 重新展示已有密钥的二维码，便于在新设备上添加；已保存的密钥是脱敏占位符，先向后端取回明文
async function showTOTP(u: AuthUser, i: number) {
  showingIndex.value = i
  try {
    let secret: string | undefined = u.totpSecret
    if (secret === SECRET_MASK) {
      secret = await fetchSecret('/api/webservice/settings/secrets/totp', savedNames.get(toRaw(u)) ?? u.name)
    }
    if (!secret) {
      return
    }
    const name = u.name.trim()
    qrSecret.value = secret
    qrName.value = name
    qr.value = await QRCode.toDataURL(totpURL(secret, name), { width: 200, margin: 1 })
    qrVisible.value = true
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    showingIndex.value = -1
  }
}

// setupTOTP 生成新密钥并展示二维码；保存设置后生效，旧密钥同时作废
async function setupTOTP(u: AuthUser, i: number) {
  if (!u.name.trim()) {
    ElMessage.warning(t('sharedUsers.nameFirst'))
    return
  }
  loadingIndex.value = i
  try {
    const r = await api.post<{ secret: string; url: string }>('/api/webservice/totp-secret', { name: u.name.trim() })
    u.totpSecret = r.secret
    qrSecret.value = r.secret
    qrName.value = u.name.trim()
    qr.value = await QRCode.toDataURL(r.url, { width: 200, margin: 1 })
    qrVisible.value = true
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loadingIndex.value = -1
  }
}
</script>

<style scoped>
.shared-users {
  width: 100%;
}
.user {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px;
  margin-bottom: 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}
.user-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.w-160 {
  width: 160px;
  flex: none;
}
.remark,
.groups,
.half {
  flex: 1;
  min-width: 0;
}
@media (max-width: 640px) {
  .user-row.profile {
    flex-wrap: wrap;
  }
  .user-row.profile > * {
    width: 100%;
    flex: 1 1 100%;
  }
}
.qr-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}
.qr {
  width: 200px;
  height: 200px;
}
.secret {
  word-break: break-all;
  text-align: center;
}
</style>
