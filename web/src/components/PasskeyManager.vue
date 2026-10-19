<template>
  <div class="g-card">
    <div class="g-card-head">
      <div class="g-card-title"><Fingerprint :size="16" />{{ t('passkey.title') }}</div>
      <StatusBadge :type="list.length ? 'ok' : 'off'" :text="list.length ? t('passkey.count', { count: list.length }) : t('settings.notEnabled')" />
    </div>
    <div class="g-card-body">
      <p class="desc">{{ t('passkey.desc') }}</p>
      <el-alert v-if="!supported" type="warning" :closable="false" show-icon :title="t('passkey.unsupported')" class="gap" />
      <el-table v-if="list.length" :data="list" size="small" class="gap">
        <el-table-column :label="t('common.name')" prop="name" min-width="140" />
        <el-table-column :label="t('passkey.createdAt')" min-width="160">
          <template #default="{ row }">{{ new Date(row.createdAt).toLocaleString() }}</template>
        </el-table-column>
        <el-table-column :label="t('passkey.lastUsed')" min-width="160">
          <template #default="{ row }">{{ row.lastUsed ? new Date(row.lastUsed).toLocaleString() : '-' }}</template>
        </el-table-column>
        <el-table-column :label="t('common.actions')" width="90" align="right">
          <template #default="{ row }">
            <el-button text type="danger" :icon="Trash2" :aria-label="t('common.delete')" @click="remove(row)" />
          </template>
        </el-table-column>
      </el-table>
      <el-button type="primary" plain :icon="Plus" :disabled="!supported" @click="openAdd">{{ t('passkey.add') }}</el-button>
    </div>

    <el-dialog v-model="dialog" :title="t('passkey.add')" width="420px" append-to-body>
      <el-form label-position="top" @submit.prevent="register">
        <el-form-item :label="t('passkey.deviceName')">
          <el-input v-model="name" :placeholder="t('passkey.deviceNamePlaceholder')" maxlength="40" />
        </el-form-item>
        <el-form-item :label="t('settings.currentPassword')" required>
          <el-input v-model="password" type="password" show-password autocomplete="current-password" />
        </el-form-item>
        <p class="desc">{{ t('passkey.addHint') }}</p>
      </el-form>
      <template #footer>
        <el-button @click="dialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="busy" @click="register">{{ t('passkey.continue') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Fingerprint, Plus, Trash2 } from 'lucide-vue-next'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import StatusBadge from './StatusBadge.vue'
import { createPasskey, passkeySupported, type CreateOptions } from '../composables/passkey'

interface PasskeyItem {
  id: string
  name: string
  createdAt: string
  lastUsed?: string
}

const { t } = useI18n()
const supported = passkeySupported()
const list = ref<PasskeyItem[]>([])
const dialog = ref(false)
const name = ref('')
const password = ref('')
const busy = ref(false)

async function load() {
  try {
    list.value = await api.get<PasskeyItem[]>('/api/account/passkeys')
  } catch {
    // 只读账号无权访问，保持空列表
  }
}

function openAdd() {
  name.value = ''
  password.value = ''
  dialog.value = true
}

async function register() {
  if (!password.value) {
    ElMessage.warning(t('passkey.needPassword'))
    return
  }
  busy.value = true
  try {
    const opts = await api.post<CreateOptions>('/api/account/passkeys/begin', {})
    const credential = await createPasskey(opts)
    list.value = await api.post<PasskeyItem[]>('/api/account/passkeys', {
      name: name.value.trim(),
      password: password.value,
      challenge: opts.challenge,
      credential,
    })
    dialog.value = false
    ElMessage.success(t('passkey.added'))
  } catch (e) {
    // 用户在系统弹窗中取消时浏览器抛出 NotAllowedError，不作为错误提示
    if ((e as Error).name !== 'NotAllowedError') {
      ElMessage.error((e as Error).message)
    }
  } finally {
    busy.value = false
  }
}

async function remove(row: PasskeyItem) {
  let pwd = ''
  try {
    const r = await ElMessageBox.prompt(t('passkey.deleteConfirm', { name: row.name }), t('passkey.deleteTitle'), {
      type: 'warning',
      inputType: 'password',
      inputPlaceholder: t('settings.currentPassword'),
      confirmButtonText: t('common.delete'),
      cancelButtonText: t('common.cancel'),
    })
    pwd = r.value
  } catch {
    return
  }
  try {
    list.value = await api.del<PasskeyItem[]>(`/api/account/passkeys/${encodeURIComponent(row.id)}`, { password: pwd })
    ElMessage.success(t('common.deleted'))
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(load)
</script>

<style scoped>
.desc {
  margin: 0 0 12px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.6;
}
.gap {
  margin-bottom: 12px;
}
</style>
