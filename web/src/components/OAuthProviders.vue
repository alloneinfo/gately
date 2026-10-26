<template>
  <div>
    <div v-for="(p, i) in model" :key="i" class="provider">
      <div class="provider-row">
        <el-select v-model="p.type" class="w-120" @change="onType(p)">
          <el-option v-for="ty in types" :key="ty.value" :label="ty.label" :value="ty.value" />
        </el-select>
        <el-input v-model="p.key" :placeholder="t('oauthProviders.keyPlaceholder')" class="w-120" />
        <el-input v-model="p.name" :placeholder="t('oauthProviders.namePlaceholder')" class="w-120" />
        <el-button text type="danger" :icon="Trash2" :aria-label="t('common.delete')" @click="model.splice(i, 1)" />
      </div>
      <div class="provider-row">
        <el-input v-model="p.clientId" :placeholder="t('oauthProviders.clientIdPlaceholder')" />
        <SecretInput v-model="p.clientSecret" :placeholder="t('oauthProviders.clientSecretPlaceholder')" :reveal="revealOf(p)" />
      </div>
      <div class="form-help">{{ hintOf(p) }}</div>
      <div v-if="p.type === 'oidc'" class="provider-row">
        <el-input v-model="p.issuer" :placeholder="t('oauthProviders.issuerPlaceholder')" />
      </div>
      <div class="provider-row">
        <el-input v-model="p.proxy" clearable :placeholder="t('oauthProviders.proxyPlaceholder')" />
      </div>
      <div class="form-help">{{ t('oauthProviders.proxyHint') }}</div>
    </div>
    <el-button size="small" :icon="Plus" @click="add">{{ t('oauthProviders.addProvider') }}</el-button>
  </div>
</template>

<script setup lang="ts">
import { computed, toRaw, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus, Trash2 } from 'lucide-vue-next'
import type { OAuthProvider } from '../api'
import SecretInput from './SecretInput.vue'
import { fetchSecret } from '../secrets'

const { t } = useI18n()
const model = defineModel<OAuthProvider[]>({ required: true })
// secretsUrl 为查看已保存 Client Secret 明文的接口（返回按登录方式标识索引的密钥表），不提供时密钥框只切换显示
const props = defineProps<{ secretsUrl?: string }>()

// savedKeys 记录每个登录方式载入时的标识：明文表按已保存的标识索引，界面上改了标识后仍按原标识取回
const savedKeys = new WeakMap<OAuthProvider, string>()
watch(
  model,
  (list) => {
    for (const p of list) {
      if (!savedKeys.has(toRaw(p))) {
        savedKeys.set(toRaw(p), p.key)
      }
    }
  },
  { immediate: true },
)

// revealOf 返回取回已保存 Client Secret 明文的函数
function revealOf(p: OAuthProvider): (() => Promise<string | undefined>) | undefined {
  const url = props.secretsUrl
  if (!url) {
    return undefined
  }
  return () => fetchSecret(url, savedKeys.get(toRaw(p)) ?? p.key)
}

const types = computed(() => [
  { value: 'github', label: 'GitHub' },
  { value: 'google', label: 'Google' },
  { value: 'qq', label: 'QQ' },
  { value: 'oidc', label: t('oauthProviders.oidc') },
])
const names: Record<string, string> = { github: 'GitHub', google: 'Google', qq: 'QQ', oidc: '' }
const hintTypes = ['github', 'google', 'qq', 'oidc']

function hintOf(p: OAuthProvider) {
  if (!hintTypes.includes(p.type)) {
    return ''
  }
  return t(`oauthProviders.hints.${p.type}`)
}

// 切换类型时，标识与显示名还是空或仍为上一个类型的默认值，则同步为新类型的默认值
function onType(p: OAuthProvider) {
  const prevDefaults = Object.keys(names)
  if (!p.key || prevDefaults.includes(p.key)) {
    p.key = p.type
  }
  if (!p.name || Object.values(names).includes(p.name)) {
    p.name = names[p.type]
  }
}

function add() {
  const used = new Set(model.value.map((p) => p.key))
  const type = ['github', 'google', 'qq'].find((k) => !used.has(k)) ?? 'oidc'
  model.value.push({ key: type === 'oidc' ? '' : type, name: names[type], type, clientId: '', clientSecret: '', issuer: '', proxy: '' })
}
</script>

<style scoped>
.w-120 {
  width: 120px;
}
.provider {
  border: 1px solid var(--g-border);
  border-radius: 8px;
  padding: 10px;
  margin-bottom: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.provider-row {
  display: flex;
  gap: 8px;
  align-items: center;
}
@media (max-width: 767.98px) {
  .provider-row {
    flex-wrap: wrap;
  }
  .provider-row .el-input,
  .provider-row .el-select {
    width: calc(50% - 4px) !important;
    flex: 1 1 calc(50% - 4px);
  }
}
</style>
